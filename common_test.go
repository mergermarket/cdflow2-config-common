package common_test

import (
	"archive/zip"
	"bytes"
	"io/ioutil"
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
	request.ReleaseRequirements["release-key"] = &common.ReleaseRequirements{Needs: []string{"need1"}}
}

func TestCreateSetupResponse(t *testing.T) {
	response := common.CreateSetupResponse()
	if !response.Success {
		t.Fatal("success didn't default to true")
	}
}

func TestCreateConfigureReleaseRequest(t *testing.T) {
	request := common.CreateConfigureReleaseRequest()
	// tests that the maps are initialised, otherwise these cause a panic
	request.Config["key"] = "value"
	request.Env["key"] = "value"
	request.ReleaseRequirements["release-key"] = &common.ReleaseRequirements{Needs: []string{"need1"}}
}

func TestCreateConfigureReleaseResponse(t *testing.T) {
	response := common.CreateConfigureReleaseResponse()
	response.Env["test-build-id"] = map[string]string{"key": "value"}
	if !response.Success {
		t.Fatal("success didn't default to true")
	}
}

func TestCreateUploadReleaseRequest(t *testing.T) {
	request := common.CreateUploadReleaseRequest()
	request.ReleaseMetadata["key"] = map[string]string{}
}

func TestCreateUploadReleaseResponse(t *testing.T) {
	response := common.CreateUploadReleaseResponse()
	if !response.Success {
		t.Fatal("success didn't default to true")
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
		t.Fatal("success didn't default to true")
	}
}

func releaseDir(t *testing.T) string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("couldn't get test filename")
	}
	return path.Join(path.Dir(filename), "test/release/")
}

func TestZipRelease(t *testing.T) {
	// Given
	releaseDir := releaseDir(t)
	var buffer bytes.Buffer

	// When
	if err := common.ZipRelease(&buffer, releaseDir, "test-component", "test-version", "test-terraform-image"); err != nil {
		t.Fatal("error zipping release:", err)
	}

	// Then
	zipReader, err := zip.NewReader(bytes.NewReader(buffer.Bytes()), int64(len(buffer.Bytes())))
	if err != nil {
		t.Fatal("could not create zip reader:", err)
	}
	if len(zipReader.File) != 2 {
		t.Fatalf("expected %v, got %v", 2, len(zipReader.File))
	}
	if zipReader.File[0].Name != "test-component-test-version/terraform-image" {
		t.Fatal("unexpected filename in zip:", zipReader.File[0].Name)
	}
	if zipReader.File[1].Name != "test-component-test-version/test.txt" {
		t.Fatal("unexpected filename in zip:", zipReader.File[1].Name)
	}
}

func TestUnzipRelease(t *testing.T) {
	// Given
	dir, err := ioutil.TempDir("", "cdlfow2-config-common-test-unzip-release")
	if err != nil {
		t.Fatal("error creating temporary directory:", err)
	}
	defer os.RemoveAll(dir)

	releaseDir := releaseDir(t)
	var buffer bytes.Buffer

	if err := common.ZipRelease(&buffer, releaseDir, "test-component", "test-version", "test-terraform-image"); err != nil {
		t.Fatal("error zipping release:", err)
	}
	data := buffer.Bytes()

	// When
	if err := common.UnzipRelease(bytes.NewReader(data), int64(len(data)), dir, "test-component", "test-version"); err != nil {
		t.Fatal("unexpected error unzipping release:", err)
	}

	// Then
	if data, err := ioutil.ReadFile(filepath.Join(dir, "test.txt")); err != nil || string(data) != "test" {
		t.Fatalf("problem reading file, data: %v, error: %v\n", data, err)
	}
}
