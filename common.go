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
	"time"
)

type message struct {
	Action string
}

const defaultSocketPath = "/sock"

// Run handles all the IO, calling into the passed in Handler to do the actual work.
func Run(handler Handler, args []string, readStream io.Reader, writeStream io.Writer, overrideSocketPath string) {
	socketPath := overrideSocketPath
	if socketPath == "" {
		socketPath = defaultSocketPath
	}
	if len(args) == 1 && args[0] == "forward" {
		forward(readStream, writeStream, socketPath)
	} else {
		Listen(handler, socketPath, getSigtermChannel())
	}
}

func connect(socketPath string) *net.UnixConn {
	var err error
	for i := 0; i < 20; i++ {
		var connection net.Conn
		connection, err = net.Dial("unix", socketPath)
		if err == nil {
			if connection, ok := connection.(*net.UnixConn); ok {
				return connection
			}
			log.Panicf("unexpected type for connection: %T", connection)
		}
		time.Sleep(200 * time.Millisecond)
	}
	log.Panicf("could not connect to unix domain socket %v after four seconds: %v\n", socketPath, err)
	return nil
}

func forward(readStream io.Reader, writeStream io.Writer, socketPath string) {
	connection := connect(socketPath)
	defer connection.Close()
	go func() {
		_, err := io.Copy(connection, readStream)
		if err != nil {
			log.Panicf("error copying to %v: %v", socketPath, err)
		}
		if err := connection.CloseWrite(); err != nil {
			log.Panicln("error closing socket:", err)
		}
	}()
	if _, err := io.Copy(writeStream, connection); err != nil {
		log.Panicf("error copying from %v: %v", socketPath, err)
	}
}

func getSigtermChannel() chan os.Signal {
	result := make(chan os.Signal, 1)
	signal.Notify(result, syscall.SIGTERM)
	return result
}

func shouldStop(sigtermChannel chan os.Signal) bool {
	select {
	case <-sigtermChannel:
		return true
	default:
		return false
	}
}

// Listen accepts connections and forwards requests and responses to and from the handler.
func Listen(handler Handler, socketPath string, sigtermChannel chan os.Signal) {
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Panicf("could not listen on unix domain socket %v: %v", socketPath, err)
	}
	defer listener.Close()

	var version string
	var config map[string]interface{}

	for !shouldStop(sigtermChannel) {
		connection, err := listener.Accept()
		if err != nil {
			log.Panicf("error accepting connection on unix domain socket %v: %v", socketPath, err)
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
