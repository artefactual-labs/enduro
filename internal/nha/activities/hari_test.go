package activities

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	temporalsdk_temporal "go.temporal.io/sdk/temporal"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"

	"github.com/artefactual-labs/enduro/internal/nha"
	"github.com/artefactual-labs/enduro/internal/workflow/hooks"
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
<avlxml xmlns:xsi="" xsi:schemaLocation=""><avlxmlversjon></avlxmlversjon><avleveringsidentifikator></avleveringsidentifikator><avleveringsbeskrivelse></avleveringsbeskrivelse><generertdato></generertdato><arkivskaper></arkivskaper><avtale></avtale></avlxml>`)

	tests := map[string]struct {
		// Activity parameters.
		params UpdateHARIActivityParams

		// HARI hook configuration. If baseURL is defined, it overrides the
		// one provided by the test HTTP server.
		hariConfig map[string]any

		// Temporary directory options. Optional.
		dirOpts []fs.PathOp

		// Payload of the wantReceipt that is expected by this test. Optional.
		wantReceipt *avlRequest

		// If non-nil, this will be the status code and status returned by the
		// handler of the fake HTTP server.
		wantResponse *serverResponse

		wantNonRetryableError bool
		wantErr               string
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
			hariConfig: map[string]any{},
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
			hariConfig: map[string]any{},
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
			hariConfig: map[string]any{},
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
			hariConfig: map[string]any{},
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
			hariConfig: map[string]any{},
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
			hariConfig: map[string]any{},
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
			hariConfig: map[string]any{},
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
			hariConfig: map[string]any{"mock": true},
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
			hariConfig: map[string]any{},
			dirOpts: []fs.PathOp{
				fs.WithDir("DPJ/journal"),
				fs.WithFile("DPJ/journal/avlxml.xml", "<avlxml/>"),
			},
			wantNonRetryableError: true,
			wantErr:               "error looking up avleveringsidentifikator: error reading identifier",
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
			hariConfig: map[string]any{},
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
			wantNonRetryableError: true,
			wantErr:               "error looking up avleveringsidentifikator: error reading identifier: not found",
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
			hariConfig: map[string]any{},
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
			wantErr: "error sending request: (unexpected response status: 500 Internal Server Error) - Backend server not available, try again later.\n",
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
			hariConfig: map[string]any{"baseurl": "http://192.168.1.50:12345"},
			dirOpts: []fs.PathOp{
				fs.WithDir("DPJ/journal"),
				fs.WithFile("DPJ/journal/_____other_name_____.xml", "<avlxml/>"),
			},
			wantErr:               "error reading AVLXML file: not found",
			wantNonRetryableError: true,
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
			hariConfig: map[string]any{"baseurl": string([]byte{0x7f})},
			dirOpts: []fs.PathOp{
				fs.WithDir("DPJ/journal"),
				fs.WithFile("DPJ/journal/avlxml.xml", "<avlxml/>"),
			},
			wantErr:               "error in URL construction: error looking up baseURL configuration attribute",
			wantNonRetryableError: true,
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
					blob, err := io.ReadAll(r.Body)
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
				if _, ok := tc.hariConfig["baseurl"]; !ok {
					tc.hariConfig["baseurl"] = deliveree.URL
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

			testError(t, err, tc.wantErr, tc.wantNonRetryableError)
		})
	}
}

func testError(t *testing.T, err error, wantErr string, wantNonRetryable bool) {
	t.Helper()

	if wantErr == "" {
		assert.NilError(t, err)
		return
	}

	assert.ErrorContains(t, err, wantErr)

	if !wantNonRetryable {
		return
	}
	var appError *temporalsdk_temporal.ApplicationError
	if errors.As(err, &appError) && !appError.NonRetryable() {
		t.Fatal("error is not non-retryable")
	}
}

func createHariActivity(t *testing.T, hariConfig map[string]any) *UpdateHARIActivity {
	t.Helper()

	hooks := hooks.NewHooks(
		map[string]map[string]any{
			"hari": hariConfig,
		},
	)

	return NewUpdateHARIActivity(hooks)
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
		act := createHariActivity(t, map[string]any{
			"baseurl": tc.baseURL,
		})

		have, err := act.url()
		assert.NilError(t, err)
		assert.Equal(t, have, tc.wantURL)
	}
}
