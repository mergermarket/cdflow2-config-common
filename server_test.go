package common_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"testing"

	common "github.com/mergermarket/cdflow2-config-common"
)

type handler struct {
	errorStream io.Writer
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

type FakeSigterm struct{}

func (FakeSigterm) String() string {
	return "SIGTERM"
}
func (FakeSigterm) Signal() {}

func TestRun(t *testing.T) {

	var errorBuffer bytes.Buffer

	socketPath := tempSock()
	defer os.Remove(socketPath)

	sigtermChannel := make(chan os.Signal, 1)

	go common.Listen(&handler{
		errorStream: &errorBuffer,
	}, socketPath, sigtermChannel)

	checkSetup(&errorBuffer, socketPath)
	checkRelease(&errorBuffer, socketPath)
	checkPrepareTerraform(&errorBuffer, socketPath)

	sigtermChannel <- FakeSigterm{}
}

func forward(request interface{}, socketPath string) (map[string]interface{}, error) {
	var requestBuffer bytes.Buffer
	var responseBuffer bytes.Buffer
	if err := json.NewEncoder(&requestBuffer).Encode(request); err != nil {
		return nil, err
	}
	common.Forward(&requestBuffer, &responseBuffer, socketPath)
	var message map[string]interface{}
	if err := json.NewDecoder(&responseBuffer).Decode(&message); err != nil {
		return nil, err
	}
	return message, nil
}

func (handler *handler) Setup(request *common.SetupRequest, response *common.SetupResponse) error {
	fmt.Fprintf(
		handler.errorStream,
		"env key: %v, config key: %v, release requirement: %v, required env: %v\n",
		request.Env["env-key"],
		request.Config["config-key"],
		request.ReleaseRequirements["release"]["key"],
		strings.Join(request.ReleaseRequiredEnv["release"], ", "),
	)
	if !response.Success {
		log.Fatal("success didn't default to true")
	}
	return nil
}

func checkSetup(errorBuffer *bytes.Buffer, socketPath string) {
	setupResponse, err := forward(map[string]interface{}{
		"Action": "setup",
		"Config": map[string]interface{}{"config-key": "config-value"},
		"Env":    map[string]string{"env-key": "env-value"},
		"ReleaseRequirements": map[string]map[string]interface{}{
			"release": {
				"env": []string{"a", "b"},
				"key": "value",
			},
		},
	}, socketPath)
	if err != nil {
		log.Fatalln("error calling setup:", err)
	}
	if success, ok := setupResponse["Success"].(bool); !ok || !success {
		log.Fatalln("success false from setup")
	}
	if errorBuffer.String() != "env key: env-value, config key: config-value, release requirement: value, required env: a, b\n" {
		log.Fatalln("unexpected setup debug output:", errorBuffer.String())
	}
	errorBuffer.Truncate(0)
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

func checkRelease(errorBuffer *bytes.Buffer, socketPath string) {
	configureReleaseResponse, err := forward(map[string]interface{}{
		"Action":  "configure_release",
		"Version": "test-version",
		"Config":  map[string]interface{}{"config-key": "config-value"},
		"Env":     map[string]string{"env-key": "env-value"},
		"ReleaseRequirements": map[string]map[string]interface{}{
			"release": {
				"env": []string{"a", "b"},
				"key": "value",
			},
		},
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
