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
		params  UpdateProductionSystemActivityParams
		content string
	}{
		"Receipt is generated successfully": {
			params: UpdateProductionSystemActivityParams{
				OriginalID:   "aa1df25d-1477-4085-8be3-a17fed20f843",
				Kind:         "DPJ-SIP",
				StoredAt:     time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
				PipelineName: "foo-bar-001",
				Status:       collection.StatusDone,
			},
			content: `{
  "identifier": "aa1df25d-1477-4085-8be3-a17fed20f843",
  "type": "dpj",
  "accepted": true,
  "message": "Package was processed by Archivematica pipeline foo-bar-001",
  "timestamp": "2009-11-10T23:00:00Z"
}
`,
		},
	}

	for name, tc := range tests {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tmpdir := fs.NewDir(t, "enduro")
			defer tmpdir.Remove()
			act := createProdActivity(t, tmpdir.Path())

			err := act.Execute(context.Background(), &tc.params)

			assert.NilError(t, err, customErrorDetails(err))
			assert.Assert(t, fs.Equal(
				tmpdir.Path(),
				fs.Expected(t,
					fs.WithFile(
						fmt.Sprintf("Receipt_%s_%s.json", tc.params.OriginalID, tc.params.StoredAt.Format(rfc3339forFilename)),
						tc.content,
						fs.WithMode(os.FileMode(0o644))),
				)))
		})
	}
}

func createProdActivity(t *testing.T, receiptPath string) *UpdateProductionSystemActivity {
	t.Helper()

	ctrl := gomock.NewController(t)

	hooks := map[string]map[string]interface{}{
		"prod": {
			"receiptPath": receiptPath,
		},
	}

	manager := manager.NewManager(
		logrt.NullLogger{},
		collectionfake.NewMockService(ctrl),
		watcherfake.NewMockService(ctrl),
		&pipeline.Registry{},
		hooks,
	)

	return NewUpdateProductionSystemActivity(manager)
}
