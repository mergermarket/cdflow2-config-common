package common

import "io"

// ReleaseRequirements contains a list of needs of a build container.
type ReleaseRequirements struct {
	Needs []string
}

// PrepareReleaseRequest contains the fields that are common for setup and configure release requests.
type PrepareReleaseRequest struct {
	Component           string
	Commit              string
	Team                string
	Config              map[string]interface{}
	Env                 map[string]string
	ReleaseRequirements map[string]*ReleaseRequirements
}

// SetupRequest is the incoming setup request format.
type SetupRequest struct {
	PrepareReleaseRequest
}

// SetupResponse is the outgoing setup response format.
type SetupResponse struct {
	Success bool
}

// ConfigureReleaseRequest is the incoming configure release request format.
type ConfigureReleaseRequest struct {
	PrepareReleaseRequest
	Version string
}

// ConfigureReleaseResponse is the outgoing configure release response format.
type ConfigureReleaseResponse struct {
	Env     map[string]map[string]string
	Success bool
}

// UploadReleaseRequest is the incoming upload release request format.
type UploadReleaseRequest struct {
	TerraformImage  string
	ReleaseMetadata map[string]map[string]string
}

// UploadReleaseResponse is the outgoing upload release response format.
type UploadReleaseResponse struct {
	Message string
	Success bool
}

// PrepareTerraformRequest is the incoming prepare terraform request format.
type PrepareTerraformRequest struct {
	Version string
	EnvName string
	Config  map[string]interface{}
	Env     map[string]string
}

// PrepareTerraformResponse is the outgoing prepare terraform response format.
type PrepareTerraformResponse struct {
	TerraformImage         string
	Env                    map[string]string
	TerraformBackendType   string
	TerraformBackendConfig map[string]string
	Success                bool
}

// Handler has methods to handle each bit of config.
type Handler interface {
	Setup(request *SetupRequest, response *SetupResponse) error
	ConfigureRelease(request *ConfigureReleaseRequest, response *ConfigureReleaseResponse) error
	UploadRelease(request *UploadReleaseRequest, response *UploadReleaseResponse, configureReleaseRequest *ConfigureReleaseRequest, releaseReader io.ReadSeeker) error
	PrepareTerraform(request *PrepareTerraformRequest, response *PrepareTerraformResponse) error
}
