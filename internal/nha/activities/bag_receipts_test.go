package activities

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"

	"github.com/artefactual-labs/enduro/internal/nha"
	workflowactivities "github.com/artefactual-labs/enduro/internal/workflow/activities"
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

func TestConversionCannotSubstituteReceiptTags(t *testing.T) {
	const journal = "<avlxml/>"
	const ids = `[{"file":"objects/DPJ/journal/avlxml.xml","identifiers":[{"identifierType":"avleveringsidentifikator","identifier":"wrong-parent"}]}]`
	info := nha.NameInfo{Type: nha.TransferTypeDPJ, Identifier: "journal-id"}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("unexpected HARI delivery")
	}))
	defer server.Close()
	hari := createHariActivity(t, map[string]any{"baseurl": server.URL})
	for _, tc := range []struct{ payload, content, tag, tagContent, missing string }{
		{"DPJ/journal/avlxml.xml", journal, "metadata/identifiers.json", ids, "identifiers.json"},
		{"metadata/identifiers.json", ids, "DPJ/journal/avlxml.xml", journal, "AVLXML file"},
	} {
		t.Run(tc.missing, func(t *testing.T) {
			dir := fs.NewDir(t, "bag",
				fs.WithFile("bagit.txt", "BagIt-Version: 0.97\nTag-File-Character-Encoding: UTF-8\n"),
				fs.WithFile("bag-info.txt", fmt.Sprintf("Payload-Oxum: %d.1\n", len(tc.content))),
				fs.WithFile("manifest-sha256.txt", fmt.Sprintf("%x  data/%s\n", sha256.Sum256([]byte(tc.content)), tc.payload)),
				fs.WithDir(filepath.Join("data", filepath.Dir(tc.payload)), fs.WithFile(filepath.Base(tc.payload), tc.content)),
				fs.WithDir(filepath.Dir(tc.tag), fs.WithFile(filepath.Base(tc.tag), tc.tagContent)),
			)
			defer dir.Remove()
			result, err := workflowactivities.NewBundleActivity(nil, nil).Execute(t.Context(), &workflowactivities.BundleActivityParams{
				TempFile: dir.Path(), TransferDir: t.TempDir(), IsDir: true, Unbag: true,
			})
			assert.NilError(t, err)
			assert.Assert(t, result.Unbagged)
			err = hari.Execute(t.Context(), &UpdateHARIActivityParams{FullPath: result.FullPath, NameInfo: info, ConvertedLayout: result.Layout})
			assert.ErrorContains(t, err, tc.missing)
			if tc.tag == "metadata/identifiers.json" {
				out := t.TempDir()
				err = createProdActivity(t, map[string]any{"receiptpath": out}).Execute(t.Context(),
					&UpdateProductionSystemActivityParams{FullPath: result.FullPath, NameInfo: info, ConvertedLayout: result.Layout})
				assert.ErrorContains(t, err, tc.missing)
				published, err := filepath.Glob(filepath.Join(out, "*.mft"))
				assert.NilError(t, err)
				assert.Equal(t, len(published), 0)
			}
		})
	}
}
