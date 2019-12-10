package amclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/schema"
)

const (
	userAgent     = "Enduro (amclient)"
	mediaTypeForm = "application/x-www-form-urlencoded"
	mediaTypeJSON = "application/json"
)

// Client manages communication with Archivematica API.
type Client struct {
	// HTTP client used to communicate with the Archivematica API.
	client *http.Client

	// Base URL for API requests.
	BaseURL *url.URL

	// User agent for client
	UserAgent string

	// Authentication
	User string
	Key  string

	// Services used for communicating with the API
	Transfer         TransferService
	Ingest           IngestService
	ProcessingConfig ProcessingConfigService
	Package          PackageService
	Jobs             JobsService
	Task             TaskService
}

// Response is an Archivematica response. This wraps the standard http.Response
// returned from Archivematica.
type Response struct {
	*http.Response
}

// An ErrorResponse reports the error caused by an API request
type ErrorResponse struct {
	// HTTP response that caused this error
	Response *http.Response
}

func (r *ErrorResponse) Error() string {
	return fmt.Sprintf("%v %v: %d",
		r.Response.Request.Method,
		r.Response.Request.URL,
		r.Response.StatusCode)
}

// NewClient returns a new Archivematica API client.
func NewClient(httpClient *http.Client, bu, u, k string) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	pur, _ := url.Parse(bu)
	c := &Client{
		client:    httpClient,
		BaseURL:   pur,
		User:      u,
		Key:       k,
		UserAgent: userAgent,
	}
	c.Transfer = &TransferServiceOp{client: c}
	c.Ingest = &IngestServiceOp{client: c}
	c.ProcessingConfig = &ProcessingConfigOp{client: c}
	c.Package = &PackageServiceOp{client: c}
	c.Jobs = &JobsServiceOp{client: c}
	c.Task = &TaskServiceOp{client: c}
	return c
}

// ClientOpt are options for New.
type ClientOpt func(*Client) error

// New returns a new Archivematica API client instance.
func New(httpClient *http.Client, bu, u, k string, opts ...ClientOpt) (*Client, error) {
	c := NewClient(httpClient, bu, u, k)
	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, err
		}
	}

	return c, nil
}

// SetUserAgent is a client option for setting the user agent.
func SetUserAgent(ua string) ClientOpt {
	return func(c *Client) error {
		c.UserAgent = fmt.Sprintf("%s+%s", ua, c.UserAgent)
		return nil
	}
}

// RequestOpt is a function type used to alter requests.
type RequestOpt func(*http.Request)

// WithRequestAcceptXML sets the Accept header to "application/xml". This is
// needed when consuming endpoints that require this configuration.
func WithRequestAcceptXML() RequestOpt {
	return func(req *http.Request) {
		req.Header.Set("Accept", "application/xml")
	}
}

func (c *Client) newRequest(ctx context.Context, method, urlStr, mediaType string, body io.Reader, opts ...RequestOpt) (*http.Request, error) {
	rel, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	u := c.BaseURL.ResolveReference(rel)

	req, err := http.NewRequest(method, u.String(), body)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", mediaType)
	req.Header.Add("Accept", mediaType)
	req.Header.Add("User-Agent", c.UserAgent)
	req.Header.Add("Authorization", fmt.Sprintf("ApiKey %s:%s", c.User, c.Key))

	for _, fn := range opts {
		fn(req)
	}

	return req, nil
}

// NewRequest creates an API request. A relative URL can be provided in urlStr,
// which will be resolved to the BaseURL of the Client. Relative URLS should
// always be specified without a preceding slash. If specified, the value
// pointed to by body is form encoded and included in as the request body.
func (c *Client) NewRequest(ctx context.Context, method, urlStr string, body interface{}, opts ...RequestOpt) (*http.Request, error) {
	form := url.Values{}
	encoder := schema.NewEncoder()
	_ = encoder.Encode(body, form)
	reader := strings.NewReader(form.Encode())
	return c.newRequest(ctx, method, urlStr, mediaTypeForm, reader, opts...)
}

// NewRequestJSON is similar to NewRequest but encodes the value to JSON.
func (c *Client) NewRequestJSON(ctx context.Context, method, urlStr string, body interface{}, opts ...RequestOpt) (*http.Request, error) {
	buf := new(bytes.Buffer)
	if body != nil {
		if err := json.NewEncoder(buf).Encode(body); err != nil {
			return nil, err
		}
	}
	return c.newRequest(ctx, method, urlStr, mediaTypeJSON, buf, opts...)
}

// newResponse creates a new Response for the provided http.Response
func newResponse(r *http.Response) *Response {
	return &Response{Response: r}
}

// Do sends an API request and returns the API response. The API response is
// JSON decoded and stored in the value pointed to by v, or returned as an error
// if an API error has occurred. If v implements the io.Writer interface, the
// raw response will be written to v, without attempting to decode it.
func (c *Client) Do(ctx context.Context, req *http.Request, v interface{}) (*Response, error) {
	req = req.WithContext(ctx)
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer func() {
		if rerr := resp.Body.Close(); err == nil {
			err = rerr
		}
	}()

	response := newResponse(resp)

	err = CheckResponse(resp)
	if err != nil {
		return response, err
	}

	if v != nil {
		if w, ok := v.(io.Writer); ok {
			_, err = io.Copy(w, resp.Body)
			if err != nil {
				return nil, err
			}
		} else {
			err = json.NewDecoder(resp.Body).Decode(v)
			if err != nil {
				return nil, err
			}
		}
	}

	return response, err
}

// CheckResponse checks the API response for errors, and returns them if
// present. A response is considered an error if it has a status code outside
// the 200 range. API error responses are expected to have either no response
// body, or a JSON response body that maps to ErrorResponse. Any other response
// body will be silently ignored.
func CheckResponse(r *http.Response) error {
	if c := r.StatusCode; c >= 200 && c <= 299 {
		return nil
	}
	return &ErrorResponse{Response: r}
}
