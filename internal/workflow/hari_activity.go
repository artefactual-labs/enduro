package workflow

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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
	if tinfo.OriginalID == "" {
		return nonRetryableError(errors.New("unknown originalID"))
	}

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

	if err := a.sendRequest(ctx, apiURL, tinfo); err != nil {
		return fmt.Errorf("error sending request: %w", err)
	}

	return nil
}

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

func (a UpdateHARIActivity) sendRequest(ctx context.Context, apiURL string, tinfo *TransferInfo) error {
	// Location of AVLXML, e.g.: // e.g. `/transfer-path/<uuid>/DPJ/journal/<uuid>.xml`.
	var path = filepath.Join(tinfo.FullPath, tinfo.OriginalID, "DPJ", "journal", tinfo.OriginalID+".xml")

	// Is there a better way to do this? We need to build the JSON document but
	// maybe this can be done with a buffer?
	blob, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf("error reading AVLXML file: %w (fullpath: %q)", err, path)
	}

	payload := &avlRequest{
		XML:       blob,
		Message:   "AVLXML was processed by DPJ Archivematica pipeline",
		Type:      strings.ToLower(tinfo.Kind),
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
