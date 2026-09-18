package openapi_test

import (
	"bytes"
	"os"
	"testing"
)

func TestServerOpenAPIMatchesRootSpecification(t *testing.T) {
	rootSpec, err := os.ReadFile("../../../../person-service.yaml")
	if err != nil {
		t.Fatalf("read root OpenAPI specification: %v", err)
	}
	serverSpec, err := os.ReadFile("../../../internal/httpapi/openapi.yaml")
	if err != nil {
		t.Fatalf("read server OpenAPI specification: %v", err)
	}
	if !bytes.Equal(rootSpec, serverSpec) {
		t.Fatal("embedded OpenAPI specification differs from person-service.yaml")
	}
}
