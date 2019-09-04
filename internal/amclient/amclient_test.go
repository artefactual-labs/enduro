package amclient

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
)

var (
	mux    *http.ServeMux
	ctx    = context.TODO()
	client *Client
	server *httptest.Server
)

func setup() {
	mux = http.NewServeMux()
	server = httptest.NewServer(mux)
	url, _ := url.Parse(server.URL)
	client = NewClient(nil, url.String(), "", "")
}

func teardown() {
	server.Close()
}

func testMethod(t *testing.T, r *http.Request, expected string) {
	if expected != r.Method {
		t.Errorf("Request method = %v, expected %v", r.Method, expected)
	}
}

func TestDo(t *testing.T) {
	setup()
	defer teardown()

	type foo struct {
		A string
	}

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if m := "GET"; m != r.Method {
			t.Errorf("Request method = %v, expected %v", r.Method, m)
		}
		fmt.Fprint(w, `{"A":"a"}`)
	})

	req, _ := client.NewRequest(ctx, "GET", "/", nil)
	body := new(foo)
	_, err := client.Do(context.Background(), req, body)
	if err != nil {
		t.Fatalf("Do(): %v", err)
	}

	expected := &foo{"a"}
	if !reflect.DeepEqual(body, expected) {
		t.Errorf("Response body = %v, expected %v", body, expected)
	}
}

func TestDo_httpError(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Bad Request", 400)
	})

	req, _ := client.NewRequest(ctx, "GET", "/", nil)
	_, err := client.Do(context.Background(), req, nil)

	if err == nil {
		t.Error("Expected HTTP 400 error.")
	}
}

func TestCustomUserAgent(t *testing.T) {
	c, err := New(nil, "http://127.0.0.1", "", "", SetUserAgent("testing"))

	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}

	expected := fmt.Sprintf("%s+%s", "testing", userAgent)
	if got := c.UserAgent; got != expected {
		t.Errorf("New() UserAgent = %s; expected %s", got, expected)
	}
}

func TestNewRequest(t *testing.T) {
	var (
		baseURL  = "http://127.0.0.1"
		user     = "Us3r"
		password = "Pa33w0rd"
	)

	c, _ := New(nil, baseURL, user, password)

	inURL, outURL := "/foo", baseURL+"/foo"
	inBody := &TransferStartRequest{Name: "My transfer", Type: "standard"}
	req, _ := c.NewRequest(context.Background(), "GET", inURL, inBody)

	// Test that relative URL was expanded.
	if got, want := req.URL.String(), outURL; got != want {
		t.Errorf("NewRequest(%q) URL is %v, want %v", inURL, got, want)
	}

	// Test that default user-agent is attached to the request.
	if got, want := req.Header.Get("User-Agent"), c.UserAgent; got != want {
		t.Errorf("NewRequest() User-Agent is %v, want %v", got, want)
	}

	// Test that the Authorization header is included.
	if got, want := req.Header.Get("Authorization"), fmt.Sprintf("ApiKey %s:%s", user, password); got != want {
		t.Fatalf("NewRequest() Authorization header: %v, want %v", got, want)
	}
}

func TestNewRequestJSON(t *testing.T) {
	var (
		baseURL  = "http://127.0.0.1"
		user     = "Us3r"
		password = "Pa33w0rd"
	)

	c, _ := New(nil, baseURL, user, password)

	inBody := struct {
		Test string `json:"string"`
	}{Test: "foobar"}
	req, _ := c.NewRequestJSON(context.Background(), "GET", "/foo", inBody)

	// Test that the Authorization header is included.
	if got, want := req.Header.Get("Content-Type"), mediaTypeJSON; got != want {
		t.Fatalf("NewRequest() Content-Type header: %v, want %v", got, want)
	}

	got, _ := ioutil.ReadAll(req.Body)
	want := []byte(`{"string":"foobar"}
`)
	defer req.Body.Close()
	if !bytes.Equal(got, want) {
		t.Fatalf("NewRequest() Body, got: %s, want %s", got, want)
	}
}
