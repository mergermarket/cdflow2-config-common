package common

import (
	"archive/zip"
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// This boilerplate is intended to be generic, copyed to any config container - put any specific logic in handler/handler.go

type message struct {
	Action string
}

// Run handles all the IO, calling into the passed in Handler to do the actual work.
func Run(handler Handler, readStream io.Reader, writeStream, errorStream io.Writer) {
	var version string
	var config map[string]interface{}
	scanner := bufio.NewScanner(readStream)
	encoder := json.NewEncoder(writeStream)
	// for sending diagnostic info for the tests
	for scanner.Scan() {
		line := scanner.Bytes()
		var msg message
		if err := json.Unmarshal(line, &msg); err != nil {
			log.Fatalln("error reading message:", err)
		}
		switch msg.Action {
		case "configure_release":
			version, config = configureRelease(handler, line, encoder, errorStream)
		case "upload_release":
			uploadRelease(handler, line, encoder, errorStream, version, config)
		case "prepare_terraform":
			prepareTerraform(handler, line, encoder, errorStream)
		case "stop":
			return
		default:
			log.Fatalln("unknown message type:", msg.Action)
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatalln("error reading from stdin:", err)
	}
}

func configureRelease(handler Handler, line []byte, encoder *json.Encoder, errorStream io.Writer) (string, map[string]interface{}) {
	var request ConfigureReleaseRequest
	if err := json.Unmarshal(line, &request); err != nil {
		log.Fatalln("error parsing configure release request:", err)
	}
	response := CreateConfigureReleaseResponse()
	if err := handler.ConfigureRelease(&request, response, errorStream); err != nil {
		log.Fatalln("error in ConfigureRelease:", err)
	}
	if err := encoder.Encode(response); err != nil {
		log.Fatalln("error encoding configure release response:", err)
	}
	return request.Version, request.Config
}

func uploadRelease(handler Handler, line []byte, encoder *json.Encoder, errorStream io.Writer, version string, config map[string]interface{}) {
	var request UploadReleaseRequest
	if err := json.Unmarshal(line, &request); err != nil {
		log.Fatalln("error parsing upload release request:", err)
	}
	// zip up /release folder here
	response := CreateUploadReleaseResponse()
	if err := handler.UploadRelease(&request, response, errorStream, version, config); err != nil {
		log.Fatalln("error in UploadRelease:", err)
	}
	if err := encoder.Encode(response); err != nil {
		log.Fatalln("error encoding upload release response:", err)
	}
}

func prepareTerraform(handler Handler, line []byte, encoder *json.Encoder, errorStream io.Writer) {
	var request PrepareTerraformRequest
	if err := json.Unmarshal(line, &request); err != nil {
		log.Fatalln("error parsing prepare terraform request:", err)
	}
	response := CreatePrepareTerraformResponse()
	if err := handler.PrepareTerraform(&request, response, errorStream); err != nil {
		log.Fatalln("error in PrepareTerraform:", err)
	}
	if err := encoder.Encode(response); err != nil {
		log.Fatalln("error encoding prepare terraform response:", err)
	}
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
