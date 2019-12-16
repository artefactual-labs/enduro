package activities

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/artefactual-labs/enduro/internal/collection"
	collectionfake "github.com/artefactual-labs/enduro/internal/collection/fake"
	"github.com/artefactual-labs/enduro/internal/pipeline"
	watcherfake "github.com/artefactual-labs/enduro/internal/watcher/fake"
	"github.com/artefactual-labs/enduro/internal/workflow/manager"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"

	logrt "github.com/go-logr/logr/testing"
	"github.com/golang/mock/gomock"
)

func TestProdActivity(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		params      UpdateProductionSystemActivityParams
		hookConfig  *map[string]interface{}
		wantContent string
		wantErr     activityError
	}{
		"Receipt is generated successfully with status 'done'": {
			params: UpdateProductionSystemActivityParams{
				OriginalID:   "aa1df25d-1477-4085-8be3-a17fed20f843",
				Kind:         "DPJ-SIP",
				StoredAt:     time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
				PipelineName: "foo-bar-001",
				Status:       collection.StatusDone,
			},
			wantContent: `{
  "identifier": "aa1df25d-1477-4085-8be3-a17fed20f843",
  "type": "dpj",
  "accepted": true,
  "message": "Package was processed by Archivematica pipeline foo-bar-001",
  "timestamp": "2009-11-10T23:00:00Z"
}
`,
		},
		"Receipt is generated successfully with status 'error'": {
			params: UpdateProductionSystemActivityParams{
				OriginalID:   "aa1df25d-1477-4085-8be3-a17fed20f843",
				Kind:         "DPJ-SIP",
				StoredAt:     time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
				PipelineName: "foo-bar-002",
				Status:       collection.StatusError,
			},
			wantContent: `{
  "identifier": "aa1df25d-1477-4085-8be3-a17fed20f843",
  "type": "dpj",
  "accepted": false,
  "message": "Package was not processed successfully",
  "timestamp": "2009-11-10T23:00:00Z"
}
`,
		},
		"Empty OriginalID is rejected": {
			params: UpdateProductionSystemActivityParams{
				OriginalID:   "",
				Kind:         "DPJ-SIP",
				StoredAt:     time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
				PipelineName: "foo-bar-002",
				Status:       collection.StatusError,
			},
			wantErr: activityError{
				Message: "OriginalID is missing or empty",
				NRE:     true,
			},
		},
		"Unknown kind is rejected": {
			params: UpdateProductionSystemActivityParams{
				OriginalID:   "aa1df25d-1477-4085-8be3-a17fed20f843",
				Kind:         "FOO-SIP",
				StoredAt:     time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
				PipelineName: "foo-bar-002",
				Status:       collection.StatusError,
			},
			wantErr: activityError{
				Message: "error validating kind attribute: attribute (FOO) is unexpected/unknown",
				NRE:     true,
			},
		},
		"Malformed kind is rejected": {
			params: UpdateProductionSystemActivityParams{
				OriginalID:   "aa1df25d-1477-4085-8be3-a17fed20f843",
				Kind:         "DPJ",
				StoredAt:     time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
				PipelineName: "foo-bar-002",
				Status:       collection.StatusError,
			},
			wantErr: activityError{
				Message: "error validating kind attribute: attribute (DPJ) does not containt suffix (\"-SIP\")",
				NRE:     true,
			},
		},
		"Missing receiptPath is rejected": {
			params: UpdateProductionSystemActivityParams{
				OriginalID:   "aa1df25d-1477-4085-8be3-a17fed20f843",
				Kind:         "DPJ-SIP",
				StoredAt:     time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
				PipelineName: "foo-bar-001",
				Status:       collection.StatusDone,
			},
			hookConfig: &map[string]interface{}{},
			wantErr: activityError{
				Message: "error looking up receiptPath configuration attribute: error accessing \"prod:receiptPath\"",
				NRE:     true,
			},
		},
		"Unexistent receiptPath is rejected": {
			params: UpdateProductionSystemActivityParams{
				OriginalID:   "aa1df25d-1477-4085-8be3-a17fed20f843",
				Kind:         "DPJ-SIP",
				StoredAt:     time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
				PipelineName: "foo-bar-001",
				Status:       collection.StatusDone,
			},
			hookConfig: &map[string]interface{}{
				"receiptPath": "/tmp/1/2/3/4/5",
			},
			wantErr: activityError{
				Message: "error creating receipt file: open /tmp/1/2/3/4/5/Receipt_aa1df25d-1477-4085-8be3-a17fed20f843_20091110.230000.json: no such file or directory",
				NRE:     true,
			},
		},
	}

	for name, tc := range tests {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tmpdir := fs.NewDir(t, "enduro")
			defer tmpdir.Remove()
			hookConfig := map[string]interface{}{"receiptPath": tmpdir.Path()}
			if tc.hookConfig != nil {
				hookConfig = *tc.hookConfig
			}

			act := createProdActivity(t, hookConfig)

			err := act.Execute(context.Background(), &tc.params)

			tc.wantErr.Assert(t, err)

			// Stop here if we were expecting the activity to fail.
			if !tc.wantErr.IsZero() {
				return
			}

			assert.Assert(t, fs.Equal(
				tmpdir.Path(),
				fs.Expected(t,
					fs.WithFile(
						fmt.Sprintf("Receipt_%s_%s.json", tc.params.OriginalID, tc.params.StoredAt.Format(rfc3339forFilename)),
						tc.wantContent,
						fs.WithMode(os.FileMode(0o644))),
				)))
		})
	}
}

func createProdActivity(t *testing.T, hookConfig map[string]interface{}) *UpdateProductionSystemActivity {
	t.Helper()

	ctrl := gomock.NewController(t)

	manager := manager.NewManager(
		logrt.NullLogger{},
		collectionfake.NewMockService(ctrl),
		watcherfake.NewMockService(ctrl),
		&pipeline.Registry{},
		map[string]map[string]interface{}{
			"prod": hookConfig,
		},
	)

	return NewUpdateProductionSystemActivity(manager)
}
