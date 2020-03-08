package activities

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
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/artefactual-labs/enduro/internal/nha"
	wferrors "github.com/artefactual-labs/enduro/internal/workflow/errors"
	"github.com/artefactual-labs/enduro/internal/workflow/manager"
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

// UpdateHARIActivity delivers a receipt to HARI.
type UpdateHARIActivity struct {
	manager *manager.Manager
}

func NewUpdateHARIActivity(m *manager.Manager) *UpdateHARIActivity {
	return &UpdateHARIActivity{manager: m}
}

type UpdateHARIActivityParams struct {
	SIPID        string
	StoredAt     time.Time
	FullPath     string
	PipelineName string
	NameInfo     nha.NameInfo
}

func (a UpdateHARIActivity) Execute(ctx context.Context, params *UpdateHARIActivityParams) error {
	if params.PipelineName == "" {
		params.PipelineName = "<unnamed>"
	}

	apiURL, err := a.url()
	if err != nil {
		return wferrors.NonRetryableError(fmt.Errorf("error in URL construction: %v", err))
	}

	mock, _ := manager.HookAttrBool(a.manager.Hooks, "hari", "mock")
	if mock {
		ts := a.buildMock()
		defer ts.Close()
		apiURL = ts.URL
	}

	var path = a.avlxml(params.FullPath, params.NameInfo.Type)
	if path == "" {
		return wferrors.NonRetryableError(fmt.Errorf("error reading AVLXML file: not found"))
	}

	blob, err := ioutil.ReadFile(path)
	if err != nil {
		return wferrors.NonRetryableError(fmt.Errorf("error reading AVLXML file: %v", err))
	}

	var parentID string
	{
		if params.NameInfo.Type != nha.TransferTypeAVLXML {
			const idtype = "avleveringsidentifikator"
			parentID, err = readIdentifier(params.FullPath, params.NameInfo.Type.String()+"/journal/avlxml.xml", idtype)
			if err != nil {
				return wferrors.NonRetryableError(fmt.Errorf("error looking up avleveringsidentifikator: %v", err))
			}
		}
	}

	if err := a.sendRequest(ctx, blob, apiURL, params.NameInfo.Type, parentID, params); err != nil {
		return fmt.Errorf("error sending request: %v", err)
	}

	return nil
}

// avlxml attempts to find the AVLXML document in multiple known locations.
func (a UpdateHARIActivity) avlxml(path string, kind nha.TransferType) string {
	firstMatch := func(locs []string) string {
		for _, loc := range locs {
			if stat, err := os.Stat(loc); err == nil && !stat.IsDir() {
				return loc
			}
		}
		return ""
	}

	if kind == nha.TransferTypeAVLXML {
		const objekter = "objekter"
		matches, err := filepath.Glob(filepath.Join(path, kind.String(), objekter, "avlxml-*.xml"))
		if err != nil {
			panic(err)
		}
		if len(matches) > 0 {
			return matches[0]
		}
		return firstMatch([]string{
			filepath.Join(path, kind.String(), objekter, "avlxml.xml"),
		})
	}

	return firstMatch([]string{
		filepath.Join(path, kind.String(), "journal/avlxml.xml"),
		filepath.Join(path, kind.String(), "Journal/avlxml.xml"),
	})
}

// url returns the HARI URL of the API endpoint for AVLXML submission.
func (a UpdateHARIActivity) url() (string, error) {
	p, _ := url.Parse("v1/hari/avlxml")

	b, err := manager.HookAttrString(a.manager.Hooks, "hari", "baseURL")
	if err != nil {
		return "", fmt.Errorf("error looking up baseURL configuration attribute: %v", err)
	}

	if !strings.HasSuffix(b, "/") {
		b = b + "/"
	}

	bu, err := url.Parse(b)
	if err != nil {
		return "", fmt.Errorf("error looking up baseURL configuration attribute: %v", err)
	}

	return bu.ResolveReference(p).String(), nil
}

func (a UpdateHARIActivity) sendRequest(ctx context.Context, blob []byte, apiURL string, kind nha.TransferType, parentID string, params *UpdateHARIActivityParams) error {
	payload := &avlRequest{
		XML:       blob,
		Message:   fmt.Sprintf("AVLXML was processed by Archivematica pipeline %s", params.PipelineName),
		Type:      kind.Lower(),
		Timestamp: avlRequestTime{params.StoredAt},
		AIPID:     params.SIPID,
		Parent:    parentID,
	}

	var buffer bytes.Buffer
	if err := json.NewEncoder(&buffer).Encode(payload); err != nil {
		return fmt.Errorf("error encoding payload: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, &buffer)
	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("User-Agent", "Enduro")

	resp, err := hariClient.Do(req)
	if err != nil {
		return err
	}

	switch {
	case resp.StatusCode >= 200 && resp.StatusCode <= 299:
		err = nil
	default:
		err = fmt.Errorf("unexpected response status: %s", resp.Status)

		// Enrich error message with the payload returned when available.
		payload, rerr := ioutil.ReadAll(resp.Body)
		if rerr == nil && len(payload) > 0 {
			err = fmt.Errorf("(%v) - %s", err, payload)
		}
		defer resp.Body.Close()
	}

	return err
}

// buildMock returns a test server used when HARI's API is not available.
func (a UpdateHARIActivity) buildMock() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		blob, _ := ioutil.ReadAll(r.Body)
		defer r.Body.Close()

		a.manager.Logger.V(1).Info(
			"Request received",
			"method", r.Method,
			"path", r.URL.Path,
			"body", string(blob),
		)

		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"message": "ok"}`)
	}))
}

// avlRequest is the payload of the HTTP request delivered to HARI.
type avlRequest struct {
	XML       []byte         `json:"xml"`              // AVLXML document encoded using base64.
	Message   string         `json:"message"`          // E.g.: "AVLXML was processed by DPJ Archivematica pipeline"
	Type      string         `json:"type"`             // Lowercase. E.g.: "dpj", "epj", "other" or "avlxml".
	Timestamp avlRequestTime `json:"timestamp"`        // E.g.: "2018-11-12T20:20:39+00:00".
	AIPID     string         `json:"aip_id"`           // Typically a UUID.
	Parent    string         `json:"parent,omitempty"` // avleveringsidentifikator (only concerns DPJ and EPJ SIPs)
}

// avlRequestTime encodes time in JSON using the format expected by HARI.
//
// * HARI wants   => "2018-11-12T20:20:39+00:00"
// * time.RFC3339 => "2006-01-02T15:04:05Z07:00"
type avlRequestTime struct {
	time.Time
}

func (t avlRequestTime) MarshalJSON() ([]byte, error) {
	const format = "2006-01-02T15:04:05-07:00"
	var s = fmt.Sprintf("\"%s\"", t.Time.Format(format))
	return []byte(s), nil
}
