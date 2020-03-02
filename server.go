package common

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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

type AcceptResult struct {
	connection net.Conn
	err        error
}

// Listen accepts connections and forwards requests and responses to and from the handler.
func Listen(handler Handler, socketPath string, sigtermChannel chan os.Signal) {

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

	var version string
	var config map[string]interface{}

	for {
		var acceptResult AcceptResult
		var connection net.Conn
		select {
		case <-sigtermChannel:
			return
		case acceptResult = <-acceptChannel:
			if acceptResult.err != nil {
				log.Panicln("error accepting connetion:", err)
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
		case "configure_release":
			response, version, config = configureRelease(handler, rawRequest)
		case "upload_release":
			response = uploadRelease(handler, rawRequest, version, config)
		case "prepare_terraform":
			response = prepareTerraform(handler, rawRequest)
		default:
			log.Panicln("unknown message type:", request.Action)
		}
		if err := json.NewEncoder(connection).Encode(response); err != nil {
			log.Panicln("error encoding response:", err)
		}
		connection.Close()
	}
}

func configureRelease(handler Handler, rawRequest []byte) (*ConfigureReleaseResponse, string, map[string]interface{}) {
	var request ConfigureReleaseRequest
	if err := json.Unmarshal(rawRequest, &request); err != nil {
		log.Fatalln("error parsing configure release request:", err)
	}
	response := CreateConfigureReleaseResponse()
	if err := handler.ConfigureRelease(&request, response); err != nil {
		log.Fatalln("error in ConfigureRelease:", err)
	}
	return response, request.Version, request.Config
}

func uploadRelease(handler Handler, rawRequest []byte, version string, config map[string]interface{}) *UploadReleaseResponse {
	var request UploadReleaseRequest
	if err := json.Unmarshal(rawRequest, &request); err != nil {
		log.Fatalln("error parsing upload release request:", err)
	}
	// TODO zip up /release folder here
	response := CreateUploadReleaseResponse()
	if err := handler.UploadRelease(&request, response, version, config); err != nil {
		log.Fatalln("error in UploadRelease:", err)
	}
	return response
}

func prepareTerraform(handler Handler, rawRequest []byte) *PrepareTerraformResponse {
	var request PrepareTerraformRequest
	if err := json.Unmarshal(rawRequest, &request); err != nil {
		log.Fatalln("error parsing prepare terraform request:", err)
	}
	response := CreatePrepareTerraformResponse()
	if err := handler.PrepareTerraform(&request, response); err != nil {
		log.Fatalln("error in PrepareTerraform:", err)
	}
	return response
}

// CreateConfigureReleaseRequest creates and returns an initialised ConfigureReleaseRequest - useful for testing config containers.
func CreateConfigureReleaseRequest() *ConfigureReleaseRequest {
	var request ConfigureReleaseRequest
	request.Env = make(map[string]string)
	request.Config = make(map[string]interface{})
	return &request
}

// CreateConfigureReleaseResponse creates and returns an initialised ConfigureReleaseResponse.
func CreateConfigureReleaseResponse() *ConfigureReleaseResponse {
	var response ConfigureReleaseResponse
	response.Env = make(map[string]string)
	response.Success = true
	return &response
}

// CreateUploadReleaseRequest creates and returns an initialised UploadReleaseRequest - useful for testing config containers.
func CreateUploadReleaseRequest() *UploadReleaseRequest {
	var request UploadReleaseRequest
	request.ReleaseMetadata = make(map[string]map[string]string)
	return &request
}

// CreateUploadReleaseResponse creates and returns an initialised UploadReleaseResponse.
func CreateUploadReleaseResponse() *UploadReleaseResponse {
	var response UploadReleaseResponse
	response.Success = true
	return &response
}

// CreatePrepareTerraformRequest creates and returns an initialised PrepareTerraformRequest - useful for testing config containers.
func CreatePrepareTerraformRequest() *PrepareTerraformRequest {
	var request PrepareTerraformRequest
	request.Config = make(map[string]interface{})
	request.Env = make(map[string]string)
	return &request
}

// CreatePrepareTerraformResponse creates and returns an initialised PrepareTerraformResponse.
func CreatePrepareTerraformResponse() *PrepareTerraformResponse {
	var response PrepareTerraformResponse
	response.Env = make(map[string]string)
	response.TerraformBackendConfig = make(map[string]string)
	response.Success = true
	return &response
}

// ZipRelease zips the release folder to a stream.
func ZipRelease(writer io.Writer, dir, component, version string) error {
	zipWriter := zip.NewWriter(writer)
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		relativePath, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}

		writer, err := zipWriter.Create(filepath.Join(component+"-"+version, relativePath))
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
	})
	if err != nil {
		return err
	}
	return zipWriter.Close()
}

// UnzipRelease unzips the release.
func UnzipRelease(reader io.ReaderAt, size int64, dir, component, version string) error {
	zipReader, err := zip.NewReader(reader, size)
	if err != nil {
		return nil
	}
	for _, file := range zipReader.File {
		if file.FileInfo().IsDir() {
			continue
		}
		if file.Name[0] == '/' {
			return fmt.Errorf("error in release zip, unexpected absolute path \"%v\"", file.Name[0])
		}
		prefix := component + "-" + version
		parts := strings.Split(file.Name, "/")
		if parts[0] != prefix {
			return fmt.Errorf("error in release zip, expected prefix \"%v\", got \"%v\"", prefix, parts[0])
		}
		destFilename := filepath.Join(dir, filepath.Join(parts[1:]...))
		reader, err := file.Open()
		if err != nil {
			return err
		}
		defer reader.Close()

		if err = os.MkdirAll(filepath.Dir(destFilename), os.FileMode(0755)); err != nil {
			return err
		}
		writer, err := os.Create(destFilename)
		if err != nil {
			return err
		}
		defer writer.Close()
		if _, err := io.Copy(writer, reader); err != nil {
			return err
		}
	}
	return nil
}
