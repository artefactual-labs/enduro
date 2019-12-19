package activities

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/artefactual-labs/enduro/internal/collection"
	collectionfake "github.com/artefactual-labs/enduro/internal/collection/fake"
	"github.com/artefactual-labs/enduro/internal/nha"
	"github.com/artefactual-labs/enduro/internal/pipeline"
	"github.com/artefactual-labs/enduro/internal/testutil"
	watcherfake "github.com/artefactual-labs/enduro/internal/watcher/fake"
	"github.com/artefactual-labs/enduro/internal/workflow/manager"
	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"

	logrt "github.com/go-logr/logr/testing"
	"github.com/golang/mock/gomock"
)

func TestProdActivity(t *testing.T) {
	t.Parallel()

	tmpdir, err := ioutil.TempDir("", "")
	assert.NilError(t, err)
	defer os.RemoveAll(tmpdir)

	tests := map[string]struct {
		params       UpdateProductionSystemActivityParams
		hookConfig   *map[string]interface{}
		wantContent  string
		wantChecksum string
		wantErr      testutil.ActivityError
	}{
		"Receipt is generated successfully with status 'done'": {
			params: UpdateProductionSystemActivityParams{
				StoredAt:     time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
				PipelineName: "foo-bar-001",
				Status:       collection.StatusDone,
				NameInfo: nha.NameInfo{
					Identifier: "aa1df25d-1477-4085-8be3-a17fed20f843",
					Type:       nha.TransferTypeDPJ,
				},
			},
			wantContent: `{
  "identifier": "aa1df25d-1477-4085-8be3-a17fed20f843",
  "type": "dpj",
  "accepted": true,
  "message": "Package was processed by Archivematica pipeline foo-bar-001",
  "timestamp": "2009-11-10T23:00:00Z"
}
`,
			wantChecksum: "eed2dd4ee8a1dcf637b0708e616a4767",
		},
		"Receipt is generated successfully with status 'error'": {
			params: UpdateProductionSystemActivityParams{
				StoredAt:     time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
				PipelineName: "foo-bar-002",
				Status:       collection.StatusError,
				NameInfo: nha.NameInfo{
					Identifier: "aa1df25d-1477-4085-8be3-a17fed20f843",
					Type:       nha.TransferTypeDPJ,
				},
			},
			wantContent: `{
  "identifier": "aa1df25d-1477-4085-8be3-a17fed20f843",
  "type": "dpj",
  "accepted": false,
  "message": "Package was not processed successfully",
  "timestamp": "2009-11-10T23:00:00Z"
}
`,
			wantChecksum: "210995b572d4e87fed73ca4312d59557",
		},
		"Missing receiptPath is rejected": {
			params: UpdateProductionSystemActivityParams{
				StoredAt:     time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
				PipelineName: "foo-bar-001",
				Status:       collection.StatusDone,
				NameInfo: nha.NameInfo{
					Identifier: "aa1df25d-1477-4085-8be3-a17fed20f843",
					Type:       nha.TransferTypeDPJ,
				},
			},
			hookConfig: &map[string]interface{}{},
			wantErr: testutil.ActivityError{
				Message: "error looking up receiptPath configuration attribute: error accessing \"prod:receiptPath\"",
				NRE:     true,
			},
		},
		"Unexistent receiptPath is rejected": {
			params: UpdateProductionSystemActivityParams{
				StoredAt:     time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
				PipelineName: "foo-bar-001",
				Status:       collection.StatusDone,
				NameInfo: nha.NameInfo{
					Identifier: "aa1df25d-1477-4085-8be3-a17fed20f843",
					Type:       nha.TransferTypeDPJ,
				},
			},
			hookConfig: &map[string]interface{}{
				"receiptPath": tmpdir,
			},
			wantErr: testutil.ActivityError{
				Message:        fmt.Sprintf("error creating receipt file: open %s: no such file or directory", filepath.Join(tmpdir, "Receipt_aa1df25d-1477-4085-8be3-a17fed20f843_20091110.230000.json")),
				MessageWindows: fmt.Sprintf("error creating receipt file: open %s: The system cannot find the path specified.", filepath.Join(tmpdir, "Receipt_aa1df25d-1477-4085-8be3-a17fed20f843_20091110.230000.json")),
				NRE:            true,
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

			var base = fmt.Sprintf("Receipt_%s_%s", tc.params.NameInfo.Identifier, tc.params.StoredAt.Format(rfc3339forFilename))
			assert.Assert(t, fs.Equal(
				tmpdir.Path(),
				fs.Expected(t,
					fs.WithFile(
						base+".mft",
						tc.wantContent,
						fs.WithMode(os.FileMode(0o644))),
					fs.WithFile(
						base+".md5",
						fmt.Sprintf("%s  %s", tc.wantChecksum, filepath.Join(tmpdir.Path(), base+".mft")),
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
