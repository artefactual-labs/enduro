package workflow

import "testing"

func TestBundleActivity_NameKind(t *testing.T) {
	a := &BundleActivity{}
	tests := []struct {
		key      string
		wantName string
		wantKind string
	}{
		{"foobar.jpg", "foobar.jpg", ""},
		{"c5ecddb0-7a61-4234-80a9-fa7993e97867.tar", "c5ecddb0-7a61-4234-80a9-fa7993e97867.tar", ""},
		{"DPJ-SIP-c5ecddb0-7a61-4234-80a9-fa7993e97867", "DPJ-SIP-c5ecddb0-7a61", "DPJ-SIP"},
		{"DPJ-SIP-c5ecddb0-7a61-4234-80a9-fa7993e97867.tar", "DPJ-SIP-c5ecddb0-7a61", "DPJ-SIP"},
		{"DPJ-SIP_c5ecddb0-7a61-4234-80a9-fa7993e97867.tar", "DPJ-SIP-c5ecddb0-7a61", "DPJ-SIP"},
	}
	for _, tt := range tests {
		gotName, gotKind := a.NameKind(tt.key)
		if gotName != tt.wantName {
			t.Errorf("BundleActivity.NameKind() gotName = %v, want %v", gotName, tt.wantName)
		}
		if gotKind != tt.wantKind {
			t.Errorf("BundleActivity.NameKind() gotKind = %v, want %v", gotKind, tt.wantKind)
		}
	}
}
