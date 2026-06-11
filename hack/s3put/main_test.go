package main

import (
	"archive/zip"
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestNextKey(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		key     string
		prefix  string
		count   int
		index   int
		want    string
		wantErr bool
	}{
		"single upload uses explicit key": {
			key:    "transfer.zip",
			prefix: "ignored",
			count:  1,
			index:  1,
			want:   "transfer.zip",
		},
		"single upload can use generated key": {
			prefix: "issue-681",
			count:  1,
			index:  1,
			want:   "issue-681-001.zip",
		},
		"multiple uploads use generated keys": {
			prefix: "issue-681.zip",
			count:  25,
			index:  7,
			want:   "issue-681-007.zip",
		},
		"generated key requires prefix": {
			count:   2,
			index:   1,
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := nextKey(tc.key, tc.prefix, tc.count, tc.index)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("nextKey returned error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("nextKey = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestGenerateTransfer(t *testing.T) {
	t.Parallel()

	const key = "issue-681-001.zip"
	payload, err := generateTransfer(key, 1)
	if err != nil {
		t.Fatalf("generateTransfer returned error: %v", err)
	}

	zr, err := zip.NewReader(bytes.NewReader(payload), int64(len(payload)))
	if err != nil {
		t.Fatalf("generated payload is not a zip file: %v", err)
	}

	files := map[string]string{}
	for _, file := range zr.File {
		rc, err := file.Open()
		if err != nil {
			t.Fatalf("open generated file %q: %v", file.Name, err)
		}
		body, err := io.ReadAll(rc)
		if closeErr := rc.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
		if err != nil {
			t.Fatalf("read generated file %q: %v", file.Name, err)
		}
		files[file.Name] = string(body)
	}

	if !strings.Contains(files["objects/hello.txt"], key) {
		t.Fatalf("objects/hello.txt does not mention object key: %q", files["objects/hello.txt"])
	}
	if !strings.Contains(files["metadata/source.txt"], key) {
		t.Fatalf("metadata/source.txt does not mention object key: %q", files["metadata/source.txt"])
	}
}
