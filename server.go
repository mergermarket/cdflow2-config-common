package common

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
)

type message struct {
	Action string
}

func getSigtermChannel() chan os.Signal {
	result := make(chan os.Signal, 1)
	signal.Notify(result, syscall.SIGTERM)
	return result
}

// AcceptResult is the result from an accept.
type AcceptResult struct {
	connection net.Conn
	err        error
}

// Listen accepts connections and forwards requests and responses to and from the handler.
func Listen(handler Handler, socketPath, releaseDir string, sigtermChannel chan os.Signal) {

	if socketPath == "" {
		socketPath = defaultSocketPath
	}

	if sigtermChannel == nil {
		sigtermChannel = getSigtermChannel()
	}

	sockdir := filepath.Dir(socketPath)
	if _, err := os.Stat(sockdir); os.IsNotExist(err) {
		if err := os.MkdirAll(sockdir, os.ModePerm); err != nil {
			log.Panicf("could not create socket dir %q: %v", sockdir, err)
		}
	}

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Panicf("could not listen on unix domain socket %v: %v", socketPath, err)
	}
	defer listener.Close()

	acceptChannel := make(chan AcceptResult)
	go func() {
		for {
			connection, err := listener.Accept()
			acceptChannel <- AcceptResult{connection, err}
		}
	}()

	var configureReleaseRequest *ConfigureReleaseRequest

	for {
		var acceptResult AcceptResult
		var connection net.Conn
		select {
		case <-sigtermChannel:
			return
		case acceptResult = <-acceptChannel:
			if acceptResult.err != nil {
				log.Panicln("error accepting connection:", err)
			}
			connection = acceptResult.connection
		}

		var buffer bytes.Buffer
		_, err = io.Copy(&buffer, connection)
		if err != nil {
			log.Panicf("error reading request from unix domain socket %v: %v", socketPath, err)
		}
		rawRequest := buffer.Bytes()
		var request message
		if err := json.Unmarshal(rawRequest, &request); err != nil {
			log.Panicln("error reading request:", err)
		}
		var response interface{}
		switch request.Action {
		case "setup":
			response = setup(handler, rawRequest)
		case "configure_release":
			response, configureReleaseRequest = configureRelease(handler, rawRequest)
		case "upload_release":
			response = uploadRelease(handler, rawRequest, configureReleaseRequest, releaseDir)
		case "prepare_terraform":
			response = prepareTerraform(handler, rawRequest, releaseDir)
		default:
			log.Panicln("unknown message type:", request.Action)
		}
		if err := json.NewEncoder(connection).Encode(response); err != nil {
			log.Panicln("error encoding response:", err)
		}
		connection.Close()
	}
}

func setup(handler Handler, rawRequest []byte) *SetupResponse {
	var request SetupRequest
	if err := json.Unmarshal(rawRequest, &request); err != nil {
		log.Fatalln("error parsing setup request:", err)
	}
	response := CreateSetupResponse()
	if err := handler.Setup(&request, response); err != nil {
		log.Fatalln("error in Setup:", err)
	}
	return response
}

func configureRelease(handler Handler, rawRequest []byte) (*ConfigureReleaseResponse, *ConfigureReleaseRequest) {
	var request ConfigureReleaseRequest
	if err := json.Unmarshal(rawRequest, &request); err != nil {
		log.Fatalln("error parsing configure release request:", err)
	}
	response := CreateConfigureReleaseResponse()
	if err := handler.ConfigureRelease(&request, response); err != nil {
		log.Fatalln("error in ConfigureRelease:", err)
	}
	return response, &request
}

func uploadRelease(handler Handler, rawRequest []byte, configureReleaseRequest *ConfigureReleaseRequest, releaseDir string) *UploadReleaseResponse {
	var request UploadReleaseRequest
	if err := json.Unmarshal(rawRequest, &request); err != nil {
		log.Fatalln("error parsing upload release request:", err)
	}
	response := CreateUploadReleaseResponse()
	if err := handler.UploadRelease(&request, response, configureReleaseRequest, releaseDir); err != nil {
		log.Fatalln("error in UploadRelease:", err)
	}
	return response
}

func prepareTerraform(handler Handler, rawRequest []byte, releaseDir string) *PrepareTerraformResponse {
	var request PrepareTerraformRequest
	if err := json.Unmarshal(rawRequest, &request); err != nil {
		log.Fatalln("error parsing prepare terraform request:", err)
	}
	response := CreatePrepareTerraformResponse()
	if err := handler.PrepareTerraform(&request, response, releaseDir); err != nil {
		log.Fatalln("error in PrepareTerraform:", err)
	}
	return response
}

// ZipRelease zips the release folder to a stream.
func ZipRelease(writer io.Writer, dir, component, version, terraformImage string) error {
	if component == "" {
		panic("no component")
	}
	prefix := component + "-" + version
	zipWriter := zip.NewWriter(writer)
	terraformImageFilename := filepath.Join(prefix, "terraform-image")
	writer, err := zipWriter.Create(terraformImageFilename)
	if err != nil {
		return err
	}
	writer.Write([]byte(terraformImage))
	if err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		relativePath, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = filepath.Join(prefix, relativePath)

		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}

		reader, err := os.Open(path)
		if err != nil {
			return err
		}
		defer reader.Close()

		_, err = io.Copy(writer, reader)
		if err != nil {
			return err
		}

		return nil
	}); err != nil {
		return err
	}
	return zipWriter.Close()
}

// UnzipRelease unzips the release.
func UnzipRelease(reader io.Reader, dir, component, version string) (string, error) {
	contents, err := ioutil.ReadAll(reader)
	if err != nil {
		return "", err
	}
	zipReader, err := zip.NewReader(bytes.NewReader(contents), int64(len(contents)))
	if err != nil {
		return "", err
	}
	var terraformImageBuffer bytes.Buffer
	for _, file := range zipReader.File {
		if file.FileInfo().IsDir() {
			continue
		}
		if file.Name[0] == '/' {
			return "", fmt.Errorf("error in release zip, unexpected absolute path \"%v\"", file.Name[0])
		}
		prefix := component + "-" + version
		parts := strings.Split(file.Name, "/")
		if parts[0] != prefix {
			return "", fmt.Errorf("error in release zip, expected prefix \"%v\", got \"%v\"", prefix, parts[0])
		}

		reader, err := file.Open()
		if err != nil {
			return "", err
		}
		defer reader.Close()

		if len(parts) == 2 && parts[1] == "terraform-image" {
			io.Copy(&terraformImageBuffer, reader)
			continue
		}

		destFilename := filepath.Join(dir, filepath.Join(parts[1:]...))
		if err = os.MkdirAll(filepath.Dir(destFilename), os.FileMode(0755)); err != nil {
			return "", err
		}
		writer, err := os.OpenFile(destFilename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return "", err
		}
		defer writer.Close()
		if _, err := io.Copy(writer, reader); err != nil {
			return "", err
		}
	}
	if terraformImageBuffer.String() == "" {
		panic("did not find terraform-image in zip")
	}
	return string(terraformImageBuffer.String()), nil
}
