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
	"go.uber.org/cadence"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"

	collectionfake "github.com/artefactual-labs/enduro/internal/collection/fake"
	"github.com/artefactual-labs/enduro/internal/pipeline"
	watcherfake "github.com/artefactual-labs/enduro/internal/watcher/fake"
	wferrors "github.com/artefactual-labs/enduro/internal/workflow/errors"
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

		// Payload of the receipt that is expected by this test. Optional.
		receipt *avlRequest

		// If non-nil, this will be the status code and status returned by the
		// handler of the fake HTTP server.
		response *serverResponse

		// Expected error: reason + details. Optional.
		//
		// If reason == NRE (non retryable error), we're confirming that the
		// workflow gives up right away (as long as the retry policy attached
		// is set up properly). details are only compared when NRE is used as
		// it's a parameter specific to cadence.CustomError.
		err []string
	}{
		"Receipt is delivered successfully (DPJ)": {
			params: UpdateHARIActivityParams{
				StoredAt:     time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
				SIPID:        "1db240cc-3cea-4e55-903c-6280562e1866",
				Kind:         "DPJ-SIP",
				PipelineName: "zr-fig-pipe-001",
			},
			hariConfig: map[string]interface{}{},
			dirOpts:    []fs.PathOp{fs.WithDir("DPJ/journal"), fs.WithFile("DPJ/journal/avlxml.xml", "<xml/>")},
			receipt: &avlRequest{
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
				Kind:         "EPJ-SIP",
				PipelineName: "zr-fig-pipe-001",
			},
			hariConfig: map[string]interface{}{},
			dirOpts:    []fs.PathOp{fs.WithDir("EPJ/journal"), fs.WithFile("EPJ/journal/avlxml.xml", "<xml/>")},
			receipt: &avlRequest{
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
				Kind:         "AVLXML-SIP",
				PipelineName: "zr-fig-pipe-001",
			},
			hariConfig: map[string]interface{}{},
			dirOpts:    []fs.PathOp{fs.WithDir("AVLXML/journal"), fs.WithFile("AVLXML/journal/avlxml.xml", "<xml/>")},
			receipt: &avlRequest{
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
				Kind:         "OTHER-SIP",
				PipelineName: "zr-fig-pipe-001",
			},
			hariConfig: map[string]interface{}{},
			dirOpts:    []fs.PathOp{fs.WithDir("OTHER/journal"), fs.WithFile("OTHER/journal/avlxml.xml", "<xml/>")},
			receipt: &avlRequest{
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
				Kind:         "DPJ-SIP",
				PipelineName: "zr-fig-pipe-001",
			},
			hariConfig: map[string]interface{}{},
			dirOpts:    []fs.PathOp{fs.WithDir("DPJ/Journal"), fs.WithFile("DPJ/Journal/avlxml.xml", "<xml/>")},
			receipt: &avlRequest{
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
				Kind:         "dpj-sip",
				PipelineName: "zr-fig-pipe-001",
			},
			hariConfig: map[string]interface{}{},
			dirOpts:    []fs.PathOp{fs.WithDir("DPJ/journal"), fs.WithFile("DPJ/journal/avlxml.xml", "<xml/>")},
			receipt: &avlRequest{
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
				Kind:         "DPJ-SIP",
				PipelineName: "zr-fig-pipe-001",
			},
			hariConfig: map[string]interface{}{"mock": true},
			dirOpts:    []fs.PathOp{fs.WithDir("DPJ/journal"), fs.WithFile("DPJ/journal/avlxml.xml", "<xml/>")},
		},
		"Failure when HARI returns a server error": {
			params: UpdateHARIActivityParams{
				StoredAt:     time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
				SIPID:        "1db240cc-3cea-4e55-903c-6280562e1866",
				Kind:         "dpj-sip",
				PipelineName: "zr-fig-pipe-001",
			},
			hariConfig: map[string]interface{}{},
			dirOpts:    []fs.PathOp{fs.WithDir("DPJ/journal"), fs.WithFile("DPJ/journal/avlxml.xml", "<xml/>")},
			response:   &serverResponse{code: 500, status: "Backend server not available, try again later."},
			receipt: &avlRequest{
				Message:   "AVLXML was processed by Archivematica pipeline zr-fig-pipe-001",
				Type:      "dpj",
				Timestamp: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
				AIPID:     "1db240cc-3cea-4e55-903c-6280562e1866",
				XML:       []byte(`<xml/>`),
			},
			err: []string{"error sending request: unexpected response status: 500 Internal Server Error"},
		},
		"Empty kind is rejected": {
			params: UpdateHARIActivityParams{
				StoredAt: time.Now(),
				SIPID:    "1db240cc-3cea-4e55-903c-6280562e1866",
				Kind:     "",
			},
			err: []string{wferrors.NRE, "error validating kind attribute: empty"},
		},
		"Unsuffixed kind is rejected": {
			params: UpdateHARIActivityParams{
				StoredAt: time.Now(),
				SIPID:    "1db240cc-3cea-4e55-903c-6280562e1866",
				Kind:     "DPJ",
			},
			err: []string{wferrors.NRE, "error validating kind attribute: attribute (DPJ) does not containt suffix (\"-SIP\")"},
		},
		"Unknown kind is rejected": {
			params: UpdateHARIActivityParams{
				StoredAt: time.Now(),
				SIPID:    "1db240cc-3cea-4e55-903c-6280562e1866",
				Kind:     "FOOBAR-SIP",
			},
			err: []string{wferrors.NRE, "error validating kind attribute: attribute (FOOBAR) is unexpected/unknown"},
		},
		"Unexisten AVLXML file causes error": {
			params: UpdateHARIActivityParams{
				StoredAt: time.Now(),
				SIPID:    "1db240cc-3cea-4e55-903c-6280562e1866",
				Kind:     "DPJ-SIP",
			},
			hariConfig: map[string]interface{}{"baseURL": "http://192.168.1.50:12345"},
			dirOpts:    []fs.PathOp{fs.WithDir("DPJ/journal"), fs.WithFile("DPJ/journal/_____other_name_____.xml", "<xml/>")},
			err:        []string{wferrors.NRE, "error reading AVLXML file: not found"},
		},
		"Unparseable baseURL is rejected": {
			params: UpdateHARIActivityParams{
				StoredAt: time.Now(),
				SIPID:    "1db240cc-3cea-4e55-903c-6280562e1866",
				Kind:     "DPJ-SIP",
			},
			hariConfig: map[string]interface{}{"baseURL": string([]byte{0x7f})},
			dirOpts:    []fs.PathOp{fs.WithDir("DPJ/journal"), fs.WithFile("DPJ/journal/avlxml.xml", "<xml/>")},
			err:        []string{wferrors.NRE, "error in URL construction: error looking up baseURL configuration attribute: parse : net/url: invalid control character in URL"},
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

				if tc.receipt != nil {
					blob, err := ioutil.ReadAll(r.Body)
					assert.NilError(t, err)
					defer r.Body.Close()

					want, have := tc.receipt, &avlRequest{}
					assert.NilError(t, json.Unmarshal(blob, have))
					assert.DeepEqual(t, want, have)
				}

				if tc.response != nil {
					w.WriteHeader(tc.response.code)
					w.Write([]byte(tc.response.status))
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

			switch {
			case tc.err != nil && len(tc.err) > 0 && tc.err[0] == wferrors.NRE:
				assertCustomError(t, err, tc.err[0], tc.err[1])
			case tc.err != nil:
				assert.Error(t, err, tc.err[0])
			case tc.err == nil:
				assert.NilError(t, err, customErrorDetails(err))
			}
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

// assertCustomError verifies the properties of a cadence.CustomError.
func assertCustomError(t *testing.T, err error, reason string, details string) {
	t.Helper()

	assert.ErrorType(t, err, &cadence.CustomError{})
	assert.ErrorContains(t, err, reason)

	var result string
	perr := err.(*cadence.CustomError)
	assert.NilError(t, perr.Details(&result))
	assert.Equal(t, result, details)
}

// customErrorDetails extracts the details of a cadence.CustomError when possible.
func customErrorDetails(err error) string {
	var result string
	perr, ok := err.(*cadence.CustomError)
	if !ok {
		return ""
	}

	if err := perr.Details(&result); err != nil {
		return ""
	}

	return result
}
