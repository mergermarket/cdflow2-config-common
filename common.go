package common

import (
	"bufio"
	"encoding/json"
	"io"
	"log"
)

// This boilerplate is intended to be generic, copyed to any config container - put any specific logic in handler/handler.go

type message struct {
	Action string
}

// Run handles all the IO, calling into the passed in Handler to do the actual work.
func Run(handler Handler, readStream io.Reader, writeStream, errorStream io.Writer) {
	var version string
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
			version = configureRelease(handler, line, encoder, errorStream)
		case "upload_release":
			uploadRelease(handler, line, encoder, errorStream, version)
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

func configureRelease(handler Handler, line []byte, encoder *json.Encoder, errorStream io.Writer) string {
	var request ConfigureReleaseRequest
	if err := json.Unmarshal(line, &request); err != nil {
		log.Fatalln("error parsing configure release request:", err)
	}
	var response ConfigureReleaseResponse
	response.Env = make(map[string]string)
	if err := handler.ConfigureRelease(&request, &response, errorStream); err != nil {
		log.Fatalln("error in ConfigureRelease:", err)
	}
	if err := encoder.Encode(response); err != nil {
		log.Fatalln("error encoding configure release response:", err)
	}
	return request.Version
}

func uploadRelease(handler Handler, line []byte, encoder *json.Encoder, errorStream io.Writer, version string) {
	var request UploadReleaseRequest
	if err := json.Unmarshal(line, &request); err != nil {
		log.Fatalln("error parsing upload release request:", err)
	}
	var response UploadReleaseResponse
	if err := handler.UploadRelease(&request, &response, errorStream, version); err != nil {
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
	var response PrepareTerraformResponse
	if err := handler.PrepareTerraform(&request, &response, errorStream); err != nil {
		log.Fatalln("error in PrepareTerraform:", err)
	}
	if err := encoder.Encode(response); err != nil {
		log.Fatalln("error encoding prepare terraform response:", err)
	}
}
