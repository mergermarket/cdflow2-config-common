package common_test

import (
	"archive/zip"
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

type handler struct {
	errorStream io.Writer
}

func (handler *handler) ConfigureRelease(request *common.ConfigureReleaseRequest, response *common.ConfigureReleaseResponse) error {
	fmt.Fprintf(handler.errorStream, "version: %v, env key: %v, config key: %v\n", request.Version, request.Env["env-key"], request.Config["config-key"])
	response.Env["response-env-key"] = "response-env-value"
	if !response.Success {
		log.Fatal("success didn't default to true")
	}
	return nil
}

func (handler *handler) UploadRelease(request *common.UploadReleaseRequest, response *common.UploadReleaseResponse, version string, config map[string]interface{}) error {
	fmt.Fprintf(handler.errorStream, "terraform image: %v, release metadata value: %v, config key: %v\n", request.TerraformImage, request.ReleaseMetadata["release"]["release-key"], config["config-key"])
	response.Message = "test-uploaded-message"
	if !response.Success {
		log.Fatal("success didn't default to true")
	}
	return nil
}

func (handler *handler) PrepareTerraform(request *common.PrepareTerraformRequest, response *common.PrepareTerraformResponse) error {
	fmt.Fprintf(handler.errorStream, "version: %v, env name: %v, config value: %v, env value: %v\n", request.Version, request.EnvName, request.Config["config-key"], request.Env["env-key"])
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

func tempSock() string {
	f, err := ioutil.TempFile("", "cdflow2-config-common-test-sock-*")
	if err != nil {
		log.Panic("could not create temp file:", err)
	}
	f.Close()
	os.Remove(f.Name())
	return f.Name()
}

func TestRun(t *testing.T) {

	var errorBuffer bytes.Buffer

	socketPath := tempSock()
	defer os.Remove(socketPath)

	go common.Run(&handler{
		errorStream: &errorBuffer,
	}, []string{}, nil, nil, socketPath)

	checkRelease(&errorBuffer, socketPath)
	checkPrepareTerraform(&errorBuffer, socketPath)
	forward(map[string]string{"Action": "stop"}, socketPath)
}

func forward(request interface{}, socketPath string) (interface{}, error) {
	var requestBuffer bytes.Buffer
	var responseBuffer bytes.Buffer
	if err := json.NewEncoder(&requestBuffer).Encode(request); err != nil {
		return nil, err
	}
	common.Run(&handler{}, []string{"forward"}, &requestBuffer, &responseBuffer, socketPath)
	var message map[string]interface{}
	if err := json.NewDecoder(&responseBuffer).Decode(&message); err != nil {
		return nil, err
	}
	return message, nil
}

func checkRelease(errorBuffer *bytes.Buffer, socketPath string) {
	configureReleaseResponse, err := forward(map[string]interface{}{
		"Action":  "configure_release",
		"Version": "test-version",
		"Config":  map[string]interface{}{"config-key": "config-value"},
		"Env":     map[string]string{"env-key": "env-value"},
	}, socketPath)
	if err != nil {
		log.Fatalln("error calling configure release:", err)
	}
	if fmt.Sprintf("%v", configureReleaseResponse) != fmt.Sprintf("%v", map[string]interface{}{
		"Env": map[string]string{
			"response-env-key": "response-env-value",
		},
		"Success": true,
	}) {
		log.Fatalln("unexpected configure release response:", configureReleaseResponse)
	}

	if errorBuffer.String() != "version: test-version, env key: env-value, config key: config-value\n" {
		log.Fatalln("unexpected configure release debug output:", errorBuffer.String())
	}

	errorBuffer.Truncate(0)

	uploadReleaseResponse, err := forward(map[string]interface{}{
		"Action":         "upload_release",
		"TerraformImage": "test-terraform-image",
		"ReleaseMetadata": map[string]map[string]string{
			"release": map[string]string{
				"release-key": "release-value",
			},
		},
	}, socketPath)
	if err != nil {
		log.Fatalln("error calling upload release:", err)
	}
	if fmt.Sprintf("%v", uploadReleaseResponse) != fmt.Sprintf("%v", map[string]interface{}{
		"Message": "test-uploaded-message",
		"Success": true,
	}) {
		log.Fatalln("unexpected upload release response:", uploadReleaseResponse)
	}

	if errorBuffer.String() != "terraform image: test-terraform-image, release metadata value: release-value, config key: config-value\n" {
		log.Fatalln("unexpected upload release debug output:", errorBuffer.String())
	}
	errorBuffer.Truncate(0)
}

func checkPrepareTerraform(errorBuffer *bytes.Buffer, socketPath string) {
	prepareTerraformResponse, err := forward(map[string]interface{}{
		"Action":  "prepare_terraform",
		"Version": "test-version",
		"EnvName": "test-env",
		"Config": map[string]interface{}{
			"config-key": "config-value",
		},
		"Env": map[string]string{
			"env-key": "env-value",
		},
	}, socketPath)
	if err != nil {
		log.Fatalln("error calling prepare terraform:", err)
	}

	if fmt.Sprintf("%v", prepareTerraformResponse) != fmt.Sprintf("%v", map[string]interface{}{
		"Env": map[string]string{
			"response-env-key": "response-env-value",
		},
		"TerraformBackendConfig": map[string]string{
			"backend-key": "backend-value",
		},
		"TerraformBackendType": "test-backend-type",
		"TerraformImage":       "test-terraform-image",
		"Success":              true,
	}) {
		log.Fatalln("unexpected prepare terraform response:", prepareTerraformResponse)
	}
	if errorBuffer.String() != "version: test-version, env name: test-env, config value: config-value, env value: env-value\n" {
		log.Fatalln("unexpected prepare terraform debug output:", errorBuffer.String())
	}

	errorBuffer.Truncate(0)
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
