package amclient

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPackage_Create(t *testing.T) {
	setup()
	defer teardown()

	var (
		path    = "<uuid>:<path>"
		pathb64 = base64.StdEncoding.EncodeToString([]byte(path))
	)

	mux.HandleFunc("/api/v2beta/package/", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")

		blob, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		defer r.Body.Close()

		assert.Equal(t,
			string(bytes.TrimSpace(blob)),
			fmt.Sprintf(`{"name":"Foobar","type":"standard","path":"%s","accession":"12345","access_system_id":"fig-123","processing_config":"automated","auto_approve":true}`, pathb64))

		fmt.Fprint(w, `{"id": "096a284d-5067-4de0-a0a4-a684018cd6df"}`)
	})

	req := &PackageCreateRequest{
		Name:             "Foobar",
		Type:             "standard",
		Path:             path,
		Accession:        "12345",
		AccessSystemID:   "fig-123",
		ProcessingConfig: "automated",
		AutoApprove:      true,
	}
	payload, _, _ := client.Package.Create(ctx, req)

	if want, got := "096a284d-5067-4de0-a0a4-a684018cd6df", payload.ID; want != got {
		t.Errorf("Package.Create() id: got %v, want %v", got, want)
	}
}
