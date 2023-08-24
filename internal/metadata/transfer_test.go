package metadata_test

import (
	"testing"

	"gotest.tools/v3/assert"

	"github.com/artefactual-labs/enduro/internal/metadata"
)

func TestFromTransferName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		transferName string
		isDir        bool
		expected     metadata.TransferName
	}{
		{
			transferName: "componentNumber",
			expected: metadata.TransferName{
				DCIdentifier: "componentNumber",
				ComponentID:  "",
				Accession:    "",
			},
		},
		{
			transferName: "componentNumber---componentId",
			expected: metadata.TransferName{
				DCIdentifier: "componentNumber",
				ComponentID:  "componentId",
				Accession:    "",
			},
		},
		{
			transferName: "componentNumber---componentId---objectId",
			expected: metadata.TransferName{
				DCIdentifier: "componentNumber",
				ComponentID:  "componentId",
				Accession:    "objectId",
			},
		},
		{
			transferName: "componentNumber---componentId---objectId.zip",
			expected: metadata.TransferName{
				DCIdentifier: "componentNumber",
				ComponentID:  "componentId",
				Accession:    "objectId",
			},
		},
		{
			transferName: "245.2016.q.x1---522207---202992",
			isDir:        true,
			expected: metadata.TransferName{
				DCIdentifier: "245.2016.q.x1",
				ComponentID:  "522207",
				Accession:    "202992",
			},
		},
		{
			transferName: "249.2016.x3---357995---202998",
			isDir:        true,
			expected: metadata.TransferName{
				DCIdentifier: "249.2016.x3",
				ComponentID:  "357995",
				Accession:    "202998",
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.transferName, func(t *testing.T) {
			t.Parallel()

			assert.DeepEqual(t, metadata.FromTransferName(tc.transferName, tc.isDir), tc.expected)
		})

	}
}
