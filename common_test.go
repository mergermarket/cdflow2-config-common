package common_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"testing"

	common "github.com/mergermarket/cdflow2-config-common"
)

type handler struct{}

func (*handler) ConfigureRelease(request *common.ConfigureReleaseRequest, response *common.ConfigureReleaseResponse, errorStream io.Writer) error {
	fmt.Fprintf(errorStream, "version: %v, env key: %v, config key: %v\n", request.Version, request.Env["env-key"], request.Config["config-key"])
	response.Env["response-env-key"] = "response-env-value"
	return nil
}

func (*handler) UploadRelease(request *common.UploadReleaseRequest, response *common.UploadReleaseResponse, errorStream io.Writer, version string) error {
	fmt.Fprintf(errorStream, "terraform image: %v, release metadata value: %v\n", request.TerraformImage, request.ReleaseMetadata["release"]["release-key"])
	response.Message = "test-uploaded-message"
	return nil
}

func (*handler) PrepareTerraform(request *common.PrepareTerraformRequest, response *common.PrepareTerraformResponse, errorStream io.Writer) error {
	fmt.Fprintf(errorStream, "version: %v, env name: %v, config value: %v, env value: %v\n", request.Version, request.EnvName, request.Config["config-key"], request.Env["env-key"])
	response.Env = map[string]string{
		"response-env-key": "response-env-value",
	}
	response.TerraformImage = "test-terraform-image"
	response.TerraformBackendType = "test-backend-type"
	response.TerraformBackendConfig = map[string]string{
		"backend-key": "backend-value",
	}
	return nil
}
func TestRun(t *testing.T) {
	inputReader, inputWriter := io.Pipe()
	encoder := json.NewEncoder(inputWriter)

	outputReader, outputWriter := io.Pipe()
	scanner := bufio.NewScanner(outputReader)

	var errorBuffer bytes.Buffer

	go common.Run(&handler{}, inputReader, outputWriter, &errorBuffer)

	checkConfigureRelease(encoder, scanner, &errorBuffer)
	checkUploadRelease(encoder, scanner, &errorBuffer)
	checkPrepareTerraform(encoder, scanner, &errorBuffer)

	encoder.Encode(map[string]string{"Action": "stop"})
}

func checkConfigureRelease(encoder *json.Encoder, scanner *bufio.Scanner, errorBuffer *bytes.Buffer) {
	if err := encoder.Encode(map[string]interface{}{
		"Action":  "configure_release",
		"Version": "test-version",
		"Config":  map[string]interface{}{"config-key": "config-value"},
		"Env":     map[string]string{"env-key": "env-value"},
	}); err != nil {
		log.Fatalln("error encoding json:", err)
	}

	readAndCheckOutputLine(scanner, map[string]interface{}{
		"Env": map[string]string{
			"response-env-key": "response-env-value",
		},
	})
	if errorBuffer.String() != "version: test-version, env key: env-value, config key: config-value\n" {
		log.Fatalln("unexpected configure release debug output:", errorBuffer.String())
	}
	errorBuffer.Truncate(0)
}

func checkUploadRelease(encoder *json.Encoder, scanner *bufio.Scanner, errorBuffer *bytes.Buffer) {
	if err := encoder.Encode(map[string]interface{}{
		"Action":         "upload_release",
		"TerraformImage": "test-terraform-image",
		"ReleaseMetadata": map[string]map[string]string{
			"release": map[string]string{
				"release-key": "release-value",
			},
		},
	}); err != nil {
		log.Fatalln("error encoding json:", err)
	}

	readAndCheckOutputLine(scanner, map[string]interface{}{
		"Message": "test-uploaded-message",
	})
	if errorBuffer.String() != "terraform image: test-terraform-image, release metadata value: release-value\n" {
		log.Fatalln("unexpected upload release debug output:", errorBuffer.String())
	}
	errorBuffer.Truncate(0)
}

func checkPrepareTerraform(encoder *json.Encoder, scanner *bufio.Scanner, errorBuffer *bytes.Buffer) {
	if err := encoder.Encode(map[string]interface{}{
		"Action":  "prepare_terraform",
		"Version": "test-version",
		"EnvName": "test-env",
		"Config": map[string]interface{}{
			"config-key": "config-value",
		},
		"Env": map[string]string{
			"env-key": "env-value",
		},
	}); err != nil {
		log.Fatalln("error encoding json:", err)
	}

	readAndCheckOutputLine(scanner, map[string]interface{}{
		"Env": map[string]string{
			"response-env-key": "response-env-value",
		},
		"TerraformBackendConfig": map[string]string{
			"backend-key": "backend-value",
		},
		"TerraformBackendType": "test-backend-type",
		"TerraformImage":       "test-terraform-image",
	})
	if errorBuffer.String() != "version: test-version, env name: test-env, config value: config-value, env value: env-value\n" {
		log.Fatalln("unexpected prepare terraform debug output:", errorBuffer.String())
	}
	errorBuffer.Truncate(0)
}

func readAndCheckOutputLine(scanner *bufio.Scanner, expected map[string]interface{}) {
	if !scanner.Scan() {
		log.Fatalln("output finished")
	}
	line := scanner.Bytes()
	var message map[string]interface{}
	if err := json.Unmarshal(line, &message); err != nil {
		log.Fatalln("error reading message:", err)
	}
	if fmt.Sprintf("%v", message) != fmt.Sprintf("%v", expected) {
		log.Fatalln("unexpected message:", message)
	}
}

func TestCreateConfigureReleaseRequest(t *testing.T) {
	request := common.CreateConfigureReleaseRequest()
	// tests that the maps are initialised, otherwise these cause a panic
	request.Config["key"] = "value"
	request.Env["key"] = "value"
}

func TestCreateConfigureReleaseResponse(t *testing.T) {
	response := common.CreateConfigureReleaseResponse()
	response.Env["key"] = "value"
}

func TestCreateUploadReleaseRequest(t *testing.T) {
	request := common.CreateUploadReleaseRequest()
	request.ReleaseMetadata["key"] = map[string]string{}
}

func TestCreateUploadReleaseResponse(t *testing.T) {
	response := common.CreateUploadReleaseResponse()
	// not much to test for this
	if response.Message != "" {
		log.Fatalln("unexpected zero value")
	}
}

func TestCreatePrepareTerraformRequest(t *testing.T) {
	request := common.CreatePrepareTerraformRequest()
	request.Config["key"] = "value"
	request.Env["key"] = "value"
}

func TestCreatePrepareTerraformResponse(t *testing.T) {
	response := common.CreatePrepareTerraformResponse()
	response.Env["key"] = "value"
	response.TerraformBackendConfig["key"] = "value"
}
