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

	if response.Monitoring == nil || response.Monitoring.Data == nil {
		t.Fatal("Monitoring data does not initialized")
	}

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
	response.AdditionalMetadata["foo"] = "bar"

	if response.Monitoring == nil || response.Monitoring.Data == nil {
		t.Fatal("Monitoring data does not initialized")
	}

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
	if request.StateShouldExist != nil {
		t.Fatalf("expected nil, got %v", request.StateShouldExist)
	}

	var (
		T = true
		F = false
	)

	request.StateShouldExist = &T
	if *request.StateShouldExist != true {
		t.Fatalf("expected %v, got %v", true, *request.StateShouldExist)
	}

	request.StateShouldExist = &F
	if *request.StateShouldExist != false {
		t.Fatalf("expected %v, got %v", false, *request.StateShouldExist)
	}
}

func TestCreatePrepareTerraformResponse(t *testing.T) {
	response := common.CreatePrepareTerraformResponse()
	response.Env["key"] = "value"
	response.TerraformBackendConfig["key"] = "value"

	if response.Monitoring == nil || response.Monitoring.Data == nil {
		t.Fatal("Monitoring data does not initialized")
	}

	if !response.Success {
		t.Fatal("success didn't default to true")
	}
}

func releaseDir(t *testing.T) string {
	t.Helper()
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

	const expectedPath = ".terraform/plugins/foo/bar"
	const expectedPluginSHA256 = "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"

	// When
	if err := common.ZipRelease(
		&buffer, releaseDir, "test-component", "test-version", "test-terraform-image",
	); err != nil {
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
	dir, err := ioutil.TempDir("", "cdflow2-config-common-test-unzip-release")
	if err != nil {
		t.Fatal("error creating temporary directory:", err)
	}
	defer os.RemoveAll(dir)

	releaseDir := releaseDir(t)
	var buffer bytes.Buffer

	terraformImage := "test-terraform-image"

	if err := common.ZipRelease(
		&buffer, releaseDir, "test-component", "test-version", terraformImage,
	); err != nil {
		t.Fatal("error zipping release:", err)
	}
	data := buffer.Bytes()

	// When
	gotTerraformImage, err := common.UnzipRelease(
		bytes.NewReader(data), dir, "test-component", "test-version",
	)
	if err != nil {
		t.Fatal("unexpected error unzipping release:", err)
	}
	if gotTerraformImage != terraformImage {
		t.Fatalf("got %q, wanted %q", gotTerraformImage, terraformImage)
	}

	// Then
	if data, err := ioutil.ReadFile(filepath.Join(dir, "test.txt")); err != nil || string(data) != "test" {
		t.Fatalf("problem reading file, data: %v, error: %v\n", data, err)
	}
}
