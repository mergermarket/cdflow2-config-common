package common

import (
	"io"
)

const defaultSocketPath = "/sock"

// Run handles all the IO, calling into the passed in Handler to do the actual work.
func Run(handler Handler, args []string, readStream io.Reader, writeStream io.Writer, overrideSocketPath string) {
	socketPath := overrideSocketPath
	if socketPath == "" {
		socketPath = defaultSocketPath
	}
	if len(args) == 1 && args[0] == "forward" {
		forward(readStream, writeStream, socketPath)
	} else {
		Listen(handler, socketPath, getSigtermChannel())
	}
}