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

	"github.com/go-logr/logr"
	"github.com/golang/mock/gomock"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"

	collectionfake "github.com/artefactual-labs/enduro/internal/collection/fake"
	"github.com/artefactual-labs/enduro/internal/nha"
	"github.com/artefactual-labs/enduro/internal/pipeline"
	"github.com/artefactual-labs/enduro/internal/testutil"
	watcherfake "github.com/artefactual-labs/enduro/internal/watcher/fake"
	"github.com/artefactual-labs/enduro/internal/workflow/manager"
)

type serverResponse struct {
	code   int
	status string
}

func TestHARIActivity(t *testing.T) {
	t.Parallel()

	// Tweak the client so we don't have to wait for too long.
	hariClient.Timeout = time.Second * 1

	// When slimDown is used.
	emptyavlxml := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<avlxml xmlns:xsi="" xsi:schemaLocation=""><avlxmlversjon></avlxmlversjon><avleveringsidentifikator></avleveringsidentifikator><avleveringsbeskrivelse></avleveringsbeskrivelse><arkivskaper></arkivskaper><avtale></avtale></avlxml>`)

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
		wantErr testutil.ActivityError
	}{
		"Receipt is delivered successfully (DPJ)": {
			params: UpdateHARIActivityParams{
				StoredAt:     time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
				SIPID:        "1db240cc-3cea-4e55-903c-6280562e1866",
				PipelineName: "zr-fig-pipe-001",
				NameInfo: nha.NameInfo{
					Identifier: "049d6a44-07d6-4aa9-9607-9347ec4d0b23",
					Type:       nha.TransferTypeDPJ,
				},
			},
			hariConfig: map[string]interface{}{},
			dirOpts: []fs.PathOp{
				fs.WithDir("DPJ/journal"),
				// XML generated is a trimmed-down version, e.g. `pasientjournal` not included.
				fs.WithFile("DPJ/journal/avlxml.xml", "<avlxml/>"),
				fs.WithDir("metadata"),
				fs.WithFile("metadata/identifiers.json", `[{
					"file": "objects/DPJ/aFoobar.jpg",
					"identifiers": [{
						"identifierType": "organisasjonsnummer",
						"identifier": "123456789"
					}]
				}, {
					"file": "objects/DPJ/journal/avlxml.xml",
					"identifiers": [{
						"identifierType": "organisasjonsnummer",
						"identifier": "123456789"
					}, {
						"identifierType": "avleveringsidentifikator",
						"identifier": "12345"
					}]
				}]`),
			},
			wantReceipt: &avlRequest{
				Message:   "AVLXML was processed by Archivematica pipeline zr-fig-pipe-001",
				Type:      "dpj",
				Timestamp: avlRequestTime{time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)},
				AIPID:     "1db240cc-3cea-4e55-903c-6280562e1866",
				Parent:    "12345",
				XML:       []byte("<avlxml/>"),
			},
		},
		"Receipt is delivered successfully (EPJ)": {
			params: UpdateHARIActivityParams{
				StoredAt:     time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
				SIPID:        "1db240cc-3cea-4e55-903c-6280562e1866",
				PipelineName: "zr-fig-pipe-001",
				NameInfo: nha.NameInfo{
					Identifier: "049d6a44-07d6-4aa9-9607-9347ec4d0b23",
					Type:       nha.TransferTypeEPJ,
				},
			},
			hariConfig: map[string]interface{}{},
			dirOpts: []fs.PathOp{
				fs.WithDir("EPJ/journal"),
				fs.WithFile("EPJ/journal/avlxml.xml", "<avlxml/>"),
				fs.WithDir("metadata"),
				fs.WithFile("metadata/identifiers.json", `[{
					"file": "objects/EPJ/journal/avlxml.xml",
					"identifiers": [{
						"identifierType": "avleveringsidentifikator",
						"identifier": "12345"
					}]
				}]`),
			},
			wantReceipt: &avlRequest{
				Message:   "AVLXML was processed by Archivematica pipeline zr-fig-pipe-001",
				Type:      "epj",
				Timestamp: avlRequestTime{time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)},
				AIPID:     "1db240cc-3cea-4e55-903c-6280562e1866",
				Parent:    "12345",
				XML:       []byte("<avlxml/>"),
			},
		},
		"Receipt is delivered successfully (AVLXML)": {
			params: UpdateHARIActivityParams{
				StoredAt:     time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
				SIPID:        "1db240cc-3cea-4e55-903c-6280562e1866",
				PipelineName: "zr-fig-pipe-001",
				NameInfo: nha.NameInfo{
					Identifier: "2.16.578.1.39.100.11.9876.4",
					Type:       nha.TransferTypeAVLXML,
				},
			},
			hariConfig: map[string]interface{}{},
			dirOpts: []fs.PathOp{
				fs.WithDir("AVLXML/objekter"),
				fs.WithFile(
					// Including pasientjournal since we want to test that is removed.
					"AVLXML/objekter/avlxml-2.16.578.1.39.100.11.9876.4-20191104.xml",
					"<avlxml><pasientjournal>12345</pasientjournal></avlxml>",
				),
			},
			wantReceipt: &avlRequest{
				Message:   "AVLXML was processed by Archivematica pipeline zr-fig-pipe-001",
				Type:      "avlxml",
				Timestamp: avlRequestTime{time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)},
				AIPID:     "1db240cc-3cea-4e55-903c-6280562e1866",
				XML:       emptyavlxml,
			},
		},
		"Receipt is delivered successfully (AVLXML alt.)": {
			params: UpdateHARIActivityParams{
				StoredAt:     time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
				SIPID:        "1db240cc-3cea-4e55-903c-6280562e1866",
				PipelineName: "zr-fig-pipe-001",
				NameInfo: nha.NameInfo{
					Identifier: "049d6a44-07d6-4aa9-9607-9347ec4d0b23",
					Type:       nha.TransferTypeAVLXML,
				},
			},
			hariConfig: map[string]interface{}{},
			dirOpts: []fs.PathOp{
				fs.WithDir("AVLXML/objekter"),
				fs.WithFile(
					// Including pasientjournal since we want to test that is removed.
					"AVLXML/objekter/avlxml-2.16.578.1.39.100.11.9876.4-20191104.xml",
					"<avlxml><pasientjournal>12345</pasientjournal></avlxml>",
				),
			},
			wantReceipt: &avlRequest{
				Message:   "AVLXML was processed by Archivematica pipeline zr-fig-pipe-001",
				Type:      "avlxml",
				Timestamp: avlRequestTime{time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)},
				AIPID:     "1db240cc-3cea-4e55-903c-6280562e1866",
				XML:       emptyavlxml,
			},
		},
		"Receipt is delivered successfully (OTHER)": {
			params: UpdateHARIActivityParams{
				StoredAt:     time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
				SIPID:        "1db240cc-3cea-4e55-903c-6280562e1866",
				PipelineName: "zr-fig-pipe-001",
				NameInfo: nha.NameInfo{
					Identifier: "049d6a44-07d6-4aa9-9607-9347ec4d0b23",
					Type:       nha.TransferTypeOther,
				},
			},
			hariConfig: map[string]interface{}{},
			dirOpts: []fs.PathOp{
				fs.WithDir("OTHER/journal"),
				fs.WithFile("OTHER/journal/avlxml.xml", "<avlxml/>"),
				fs.WithDir("metadata"),
				fs.WithFile("metadata/identifiers.json", `[{
					"file": "objects/OTHER/journal/avlxml.xml",
					"identifiers": [{
						"identifierType": "avleveringsidentifikator",
						"identifier": "12345"
					}]
				}]`),
			},
			wantReceipt: &avlRequest{
				Message:   "AVLXML was processed by Archivematica pipeline zr-fig-pipe-001",
				Type:      "other",
				Timestamp: avlRequestTime{time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)},
				AIPID:     "1db240cc-3cea-4e55-903c-6280562e1866",
				Parent:    "12345",
				XML:       []byte("<avlxml/>"),
			},
		},
		"Capital letter in journal directory is reached": {
			params: UpdateHARIActivityParams{
				StoredAt:     time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
				SIPID:        "1db240cc-3cea-4e55-903c-6280562e1866",
				PipelineName: "zr-fig-pipe-001",
				NameInfo: nha.NameInfo{
					Identifier: "049d6a44-07d6-4aa9-9607-9347ec4d0b23",
					Type:       nha.TransferTypeDPJ,
				},
			},
			hariConfig: map[string]interface{}{},
			dirOpts: []fs.PathOp{
				fs.WithDir("DPJ/Journal"),
				fs.WithFile("DPJ/Journal/avlxml.xml", "<avlxml/>"),
				fs.WithDir("metadata"),
				fs.WithFile("metadata/identifiers.json", `[{
					"file": "objects/DPJ/Journal/avlxml.xml",
					"identifiers": [{
						"identifierType": "avleveringsidentifikator",
						"identifier": "12345"
					}]
				}]`),
			},
			wantReceipt: &avlRequest{
				Message:   "AVLXML was processed by Archivematica pipeline zr-fig-pipe-001",
				Type:      "dpj",
				Timestamp: avlRequestTime{time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)},
				AIPID:     "1db240cc-3cea-4e55-903c-6280562e1866",
				Parent:    "12345",
				XML:       []byte("<avlxml/>"),
			},
		},
		"Lowercase kind attribute is handled successfully": {
			params: UpdateHARIActivityParams{
				StoredAt:     time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
				SIPID:        "1db240cc-3cea-4e55-903c-6280562e1866",
				PipelineName: "zr-fig-pipe-001",
				NameInfo: nha.NameInfo{
					Identifier: "049d6a44-07d6-4aa9-9607-9347ec4d0b23",
					Type:       nha.TransferTypeDPJ,
				},
			},
			hariConfig: map[string]interface{}{},
			dirOpts: []fs.PathOp{
				fs.WithDir("DPJ/journal"),
				fs.WithFile("DPJ/journal/avlxml.xml", "<avlxml/>"),
				fs.WithDir("metadata"),
				fs.WithFile("metadata/identifiers.json", `[{
					"file": "objects/DPJ/journal/avlxml.xml",
					"identifiers": [{
						"identifierType": "avleveringsidentifikator",
						"identifier": "12345"
					}]
				}]`),
			},
			wantReceipt: &avlRequest{
				Message:   "AVLXML was processed by Archivematica pipeline zr-fig-pipe-001",
				Type:      "dpj",
				Timestamp: avlRequestTime{time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)},
				AIPID:     "1db240cc-3cea-4e55-903c-6280562e1866",
				Parent:    "12345",
				XML:       []byte("<avlxml/>"),
			},
		},
		"Mock option is honoured": {
			params: UpdateHARIActivityParams{
				StoredAt:     time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
				SIPID:        "1db240cc-3cea-4e55-903c-6280562e1866",
				PipelineName: "zr-fig-pipe-001",
				NameInfo: nha.NameInfo{
					Identifier: "049d6a44-07d6-4aa9-9607-9347ec4d0b23",
					Type:       nha.TransferTypeDPJ,
				},
			},
			hariConfig: map[string]interface{}{"mock": true},
			dirOpts: []fs.PathOp{
				fs.WithDir("DPJ/journal"),
				fs.WithFile("DPJ/journal/avlxml.xml", "<avlxml/>"),
				fs.WithDir("metadata"),
				fs.WithFile("metadata/identifiers.json", `[{
					"file": "objects/DPJ/journal/avlxml.xml",
					"identifiers": [{
						"identifierType": "avleveringsidentifikator",
						"identifier": "12345"
					}]
				}]`),
			},
		},
		"Failure when identifiers.json is missing (DPJ/EPJ/OTHER)": {
			params: UpdateHARIActivityParams{
				StoredAt:     time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
				SIPID:        "1db240cc-3cea-4e55-903c-6280562e1866",
				PipelineName: "zr-fig-pipe-001",
				NameInfo: nha.NameInfo{
					Identifier: "049d6a44-07d6-4aa9-9607-9347ec4d0b23",
					Type:       nha.TransferTypeDPJ,
				},
			},
			hariConfig: map[string]interface{}{},
			dirOpts: []fs.PathOp{
				fs.WithDir("DPJ/journal"),
				fs.WithFile("DPJ/journal/avlxml.xml", "<avlxml/>"),
			},
			wantErr: testutil.ActivityError{
				NRE: true,
			},
		},
		"Failure when identifier cannot be found (DPJ/EPJ/OTHER)": {
			params: UpdateHARIActivityParams{
				StoredAt:     time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
				SIPID:        "1db240cc-3cea-4e55-903c-6280562e1866",
				PipelineName: "zr-fig-pipe-001",
				NameInfo: nha.NameInfo{
					Identifier: "049d6a44-07d6-4aa9-9607-9347ec4d0b23",
					Type:       nha.TransferTypeDPJ,
				},
			},
			hariConfig: map[string]interface{}{},
			dirOpts: []fs.PathOp{
				fs.WithDir("DPJ/journal"),
				fs.WithFile("DPJ/journal/avlxml.xml", "<avlxml/>"),
				fs.WithDir("metadata"),
				fs.WithFile("metadata/identifiers.json", `[{
					"file": "objects/DPJ/journal/avlxml.xml",
					"identifiers": [{
						"identifierType": "not-the-identifier-we-wanted",
						"identifier": "12345"
					}]
				}]`),
			},
			wantErr: testutil.ActivityError{
				Message: "error looking up avleveringsidentifikator: error reading identifier: not found",
				NRE:     true,
			},
		},
		"Failure when HARI returns a server error": {
			params: UpdateHARIActivityParams{
				StoredAt:     time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
				SIPID:        "1db240cc-3cea-4e55-903c-6280562e1866",
				PipelineName: "zr-fig-pipe-001",
				NameInfo: nha.NameInfo{
					Identifier: "049d6a44-07d6-4aa9-9607-9347ec4d0b23",
					Type:       nha.TransferTypeDPJ,
				},
			},
			hariConfig: map[string]interface{}{},
			dirOpts: []fs.PathOp{
				fs.WithDir("DPJ/journal"),
				fs.WithFile("DPJ/journal/avlxml.xml", "<avlxml/>"),
				fs.WithDir("metadata"),
				fs.WithFile("metadata/identifiers.json", `[{
					"file": "objects/DPJ/journal/avlxml.xml",
					"identifiers": [{
						"identifierType": "avleveringsidentifikator",
						"identifier": "12345"
					}]
				}]`),
			},
			wantResponse: &serverResponse{code: 500, status: "Backend server not available, try again later."},
			wantReceipt: &avlRequest{
				Message:   "AVLXML was processed by Archivematica pipeline zr-fig-pipe-001",
				Type:      "dpj",
				Timestamp: avlRequestTime{time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)},
				AIPID:     "1db240cc-3cea-4e55-903c-6280562e1866",
				Parent:    "12345",
				XML:       []byte("<avlxml/>"),
			},
			wantErr: testutil.ActivityError{
				Message: "error sending request: (unexpected response status: 500 Internal Server Error) - Backend server not available, try again later.\n",
				NRE:     false,
			},
		},
		"Unexisten AVLXML file causes error": {
			params: UpdateHARIActivityParams{
				StoredAt: time.Now(),
				SIPID:    "1db240cc-3cea-4e55-903c-6280562e1866",
				NameInfo: nha.NameInfo{
					Identifier: "049d6a44-07d6-4aa9-9607-9347ec4d0b23",
					Type:       nha.TransferTypeDPJ,
				},
			},
			hariConfig: map[string]interface{}{"baseURL": "http://192.168.1.50:12345"},
			dirOpts: []fs.PathOp{
				fs.WithDir("DPJ/journal"),
				fs.WithFile("DPJ/journal/_____other_name_____.xml", "<avlxml/>"),
			},
			wantErr: testutil.ActivityError{
				Message: "error reading AVLXML file: not found",
				NRE:     true,
			},
		},
		"Unparseable baseURL is rejected": {
			params: UpdateHARIActivityParams{
				StoredAt: time.Now(),
				SIPID:    "1db240cc-3cea-4e55-903c-6280562e1866",
				NameInfo: nha.NameInfo{
					Identifier: "049d6a44-07d6-4aa9-9607-9347ec4d0b23",
					Type:       nha.TransferTypeDPJ,
				},
			},
			hariConfig: map[string]interface{}{"baseURL": string([]byte{0x7f})},
			dirOpts: []fs.PathOp{
				fs.WithDir("DPJ/journal"),
				fs.WithFile("DPJ/journal/avlxml.xml", "<avlxml/>"),
			},
			wantErr: testutil.ActivityError{
				NRE: true,
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
					http.Error(w, tc.wantResponse.status, tc.wantResponse.code)
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
		logr.Discard(),
		collectionfake.NewMockService(ctrl),
		watcherfake.NewMockService(ctrl),
		&pipeline.Registry{},
		hooks,
	)

	return NewUpdateHARIActivity(manager)
}

func TestHARIURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		baseURL string
		wantURL string
	}{
		{
			baseURL: "http://domain.tld/api/",
			wantURL: "http://domain.tld/api/v1/hari/avlxml",
		},
		{
			baseURL: "http://domain.tld/foobar/api/",
			wantURL: "http://domain.tld/foobar/api/v1/hari/avlxml",
		},
		{
			baseURL: "https://domain.tld:12345/api",
			wantURL: "https://domain.tld:12345/api/v1/hari/avlxml",
		},
	}
	for _, tc := range tests {
		act := createHariActivity(t, map[string]interface{}{
			"baseURL": tc.baseURL,
		})

		have, err := act.url()
		assert.NilError(t, err)
		assert.Equal(t, have, tc.wantURL)
	}
}
