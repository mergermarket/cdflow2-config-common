package common

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
)

const defaultSocketPath = "/run/cdflow2-config/sock"

// CreateSetupRequest creates and returns an initialised SetupRequest - useful for testing config containers.
func CreateSetupRequest() *SetupRequest {
	var request SetupRequest
	request.Env = make(map[string]string)
	request.Config = make(map[string]interface{})
	request.ReleaseRequirements = make(map[string]*ReleaseRequirements)
	return &request
}

// CreateSetupResponse creates and returns an initialised SetupResponse.
func CreateSetupResponse() *SetupResponse {
	var response SetupResponse
	response.Success = true
	return &response
}

// CreateConfigureReleaseRequest creates and returns an initialised ConfigureReleaseRequest - useful for testing config containers.
func CreateConfigureReleaseRequest() *ConfigureReleaseRequest {
	var request ConfigureReleaseRequest
	request.Env = make(map[string]string)
	request.Config = make(map[string]interface{})
	request.ReleaseRequirements = make(map[string]*ReleaseRequirements)
	return &request
}

// CreateConfigureReleaseResponse creates and returns an initialised ConfigureReleaseResponse.
func CreateConfigureReleaseResponse() *ConfigureReleaseResponse {
	var response ConfigureReleaseResponse
	response.Env = make(map[string]map[string]string)
	response.AdditionalMetadata = make(map[string]string)
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
	response.TerraformBackendConfigParameters = make(map[string]*TerraformBackendConfigParameter)
	response.Success = true
	return &response
}

type releaseLoader struct{}

// CreateReleaseLoader returns a ReleaseLoader.
func CreateReleaseLoader() ReleaseLoader {
	return &releaseLoader{}
}

// Load unpacks a release into a release directory.
func (*releaseLoader) Load(
	reader io.Reader, component, version, releaseDir string,
) (string, error) {
	terraformImage, err := UnzipRelease(reader, releaseDir, component, version)
	if err != nil {
		return "", fmt.Errorf("error unzipping release in PrepareTerraform: %s", err)
	}
	return terraformImage, nil
}

type releaseSaver struct{}

// CreateReleaseSaver returns a ReleaseSaver.
func CreateReleaseSaver() ReleaseSaver {
	return &releaseSaver{}
}

// Save returns a reader for the release zip.
func (*releaseSaver) Save(
	component, version, terraformImage, releaseDir string,
) (io.ReadCloser, error) {
	file, err := ioutil.TempFile("", "cdflow2-config-common-release")
	if err != nil {
		return nil, err
	}
	defer os.Remove(file.Name())
	if err := ZipRelease(
		file, releaseDir, component, version, terraformImage,
	); err != nil {
		return nil, err
	}
	file.Seek(0, 0)
	return file, nil
}
