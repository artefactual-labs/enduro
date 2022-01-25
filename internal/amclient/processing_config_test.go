package amclient

import (
	"fmt"
	"net/http"
	"reflect"
	"testing"
)

func TestProcessingConfig_Get(t *testing.T) {
	setup()
	defer teardown()

	const document = `<processingMCP>
  <preconfiguredChoices>
    <preconfiguredChoice>
      <appliesTo>56eebd45-5600-4768-a8c2-ec0114555a3d</appliesTo>
      <goToChain>e9eaef1e-c2e0-4e3b-b942-bfb537162795</goToChain>
    </preconfiguredChoice>
  </preconfiguredChoices>
</processingMCP>`

	mux.HandleFunc("/api/processing-configuration/default/", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		fmt.Fprint(w, document)
	})

	payload, _, err := client.ProcessingConfig.Get(ctx, "default")
	if err != nil {
		t.Fatalf("ProcessingConfig.Get returned error: %v", err)
	}

	if want, got := document, payload.String(); want != got {
		t.Fatalf("ProcessingConfig.Get: Document = %v, want %v", got, want)
	}
}

func TestProcessingConfig_List(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/api/processing-configuration/", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		fmt.Fprint(w, `{"processing_configurations": ["automated", "default"]}`)
	})

	payload, _, err := client.ProcessingConfig.List(ctx)
	if err != nil {
		t.Fatalf("ProcessingConfig.List returned error: %v", err)
	}

	if want, got := []string{"automated", "default"}, payload; !reflect.DeepEqual(want, got) {
		t.Fatalf("ProcessingConfig.Get: got = %v, want %v", got, want)
	}
}
