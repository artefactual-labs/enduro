package activities

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"

	"github.com/artefactual-labs/enduro/internal/nha"
)

func TestDPJReceiptsRejectTagFilesOutsidePayload(t *testing.T) {
	const identifiers = `[{"file":"objects/DPJ/journal/avlxml.xml","identifiers":[{"identifierType":"avleveringsidentifikator","identifier":"12345"}]}]`
	journal := fs.WithDir("DPJ/journal", fs.WithFile("avlxml.xml", "<avlxml/>"))
	metadata := fs.WithDir("metadata", fs.WithFile("identifiers.json", identifiers))
	info := nha.NameInfo{Type: nha.TransferTypeDPJ, Identifier: "journal-id"}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected HARI request: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()
	hari := createHariActivity(t, map[string]any{"baseurl": server.URL})

	for _, tc := range []struct {
		name    string
		payload fs.PathOp
		wantErr string
	}{
		{"missing payload journal", metadata, "AVLXML file: not found"},
		{"missing payload identifiers", journal, "identifiers.json"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bag := fs.NewDir(t, "bag",
				fs.WithFile("bagit.txt", "BagIt-Version: 1.0\nTag-File-Character-Encoding: UTF-8\n"),
				journal, metadata, fs.WithDir("data", tc.payload),
			)
			defer bag.Remove()
			err := hari.Execute(t.Context(), &UpdateHARIActivityParams{FullPath: bag.Path(), NameInfo: info})
			assert.ErrorContains(t, err, tc.wantErr)
		})
	}

	t.Run("PROD requires payload identifiers", func(t *testing.T) {
		bag := fs.NewDir(t, "bag",
			fs.WithFile("bagit.txt", "BagIt-Version: 1.0\nTag-File-Character-Encoding: UTF-8\n"),
			journal, metadata, fs.WithDir("data", journal),
		)
		defer bag.Remove()
		out := t.TempDir()
		prod := createProdActivity(t, map[string]any{"receiptpath": out})
		err := prod.Execute(t.Context(), &UpdateProductionSystemActivityParams{FullPath: bag.Path(), NameInfo: info})
		assert.ErrorContains(t, err, "identifiers.json")
		published, err := filepath.Glob(filepath.Join(out, "*.mft"))
		assert.NilError(t, err)
		assert.Equal(t, len(published), 0)
	})
}

func TestConvertedReceiptsRequireManifestedInputs(t *testing.T) {
	info := nha.NameInfo{Type: nha.TransferTypeDPJ, Identifier: "journal-id"}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected HARI request: %s", r.URL.Path)
	}))
	defer server.Close()
	hari := createHariActivity(t, map[string]any{"baseurl": server.URL})
	for _, tc := range []struct{ name, entry, wantErr string }{
		{"tag identifiers", "data/DPJ/journal/avlxml.xml", "identifiers.json missing from original payload"},
		{"tag journal", "data/metadata/identifiers.json", "AVLXML file: not found"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := fs.NewDir(t, "converted",
				fs.WithDir("metadata/submissionDocumentation", fs.WithFile("bagit.txt", "BagIt-Version: 0.97\n"),
					fs.WithFile("manifest-sha256.txt", "digest  "+tc.entry+"\n")),
				fs.WithDir("DPJ/journal", fs.WithFile("avlxml.xml", "<tag/>")),
				fs.WithDir("objects/DPJ/journal", fs.WithFile("avlxml.xml", "<avlxml/>")),
				fs.WithFile("metadata/identifiers.json", `[{"file":"objects/DPJ/journal/avlxml.xml","identifiers":[{"identifierType":"avleveringsidentifikator","identifier":"wrong-parent"}]}]`),
			)
			defer dir.Remove()
			assert.NilError(t, os.WriteFile(dir.Join("metadata", "submissionDocumentation", "manifest-aaa.txt"), []byte("digest  data/DPJ/journal/avlxml.xml\ndigest  data/metadata/identifiers.json\n"), 0o600))
			err := hari.Execute(t.Context(), &UpdateHARIActivityParams{FullPath: dir.Path(), NameInfo: info})
			assert.ErrorContains(t, err, tc.wantErr)
			if tc.name == "tag identifiers" {
				out := t.TempDir()
				prod := createProdActivity(t, map[string]any{"receiptpath": out})
				err := prod.Execute(t.Context(), &UpdateProductionSystemActivityParams{FullPath: dir.Path(), NameInfo: info})
				assert.ErrorContains(t, err, tc.wantErr)
				published, err := filepath.Glob(filepath.Join(out, "*.mft"))
				assert.NilError(t, err)
				assert.Equal(t, len(published), 0)
			}
		})
	}
}
