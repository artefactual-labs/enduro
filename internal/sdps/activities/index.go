package activities

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	"github.com/opensearch-project/opensearch-go"
	"github.com/opensearch-project/opensearch-go/opensearchapi"
	"go.uber.org/multierr"

	"github.com/artefactual-labs/enduro/internal/sdps"
)

// IndexActivity stores Batch data in OpenSearch
type IndexActivity struct {
	logger       logr.Logger
	searchClient *opensearch.Client
}

func NewIndexActivity(logger logr.Logger, searchClient *opensearch.Client) *IndexActivity {
	return &IndexActivity{logger: logger, searchClient: searchClient}
}

type IndexActivityParams struct {
	Path        string
	SearchIndex string
}

func (a IndexActivity) Execute(ctx context.Context, params *IndexActivityParams) error {
	a.logger.Info("Creating batch", "path", params.Path)
	b, err := sdps.OpenBatch(params.Path)
	if err != nil {
		return err
	}
	return batchIndexer(ctx, b, a.searchClient, params.SearchIndex, a.logger)
}

func batchIndexer(ctx context.Context, bi sdps.BatchIterator, searchClient *opensearch.Client, index string, logger logr.Logger) error {
	var errors error
	for bi.Next() {
		cur := bi.Item()

		blob, err := json.Marshal(cur)
		if err != nil {
			multierr.AppendInto(&errors, err)
			continue
		}

		documentID := cur.String()
		req := opensearchapi.IndexRequest{
			Index:      index,
			DocumentID: documentID,
			Body:       strings.NewReader(string(blob)),
			Refresh:    "true",
		}

		res, err := req.Do(ctx, searchClient)
		if err != nil {
			multierr.AppendInto(&errors, fmt.Errorf("Error getting response: %s", err))
			continue
		}
		defer res.Body.Close()

		if res.IsError() {
			multierr.AppendInto(&errors, fmt.Errorf("[%s] Error indexing document (ID=%s)", res.Status(), documentID))
		} else {
			logger.V(1).Info(
				"Document indexed",
				"ID", documentID,
				"body", string(blob),
			)
		}
	}

	if len(multierr.Errors(errors)) > 0 {
		return errors
	}

	return nil
}
