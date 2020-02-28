package common

// ConfigureReleaseRequest is the incoming configure release request format.
type ConfigureReleaseRequest struct {
	Version string
	Config  map[string]interface{}
	Env     map[string]string
}

// ConfigureReleaseResponse is the outgoing configure release response format.
type ConfigureReleaseResponse struct {
	Env     map[string]string
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
	ConfigureRelease(request *ConfigureReleaseRequest, response *ConfigureReleaseResponse) error
	UploadRelease(request *UploadReleaseRequest, response *UploadReleaseResponse, version string, config map[string]interface{}) error
	PrepareTerraform(request *PrepareTerraformRequest, response *PrepareTerraformResponse) error
}
