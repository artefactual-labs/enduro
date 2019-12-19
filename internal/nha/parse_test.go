package nha

import (
	"testing"
	"time"

	"gotest.tools/v3/assert"
)

func TestParse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc       string
		input      string
		wantErr    string
		wantResult NameInfo
	}{
		{
			desc:  "Canonical DPJ-SIP",
			input: "dpj-sip-814ecc88-b459-4304-8868-9ed72875f5fc.tar",
			wantResult: NameInfo{
				Identifier: "814ecc88-b459-4304-8868-9ed72875f5fc",
				Type:       TransferTypeDPJ,
				Date:       time.Time{},
				Extension:  "tar",
				Value:      "dpj-sip-814ecc88-b459-4304-8868-9ed72875f5fc.tar",
			},
		},
		{
			desc:  "Canonical EPJ-SIP",
			input: "epj-sip-814ecc88-b459-4304-8868-9ed72875f5fc.tar",
			wantResult: NameInfo{
				Identifier: "814ecc88-b459-4304-8868-9ed72875f5fc",
				Type:       TransferTypeEPJ,
				Date:       time.Time{},
				Extension:  "tar",
				Value:      "epj-sip-814ecc88-b459-4304-8868-9ed72875f5fc.tar",
			},
		},
		{
			desc:  "Canonical OTHER-SIP",
			input: "other-sip-814ecc88-b459-4304-8868-9ed72875f5fc.tar",
			wantResult: NameInfo{
				Identifier: "814ecc88-b459-4304-8868-9ed72875f5fc",
				Type:       TransferTypeOther,
				Date:       time.Time{},
				Extension:  "tar",
				Value:      "other-sip-814ecc88-b459-4304-8868-9ed72875f5fc.tar",
			},
		},
		{
			desc:  "Canonical AVL-EPJ-SIP",
			input: "avl-epj-sip-2.16.578.1.39.100.11.9876.4-20191104.xml.tar",
			wantResult: NameInfo{
				Identifier: "2.16.578.1.39.100.11.9876.4",
				Type:       TransferTypeAVLXML,
				Date:       time.Date(2019, 11, 4, 0, 0, 0, 0, time.UTC),
				Extension:  "xml.tar",
				Value:      "avl-epj-sip-2.16.578.1.39.100.11.9876.4-20191104.xml.tar",
			},
		},
		{
			desc:  "Canonical AVL-DPJ-SIP",
			input: "avl-dpj-sip-2.16.578.1.39.100.11.9876.4-20191104.xml.tar",
			wantResult: NameInfo{
				Identifier: "2.16.578.1.39.100.11.9876.4",
				Type:       TransferTypeAVLXML,
				Date:       time.Date(2019, 11, 4, 0, 0, 0, 0, time.UTC),
				Extension:  "xml.tar",
				Value:      "avl-dpj-sip-2.16.578.1.39.100.11.9876.4-20191104.xml.tar",
			},
		},
		{
			desc:  "Canonical AVL-OTHER-SIP",
			input: "avl-other-sip-2.16.578.1.39.100.11.9876.4-20191104.xml.tar",
			wantResult: NameInfo{
				Identifier: "2.16.578.1.39.100.11.9876.4",
				Type:       TransferTypeAVLXML,
				Date:       time.Date(2019, 11, 4, 0, 0, 0, 0, time.UTC),
				Extension:  "xml.tar",
				Value:      "avl-other-sip-2.16.578.1.39.100.11.9876.4-20191104.xml.tar",
			},
		},
		{
			desc:  "Mixed case",
			input: "DPJ-SIP_814ecc88-b459-4304-8868-9ed72875f5fc.zip",
			wantResult: NameInfo{
				Identifier: "814ecc88-b459-4304-8868-9ed72875f5fc",
				Type:       TransferTypeDPJ,
				Date:       time.Time{},
				Extension:  "zip",
				Value:      "DPJ-SIP_814ecc88-b459-4304-8868-9ed72875f5fc.zip",
			},
		},
		{
			desc:  "Mixed case and underscore separator",
			input: "AvL-OthEr-SiP_814eCC88-B459-4304-8868-9ED72875F5FC.XML.zip",
			wantResult: NameInfo{
				Identifier: "814eCC88-B459-4304-8868-9ED72875F5FC",
				Type:       TransferTypeAVLXML,
				Date:       time.Time{},
				Extension:  "XML.zip",
				Value:      "AvL-OthEr-SiP_814eCC88-B459-4304-8868-9ED72875F5FC.XML.zip",
			},
		},
		{
			desc:  "Without extension",
			input: "dpj-sip-814ecc88-b459-4304-8868-9ed72875f5fc",
			wantResult: NameInfo{
				Identifier: "814ecc88-b459-4304-8868-9ed72875f5fc",
				Type:       TransferTypeDPJ,
				Date:       time.Time{},
				Extension:  "",
				Value:      "dpj-sip-814ecc88-b459-4304-8868-9ed72875f5fc",
			},
		},
		{
			desc:    "Unexpected UUID format",
			input:   "dpj-sip-814ecc88-b459-4304-8868-9ed72875f5fZ",
			wantErr: "error parsing uuid: invalid UUID format",
		},
		{
			desc:    "Unexpected date format",
			input:   "avl-other-sip-2.16.578.1.39.100.11.9876.4-20191404.xml.tar",
			wantErr: "error parsing date: parsing time \"20191404\": month out of range",
		},
		{
			desc:    "Unexpected name",
			input:   "12345.xml.tar",
			wantErr: "error parsing name: no matches",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()

			res, err := ParseName(tc.input)

			if tc.wantErr != "" {
				assert.Error(t, err, tc.wantErr, tc.input)
				return
			}

			assert.NilError(t, err, tc.input)
			assert.DeepEqual(t, *res, tc.wantResult)
			assert.Equal(t, res.Value, tc.wantResult.String())
		})
	}
}
