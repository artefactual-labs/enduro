package frontend

import "testing"

func TestNormalizeBasePath(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty", in: "", want: "/"},
		{name: "root", in: "/", want: "/"},
		{name: "without slash", in: "v2", want: "/v2"},
		{name: "trailing slash", in: "/v2/", want: "/v2"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizeBasePath(tc.in)
			if got != tc.want {
				t.Fatalf("normalizeBasePath(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestStripBasePath(t *testing.T) {
	tests := []struct {
		name        string
		requestPath string
		basePath    string
		wantPath    string
		wantOK      bool
	}{
		{name: "exact base", requestPath: "/v2", basePath: "/v2", wantPath: "/", wantOK: true},
		{name: "within base", requestPath: "/v2/collections", basePath: "/v2", wantPath: "/collections", wantOK: true},
		{name: "outside base", requestPath: "/collections", basePath: "/v2", wantPath: "", wantOK: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotPath, gotOK := stripBasePath(tc.requestPath, tc.basePath)
			if gotPath != tc.wantPath || gotOK != tc.wantOK {
				t.Fatalf("stripBasePath(%q, %q) = (%q, %v), want (%q, %v)",
					tc.requestPath, tc.basePath, gotPath, gotOK, tc.wantPath, tc.wantOK)
			}
		})
	}
}

func TestPathRouting(t *testing.T) {
	tests := []struct {
		name             string
		requestPath      string
		wantAssetPath    string
		wantRouteIndex   bool
		wantRootFallback bool
	}{
		{
			name:             "root",
			requestPath:      "/",
			wantAssetPath:    indexPath,
			wantRouteIndex:   false,
			wantRootFallback: false,
		},
		{
			name:             "prerendered route",
			requestPath:      "/collections",
			wantAssetPath:    buildRoot + "/collections",
			wantRouteIndex:   true,
			wantRootFallback: true,
		},
		{
			name:             "static asset",
			requestPath:      "/_nuxt/app.js",
			wantAssetPath:    buildRoot + "/_nuxt/app.js",
			wantRouteIndex:   false,
			wantRootFallback: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := pathToAsset(tc.requestPath); got != tc.wantAssetPath {
				t.Fatalf("pathToAsset(%q) = %q, want %q", tc.requestPath, got, tc.wantAssetPath)
			}
			if got := shouldTryDirectoryIndex(tc.requestPath); got != tc.wantRouteIndex {
				t.Fatalf("shouldTryDirectoryIndex(%q) = %v, want %v", tc.requestPath, got, tc.wantRouteIndex)
			}
			if got := shouldFallbackToIndex(tc.requestPath); got != tc.wantRootFallback {
				t.Fatalf("shouldFallbackToIndex(%q) = %v, want %v", tc.requestPath, got, tc.wantRootFallback)
			}
		})
	}
}
