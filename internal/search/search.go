/*
Package search provides search capabilities to Enduro.
*/
package search

import (
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/opensearch-project/opensearch-go"
)

var defaultResponse = http.Response{
	Status:        "200 OK",
	StatusCode:    200,
	ContentLength: 2,
	Header:        http.Header(map[string][]string{"Content-Type": {"application/json"}}),
	Body:          ioutil.NopCloser(strings.NewReader(`{}`)),
}

type FakeTransport struct {
	FakeResponse *http.Response
}

func (t *FakeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return t.FakeResponse, nil
}

func newFakeTransport() *FakeTransport {
	return &FakeTransport{FakeResponse: &defaultResponse}
}

func NewFakeClient() *opensearch.Client {
	client, _ := opensearch.NewClient(opensearch.Config{
		Addresses: []string{"http://localhost:9200"},
		Transport: newFakeTransport(),
	})
	return client
}
