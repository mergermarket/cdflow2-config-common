Common parts for a cdflow2 config container.

The main package should look like:

```go
package main

import (
	"os"

	common "github.com/mergermarket/cdflow2-config-common"
	handler "TODO: handler package"
)

func main() {
	common.Run(handler.New(), os.Stdin, os.Stdout, os.Stderr)
}
```

The handler package should look like:

```go
package handler

import (
	"io"

	common "github.com/mergermarket/cdflow2-config-common"
)

// this can be used to persist state between requests
type handler struct{}

// New returns a new handler.
func New() common.Handler {
	return &handler{}
}

// ConfigureRelease handles a configure release request in order to prepare for the release container to be ran.
func (*handler) ConfigureRelease(request *common.ConfigureReleaseRequest, response *common.ConfigureReleaseResponse, errorStream io.Writer) error {

	return nil
}

// UploadRelease handles an upload release request in order to upload the release after the release container is run.
func (*handler) UploadRelease(request *common.UploadReleaseRequest, response *common.UploadReleaseResponse, errorStream io.Writer, version string) error {

	return nil
}

// PrepareTerraform handles a prepare terraform request in order to provide configuration for terraform during a deploy, destroy, etc.
func (*handler) PrepareTerraform(request *common.PrepareTerraformRequest, response *common.PrepareTerraformResponse, errorStream io.Writer) error {

	return nil
}
```

See [interface.go](interface.go) for the `Handler` interface and associated request and resposne types.
