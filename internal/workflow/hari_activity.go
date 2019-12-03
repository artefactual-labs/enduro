package workflow

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"time"
)

var hariClient = &http.Client{
	Timeout: 10 * time.Second,
	Transport: &http.Transport{
		DialContext: (&net.Dialer{
			Timeout: 5 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout: 5 * time.Second,
	},
}

type UpdateHARIActivity struct {
	manager *Manager
}

func NewUpdateHARIActivity(m *Manager) *UpdateHARIActivity {
	return &UpdateHARIActivity{manager: m}
}

func (a UpdateHARIActivity) Execute(ctx context.Context, tinfo *TransferInfo) error {
	apiURL, err := a.url()
	if err != nil {
		return nonRetryableError(fmt.Errorf("error in URL construction: %w", err))
	}

	mock, _ := hookAttrBool(a.manager.Hooks, "hari", "mock")
	if mock {
		ts := a.buildMock()
		defer ts.Close()
		apiURL = ts.URL
	}

	// Location of AVLXML, e.g.: `/transfer-path/<kind>/Journal/avlxml.xml`.
	var kind = strings.TrimSuffix(tinfo.Bundle.Kind, "-SIP")
	var path = filepath.Join(tinfo.Bundle.FullPath, kind, "Journal/avlxml.xml")
	blob, err := ioutil.ReadFile(path)
	if err != nil {
		return nonRetryableError(fmt.Errorf("error reading AVLXML file: %w (fullpath: %q)", err, path))
	}

	if err := a.sendRequest(ctx, blob, apiURL, tinfo); err != nil {
		return fmt.Errorf("error sending request: %w", err)
	}

	return nil
}

// url returns the HARI URL of the API endpoint for AVLXML submission.
func (a UpdateHARIActivity) url() (string, error) {
	p, _ := url.Parse("/v1/hari/avlxml")

	b, err := hookAttrString(a.manager.Hooks, "hari", "baseURL")
	if err != nil {
		return "", fmt.Errorf("error looking up baseURL configuration attribute: %w", err)
	}

	bu, err := url.Parse(b)
	if err != nil {
		return "", fmt.Errorf("error looking up baseURL configuration attribute: %w", err)
	}

	return bu.ResolveReference(p).String(), nil
}

func (a UpdateHARIActivity) sendRequest(ctx context.Context, blob []byte, apiURL string, tinfo *TransferInfo) error {
	payload := &avlRequest{
		XML:       blob,
		Message:   "AVLXML was processed by DPJ Archivematica pipeline",
		Type:      strings.ToLower(tinfo.Bundle.Kind),
		Timestamp: tinfo.StoredAt,
		AIPID:     tinfo.SIPID,
	}

	var buffer bytes.Buffer
	if err := json.NewEncoder(&buffer).Encode(payload); err != nil {
		return fmt.Errorf("error encoding payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, &buffer)
	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "application/json")

	resp, err := hariClient.Do(req)
	if err != nil {
		return err
	}

	switch {
	case resp.StatusCode >= 200 || resp.StatusCode <= 299:
		err = nil
	default:
		err = fmt.Errorf("unexpected status code: %s (%d)", resp.Status, resp.StatusCode)
	}

	return err
}

// buildMock returns a test server used when HARI's API is not available.
func (a UpdateHARIActivity) buildMock() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a.manager.Logger.Info(
			"Request received",
			"method", r.Method,
			"path", r.URL.Path,
		)
		fmt.Fprintln(w, "Hello!")
	}))
}

type avlRequest struct {
	XML       []byte    `json:"xml"`       // AVLXML document encoded using base64.
	Message   string    `json:"message"`   // E.g.: "AVLXML was processed by DPJ Archivematica pipeline"
	Type      string    `json:"type"`      // Lowercase. E.g.: "dpj", "epj", "other" or "avlxml".
	Timestamp time.Time `json:"timestamp"` // RFC3339. E.g. "2006-01-02T15:04:05Z07:00".
	AIPID     string    `json:"aip_id"`
}
