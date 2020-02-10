package common_test

import (
	"archive/zip"
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"testing"

	common "github.com/mergermarket/cdflow2-config-common"
)

type handler struct{}

func (*handler) ConfigureRelease(request *common.ConfigureReleaseRequest, response *common.ConfigureReleaseResponse, errorStream io.Writer) error {
	fmt.Fprintf(errorStream, "version: %v, env key: %v, config key: %v\n", request.Version, request.Env["env-key"], request.Config["config-key"])
	response.Env["response-env-key"] = "response-env-value"
	if !response.Success {
		log.Fatal("success didn't default to true")
	}
	return nil
}

func (*handler) UploadRelease(request *common.UploadReleaseRequest, response *common.UploadReleaseResponse, errorStream io.Writer, version string, config map[string]interface{}) error {
	fmt.Fprintf(errorStream, "terraform image: %v, release metadata value: %v, config key: %v\n", request.TerraformImage, request.ReleaseMetadata["release"]["release-key"], config["config-key"])
	response.Message = "test-uploaded-message"
	if !response.Success {
		log.Fatal("success didn't default to true")
	}
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
	if !response.Success {
		log.Fatal("success didn't default to true")
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

	checkRelease(encoder, scanner, &errorBuffer)
	checkPrepareTerraform(encoder, scanner, &errorBuffer)

	encoder.Encode(map[string]string{"Action": "stop"})
}

func checkRelease(encoder *json.Encoder, scanner *bufio.Scanner, errorBuffer *bytes.Buffer) {
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
		"Success": true,
	})
	if errorBuffer.String() != "version: test-version, env key: env-value, config key: config-value\n" {
		log.Fatalln("unexpected configure release debug output:", errorBuffer.String())
	}
	errorBuffer.Truncate(0)

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
		"Success": true,
	})
	if errorBuffer.String() != "terraform image: test-terraform-image, release metadata value: release-value, config key: config-value\n" {
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
		"Success":              true,
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
	if !response.Success {
		log.Fatal("success didn't default to true")
	}
}

func TestCreateUploadReleaseRequest(t *testing.T) {
	request := common.CreateUploadReleaseRequest()
	request.ReleaseMetadata["key"] = map[string]string{}
}

func TestCreateUploadReleaseResponse(t *testing.T) {
	response := common.CreateUploadReleaseResponse()
	if !response.Success {
		log.Fatal("success didn't default to true")
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
	if !response.Success {
		log.Fatal("success didn't default to true")
	}
}

func releaseDir() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatalln("couldn't get test filename")
	}
	return path.Join(path.Dir(filename), "test/release/")
}

func TestZipRelease(t *testing.T) {
	// Given
	releaseDir := releaseDir()
	var buffer bytes.Buffer

	// When
	if err := common.ZipRelease(&buffer, releaseDir, "test-component", "test-version"); err != nil {
		log.Fatalln("error zipping release:", err)
	}

	// Then
	zipReader, err := zip.NewReader(bytes.NewReader(buffer.Bytes()), int64(len(buffer.Bytes())))
	if err != nil {
		log.Fatalln("could not create zip reader:", err)
	}
	if len(zipReader.File) != 1 || zipReader.File[0].Name != "test-component-test-version/test.txt" {
		log.Fatalln("unexpected filename in zip:", zipReader.File[0].Name)
	}
}

func TestUnzipRelease(t *testing.T) {
	// Given
	dir, err := ioutil.TempDir("", "cdlfow2-config-common-test-unzip-release")
	if err != nil {
		log.Panicln("error creating temporary directory:", err)
	}
	defer os.RemoveAll(dir)

	releaseDir := releaseDir()
	var buffer bytes.Buffer

	if err := common.ZipRelease(&buffer, releaseDir, "test-component", "test-version"); err != nil {
		log.Panicln("error zipping release:", err)
	}
	data := buffer.Bytes()

	// When
	if err := common.UnzipRelease(bytes.NewReader(data), int64(len(data)), dir, "test-component", "test-version"); err != nil {
		log.Panicln("unexpected error unzipping release:", err)
	}

	// Then
	if data, err := ioutil.ReadFile(filepath.Join(dir, "test.txt")); err != nil || string(data) != "test" {
		log.Panicf("problem reading file, data: %v, error: %v\n", data, err)
	}
}
