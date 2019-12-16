package activities

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	logrt "github.com/go-logr/logr/testing"
	"github.com/golang/mock/gomock"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"

	collectionfake "github.com/artefactual-labs/enduro/internal/collection/fake"
	"github.com/artefactual-labs/enduro/internal/pipeline"
	watcherfake "github.com/artefactual-labs/enduro/internal/watcher/fake"
	"github.com/artefactual-labs/enduro/internal/workflow/manager"
)

type serverResponse struct {
	code   int
	status string
}

func TestTable(t *testing.T) {
	t.Parallel()

	// Tweak the client so we don't have to wait for too long.
	hariClient.Timeout = time.Second * 1

	tests := map[string]struct {
		// Activity parameters.
		params UpdateHARIActivityParams

		// HARI hook configuration. If baseURL is defined, it overrides the
		// one provided by the test HTTP server.
		hariConfig map[string]interface{}

		// Temporary directory options. Optional.
		dirOpts []fs.PathOp

		// Payload of the wantReceipt that is expected by this test. Optional.
		wantReceipt *avlRequest

		// If non-nil, this will be the status code and status returned by the
		// handler of the fake HTTP server.
		wantResponse *serverResponse

		// Expected error: see activityError for more.
		wantErr activityError
	}{
		"Receipt is delivered successfully (DPJ)": {
			params: UpdateHARIActivityParams{
				StoredAt:     time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
				SIPID:        "1db240cc-3cea-4e55-903c-6280562e1866",
				Name:         "DPJ-SIP-049d6a44-07d6-4aa9-9607-9347ec4d0b23",
				PipelineName: "zr-fig-pipe-001",
			},
			hariConfig: map[string]interface{}{},
			dirOpts:    []fs.PathOp{fs.WithDir("DPJ/journal"), fs.WithFile("DPJ/journal/avlxml.xml", "<xml/>")},
			wantReceipt: &avlRequest{
				Message:   "AVLXML was processed by Archivematica pipeline zr-fig-pipe-001",
				Type:      "dpj",
				Timestamp: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
				AIPID:     "1db240cc-3cea-4e55-903c-6280562e1866",
				XML:       []byte(`<xml/>`),
			},
		},
		"Receipt is delivered successfully (EPJ)": {
			params: UpdateHARIActivityParams{
				StoredAt:     time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
				SIPID:        "1db240cc-3cea-4e55-903c-6280562e1866",
				Name:         "EPJ-SIP-049d6a44-07d6-4aa9-9607-9347ec4d0b23",
				PipelineName: "zr-fig-pipe-001",
			},
			hariConfig: map[string]interface{}{},
			dirOpts:    []fs.PathOp{fs.WithDir("EPJ/journal"), fs.WithFile("EPJ/journal/avlxml.xml", "<xml/>")},
			wantReceipt: &avlRequest{
				Message:   "AVLXML was processed by Archivematica pipeline zr-fig-pipe-001",
				Type:      "epj",
				Timestamp: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
				AIPID:     "1db240cc-3cea-4e55-903c-6280562e1866",
				XML:       []byte(`<xml/>`),
			},
		},
		"Receipt is delivered successfully (AVLXML)": {
			params: UpdateHARIActivityParams{
				StoredAt:     time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
				SIPID:        "1db240cc-3cea-4e55-903c-6280562e1866",
				Name:         "AVLXML-SIP-049d6a44-07d6-4aa9-9607-9347ec4d0b23",
				PipelineName: "zr-fig-pipe-001",
			},
			hariConfig: map[string]interface{}{},
			dirOpts:    []fs.PathOp{fs.WithDir("AVLXML/objekter"), fs.WithFile("AVLXML/objekter/avlxml-2.16.578.1.39.100.11.9876.4-20191104.xml", "<xml/>")},
			wantReceipt: &avlRequest{
				Message:   "AVLXML was processed by Archivematica pipeline zr-fig-pipe-001",
				Type:      "avlxml",
				Timestamp: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
				AIPID:     "1db240cc-3cea-4e55-903c-6280562e1866",
				XML:       []byte(`<xml/>`),
			},
		},
		"Receipt is delivered successfully (AVLXML alt.)": {
			params: UpdateHARIActivityParams{
				StoredAt:     time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
				SIPID:        "1db240cc-3cea-4e55-903c-6280562e1866",
				Name:         "AVLXML-SIP-049d6a44-07d6-4aa9-9607-9347ec4d0b23",
				PipelineName: "zr-fig-pipe-001",
			},
			hariConfig: map[string]interface{}{},
			dirOpts:    []fs.PathOp{fs.WithDir("AVLXML/objekter"), fs.WithFile("AVLXML/objekter/avlxml.xml", "<xml/>")},
			wantReceipt: &avlRequest{
				Message:   "AVLXML was processed by Archivematica pipeline zr-fig-pipe-001",
				Type:      "avlxml",
				Timestamp: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
				AIPID:     "1db240cc-3cea-4e55-903c-6280562e1866",
				XML:       []byte(`<xml/>`),
			},
		},
		"Receipt is delivered successfully (OTHER)": {
			params: UpdateHARIActivityParams{
				StoredAt:     time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
				SIPID:        "1db240cc-3cea-4e55-903c-6280562e1866",
				Name:         "OTHER-SIP-049d6a44-07d6-4aa9-9607-9347ec4d0b23",
				PipelineName: "zr-fig-pipe-001",
			},
			hariConfig: map[string]interface{}{},
			dirOpts:    []fs.PathOp{fs.WithDir("OTHER/journal"), fs.WithFile("OTHER/journal/avlxml.xml", "<xml/>")},
			wantReceipt: &avlRequest{
				Message:   "AVLXML was processed by Archivematica pipeline zr-fig-pipe-001",
				Type:      "other",
				Timestamp: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
				AIPID:     "1db240cc-3cea-4e55-903c-6280562e1866",
				XML:       []byte(`<xml/>`),
			},
		},
		"Capital letter in journal directory is reached": {
			params: UpdateHARIActivityParams{
				StoredAt:     time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
				SIPID:        "1db240cc-3cea-4e55-903c-6280562e1866",
				Name:         "DPJ-SIP-049d6a44-07d6-4aa9-9607-9347ec4d0b23",
				PipelineName: "zr-fig-pipe-001",
			},
			hariConfig: map[string]interface{}{},
			dirOpts:    []fs.PathOp{fs.WithDir("DPJ/Journal"), fs.WithFile("DPJ/Journal/avlxml.xml", "<xml/>")},
			wantReceipt: &avlRequest{
				Message:   "AVLXML was processed by Archivematica pipeline zr-fig-pipe-001",
				Type:      "dpj",
				Timestamp: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
				AIPID:     "1db240cc-3cea-4e55-903c-6280562e1866",
				XML:       []byte(`<xml/>`),
			},
		},
		"Lowercase kind attribute is handled successfully": {
			params: UpdateHARIActivityParams{
				StoredAt:     time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
				SIPID:        "1db240cc-3cea-4e55-903c-6280562e1866",
				Name:         "dpj-sip-049d6a44-07d6-4aa9-9607-9347ec4d0b23",
				PipelineName: "zr-fig-pipe-001",
			},
			hariConfig: map[string]interface{}{},
			dirOpts:    []fs.PathOp{fs.WithDir("DPJ/journal"), fs.WithFile("DPJ/journal/avlxml.xml", "<xml/>")},
			wantReceipt: &avlRequest{
				Message:   "AVLXML was processed by Archivematica pipeline zr-fig-pipe-001",
				Type:      "dpj",
				Timestamp: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
				AIPID:     "1db240cc-3cea-4e55-903c-6280562e1866",
				XML:       []byte(`<xml/>`),
			},
		},
		"Mock option is honoured": {
			params: UpdateHARIActivityParams{
				StoredAt:     time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
				SIPID:        "1db240cc-3cea-4e55-903c-6280562e1866",
				Name:         "DPJ-SIP-049d6a44-07d6-4aa9-9607-9347ec4d0b23",
				PipelineName: "zr-fig-pipe-001",
			},
			hariConfig: map[string]interface{}{"mock": true},
			dirOpts:    []fs.PathOp{fs.WithDir("DPJ/journal"), fs.WithFile("DPJ/journal/avlxml.xml", "<xml/>")},
		},
		"Failure when HARI returns a server error": {
			params: UpdateHARIActivityParams{
				StoredAt:     time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
				SIPID:        "1db240cc-3cea-4e55-903c-6280562e1866",
				Name:         "DPJ-SIP-049d6a44-07d6-4aa9-9607-9347ec4d0b23",
				PipelineName: "zr-fig-pipe-001",
			},
			hariConfig:   map[string]interface{}{},
			dirOpts:      []fs.PathOp{fs.WithDir("DPJ/journal"), fs.WithFile("DPJ/journal/avlxml.xml", "<xml/>")},
			wantResponse: &serverResponse{code: 500, status: "Backend server not available, try again later."},
			wantReceipt: &avlRequest{
				Message:   "AVLXML was processed by Archivematica pipeline zr-fig-pipe-001",
				Type:      "dpj",
				Timestamp: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
				AIPID:     "1db240cc-3cea-4e55-903c-6280562e1866",
				XML:       []byte(`<xml/>`),
			},
			wantErr: activityError{
				Message: "error sending request: unexpected response status: 500 Internal Server Error",
				NRE:     false,
			},
		},
		"Empty kind is rejected": {
			params: UpdateHARIActivityParams{
				StoredAt: time.Now(),
				SIPID:    "1db240cc-3cea-4e55-903c-6280562e1866",
				Name:     "",
			},
			wantErr: activityError{
				Message: "Name is missing or empty",
				NRE:     true,
			},
		},
		"Unsuffixed kind is rejected": {
			params: UpdateHARIActivityParams{
				StoredAt: time.Now(),
				SIPID:    "1db240cc-3cea-4e55-903c-6280562e1866",
				Name:     "DPJ-049d6a44-07d6-4aa9-9607-9347ec4d0b23",
			},
			wantErr: activityError{
				Message: "error extracting kind attribute: attribute (DPJ) does not containt suffix (\"-SIP\")",
				NRE:     true,
			},
		},
		"Unknown kind is rejected": {
			params: UpdateHARIActivityParams{
				StoredAt: time.Now(),
				SIPID:    "1db240cc-3cea-4e55-903c-6280562e1866",
				Name:     "FOOBAR-SIP-049d6a44-07d6-4aa9-9607-9347ec4d0b23",
			},
			wantErr: activityError{
				Message: "error extracting kind attribute: attribute (FOOBAR) is unexpected/unknown",
				NRE:     true,
			},
		},
		"Unexisten AVLXML file causes error": {
			params: UpdateHARIActivityParams{
				StoredAt: time.Now(),
				SIPID:    "1db240cc-3cea-4e55-903c-6280562e1866",
				Name:     "DPJ-SIP-049d6a44-07d6-4aa9-9607-9347ec4d0b23",
			},
			hariConfig: map[string]interface{}{"baseURL": "http://192.168.1.50:12345"},
			dirOpts:    []fs.PathOp{fs.WithDir("DPJ/journal"), fs.WithFile("DPJ/journal/_____other_name_____.xml", "<xml/>")},
			wantErr: activityError{
				Message: "error reading AVLXML file: not found",
				NRE:     true,
			},
		},
		"Unparseable baseURL is rejected": {
			params: UpdateHARIActivityParams{
				StoredAt: time.Now(),
				SIPID:    "1db240cc-3cea-4e55-903c-6280562e1866",
				Name:     "DPJ-SIP-049d6a44-07d6-4aa9-9607-9347ec4d0b23",
			},
			hariConfig: map[string]interface{}{"baseURL": string([]byte{0x7f})},
			dirOpts:    []fs.PathOp{fs.WithDir("DPJ/journal"), fs.WithFile("DPJ/journal/avlxml.xml", "<xml/>")},
			wantErr: activityError{
				Message: "error in URL construction: error looking up baseURL configuration attribute: parse : net/url: invalid control character in URL",
				NRE:     true,
			},
		},
	}

	for name, tc := range tests {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Test our receipt from a fake HTTP server.
			deliveree := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, r.Method, http.MethodPost)
				assert.Equal(t, r.URL.Path, "/v1/hari/avlxml")
				assert.Equal(t, r.Header.Get("Content-Type"), "application/json")
				assert.Equal(t, r.Header.Get("User-Agent"), "Enduro")

				if tc.wantReceipt != nil {
					blob, err := ioutil.ReadAll(r.Body)
					assert.NilError(t, err)
					defer r.Body.Close()

					want, have := tc.wantReceipt, &avlRequest{}
					assert.NilError(t, json.Unmarshal(blob, have))
					assert.DeepEqual(t, want, have)
				}

				if tc.wantResponse != nil {
					w.WriteHeader(tc.wantResponse.code)
					w.Write([]byte(tc.wantResponse.status))
					return
				}

				w.WriteHeader(http.StatusOK)
				fmt.Fprintf(w, `{"message": "ok"}`)
			}))

			// Only override baseURL when the test case did not define it.
			if tc.hariConfig != nil {
				if _, ok := tc.hariConfig["baseURL"]; !ok {
					tc.hariConfig["baseURL"] = deliveree.URL
				}
			}

			act := createHariActivity(t, tc.hariConfig)

			if tc.dirOpts != nil {
				tmpdir := fs.NewDir(t, "enduro", tc.dirOpts...)
				defer tmpdir.Remove()

				if tc.params.FullPath == "" {
					tc.params.FullPath = tmpdir.Path()
				}
			}

			err := act.Execute(context.Background(), &tc.params)

			tc.wantErr.Assert(t, err)
		})
	}
}

func createHariActivity(t *testing.T, hariConfig map[string]interface{}) *UpdateHARIActivity {
	t.Helper()

	ctrl := gomock.NewController(t)

	hooks := map[string]map[string]interface{}{
		"hari": hariConfig,
	}

	manager := manager.NewManager(
		logrt.NullLogger{},
		collectionfake.NewMockService(ctrl),
		watcherfake.NewMockService(ctrl),
		&pipeline.Registry{},
		hooks,
	)

	return NewUpdateHARIActivity(manager)
}

func TestExtractKind(t *testing.T) {
	t.Parallel()

	tests := []struct {
		key            string
		wantKind       string
		wantErr        bool
		wantErrMessage string
	}{
		{
			key:            "",
			wantKind:       "",
			wantErr:        true,
			wantErrMessage: "unexpected format",
		},
		{
			key:            "foobar.jpg",
			wantKind:       "",
			wantErr:        true,
			wantErrMessage: "unexpected format",
		},
		{
			key:            "c5ecddb0-7a61-4234-80a9-fa7993e97867.tar",
			wantKind:       "",
			wantErr:        true,
			wantErrMessage: "unexpected format",
		},
		{
			key:            "dpj-sip-12345",
			wantKind:       "",
			wantErr:        true,
			wantErrMessage: "unexpected format",
		},
		{
			key:            "dpj-c5ecddb0-7a61-4234-80a9-fa7993e97867",
			wantKind:       "",
			wantErr:        true,
			wantErrMessage: "attribute (DPJ) does not containt suffix (\"-SIP\")",
		},
		{
			key:            "unknown-sip-c5ecddb0-7a61-4234-80a9-fa7993e97867.tar",
			wantKind:       "",
			wantErr:        true,
			wantErrMessage: "attribute (UNKNOWN) is unexpected/unknown",
		},
		{
			key:      "dpj-sip_c5ecddb0-7a61-4234-80a9-fa7993e97867.tar",
			wantKind: "DPJ",
			wantErr:  false,
		},
		{
			key:      "dpj-sip-c5ecddb0-7a61-4234-80a9-fa7993e97867.tar",
			wantKind: "DPJ",
			wantErr:  false,
		},
		{
			key:      "dpj-sip-c5ecddb0-7a61-4234-80a9-fa7993e97867.tar",
			wantKind: "DPJ",
			wantErr:  false,
		},
		{
			key:      "epj-sip-c5ecddb0-7a61-4234-80a9-fa7993e97867.tar",
			wantKind: "EPJ",
			wantErr:  false,
		},
		{
			key:      "avlxml-sip-c5ecddb0-7a61-4234-80a9-fa7993e97867.tar",
			wantKind: "AVLXML",
			wantErr:  false,
		},
		{
			key:      "other-sip-c5ecddb0-7a61-4234-80a9-fa7993e97867.tar",
			wantKind: "OTHER",
			wantErr:  false,
		},
	}
	for _, tc := range tests {
		kind, err := extractKind(tc.key)

		assert.Equal(t, kind, tc.wantKind)

		if tc.wantErr {
			assert.Error(t, err, tc.wantErrMessage)
		} else {
			assert.NilError(t, err)
		}
	}
}
