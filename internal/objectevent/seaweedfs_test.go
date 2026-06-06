package objectevent

import (
	"testing"

	"gotest.tools/v3/assert"

	"github.com/artefactual-labs/enduro/internal/watcher"
)

func TestEnduroEventFromSeaweedFS(t *testing.T) {
	for _, tc := range []struct {
		name        string
		event       seaweedFSEvent
		expected    *watcher.EnduroEvent
		expectedOK  bool
		expectedErr string
	}{
		{
			name: "CreateFile",
			event: seaweedFSEvent{
				Key:       "/buckets/sips/path/to/transfer.zip",
				EventType: "create",
				Message: seaweedFSEventMessage{
					NewEntry: &seaweedFSEntry{},
				},
			},
			expected: &watcher.EnduroEvent{
				Version: "1",
				Type:    watcher.EnduroEventTypeObjectCreated,
				Bucket:  "sips",
				Key:     "path/to/transfer.zip",
				Source:  "seaweedfs",
			},
			expectedOK: true,
		},
		{
			name: "CustomBucketsPath",
			event: seaweedFSEvent{
				Key:       "/custom/sips/transfer.zip",
				EventType: "create",
				Message: seaweedFSEventMessage{
					NewEntry: &seaweedFSEntry{},
				},
			},
			expected: &watcher.EnduroEvent{
				Version: "1",
				Type:    watcher.EnduroEventTypeObjectCreated,
				Bucket:  "sips",
				Key:     "transfer.zip",
				Source:  "seaweedfs",
			},
			expectedOK: true,
		},
		{
			name: "PreserveRawObjectKey",
			event: seaweedFSEvent{
				Key:       "/buckets/sips/list+%C3%A9mail+draft.txt",
				EventType: "create",
				Message: seaweedFSEventMessage{
					NewEntry: &seaweedFSEntry{},
				},
			},
			expected: &watcher.EnduroEvent{
				Version: "1",
				Type:    watcher.EnduroEventTypeObjectCreated,
				Bucket:  "sips",
				Key:     "list+%C3%A9mail+draft.txt",
				Source:  "seaweedfs",
			},
			expectedOK: true,
		},
		{
			name: "UpdateIgnored",
			event: seaweedFSEvent{
				Key:       "/buckets/sips/transfer.zip",
				EventType: "update",
			},
			expectedOK: false,
		},
		{
			name: "DirectoryCreateIgnored",
			event: seaweedFSEvent{
				Key:       "/buckets/sips/folder",
				EventType: "create",
				Message: seaweedFSEventMessage{
					NewEntry: &seaweedFSEntry{IsDirectory: true},
				},
			},
			expectedOK: false,
		},
		{
			name: "CreateMissingNewEntry",
			event: seaweedFSEvent{
				Key:       "/buckets/sips/transfer.zip",
				EventType: "create",
			},
			expectedErr: "create event missing new_entry",
		},
		{
			name: "OutsideBucketsPath",
			event: seaweedFSEvent{
				Key:       "/documents/report.pdf",
				EventType: "create",
				Message: seaweedFSEventMessage{
					NewEntry: &seaweedFSEntry{},
				},
			},
			expectedErr: `event key "/documents/report.pdf" is outside buckets path "/buckets"`,
		},
		{
			name: "MissingObjectKey",
			event: seaweedFSEvent{
				Key:       "/buckets/sips",
				EventType: "create",
				Message: seaweedFSEventMessage{
					NewEntry: &seaweedFSEntry{},
				},
			},
			expectedErr: `event key "/buckets/sips" does not include bucket and object key`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bucketsPath := "/buckets"
			if tc.name == "CustomBucketsPath" {
				bucketsPath = "/custom"
			}

			event, ok, err := enduroEventFromSeaweedFS(tc.event, bucketsPath)
			if tc.expectedErr != "" {
				assert.ErrorContains(t, err, tc.expectedErr)
				return
			}
			assert.NilError(t, err)
			assert.Equal(t, ok, tc.expectedOK)
			assert.DeepEqual(t, event, tc.expected)
		})
	}
}
