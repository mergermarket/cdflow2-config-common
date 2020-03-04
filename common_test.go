package common_test

import (
	"archive/zip"
	"bytes"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"testing"

	common "github.com/mergermarket/cdflow2-config-common"
)

func TestCreateSetupRequest(t *testing.T) {
	request := common.CreateSetupRequest()
	// tests that the maps are initialised, otherwise these cause a panic
	request.Config["key"] = "value"
	request.Env["key"] = "value"
	request.ReleaseRequirements["key"] = "value"
}

func TestCreateSetupResponse(t *testing.T) {
	response := common.CreateSetupResponse()
	if !response.Success {
		log.Fatal("success didn't default to true")
	}
}

func TestCreateConfigureReleaseRequest(t *testing.T) {
	request := common.CreateConfigureReleaseRequest()
	// tests that the maps are initialised, otherwise these cause a panic
	request.Config["key"] = "value"
	request.Env["key"] = "value"
	request.ReleaseRequirements["key"] = "value"
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
