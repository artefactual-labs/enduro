package pipeline

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	kabs "github.com/microsoft/kiota-abstractions-go"
	"github.com/microsoft/kiota-abstractions-go/serialization"
	"github.com/microsoft/kiota-abstractions-go/store"
	ssclient "go.artefactual.dev/ssclient"
	"go.artefactual.dev/ssclient/kiota/models"
	"gotest.tools/v3/assert"
)

func TestGetStoragePackage(t *testing.T) {
	t.Parallel()

	t.Run("Returns normalized package info", func(t *testing.T) {
		t.Parallel()

		const (
			primaryID  = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
			primaryLoc = "33333333-3333-4333-8333-333333333333"
			replicaID  = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
			replicaLoc = "44444444-4444-4444-8444-444444444444"
		)

		p := newStoragePipelineForTest(t, &fakeRequestAdapter{
			send: func(ctx context.Context, requestInfo *kabs.RequestInformation, constructor serialization.ParsableFactory, errorMappings kabs.ErrorMappings) (serialization.Parsable, error) {
				id := pathUUIDString(t, requestInfo)
				switch id {
				case primaryID:
					primaryStoredAt := time.Date(2026, 3, 17, 7, 0, 0, 0, time.UTC)
					pkg := models.NewPackageEscaped()
					pkg.SetUuid(uuidPtr("11111111-1111-1111-1111-111111111111"))
					pkg.SetStatus(stringPtr("UPLOADED"))
					pkg.SetStoredDate(&primaryStoredAt)
					pkg.SetCurrentFullPath(stringPtr("/var/aips/" + primaryID + ".7z"))
					pkg.SetCurrentLocation(stringPtr("/api/v2/location/" + primaryLoc + "/"))
					pkg.SetReplicas([]string{"/api/v2/file/" + replicaID + "/"})
					return pkg, nil
				case replicaID:
					replicaStoredAt := time.Date(2026, 3, 17, 8, 0, 0, 0, time.UTC)
					pkg := models.NewPackageEscaped()
					pkg.SetUuid(uuidPtr("22222222-2222-2222-2222-222222222222"))
					pkg.SetStatus(stringPtr("UPLOADED"))
					pkg.SetStoredDate(&replicaStoredAt)
					pkg.SetCurrentFullPath(stringPtr("/replicas/" + primaryID + ".7z"))
					pkg.SetCurrentLocation(stringPtr("/api/v2/location/" + replicaLoc + "/"))
					pkg.SetReplicas([]string{})
					return pkg, nil
				default:
					assert.Assert(t, false, "unexpected request UUID "+id)
					return nil, nil
				}
			},
		})

		pkg, err := p.GetStoragePackage(context.Background(), primaryID)

		assert.NilError(t, err)
		assert.Equal(t, pkg.UUID, "11111111-1111-1111-1111-111111111111")
		assert.Equal(t, pkg.Status, "UPLOADED")
		assert.Equal(t, pkg.CurrentFullPath, "/var/aips/"+primaryID+".7z")
		assert.Assert(t, pkg.StoredDate != nil)
		assert.Assert(t, pkg.StoredDate.Equal(time.Date(2026, 3, 17, 7, 0, 0, 0, time.UTC)))
		assert.Equal(t, pkg.CurrentLocation.UUID, primaryLoc)
		assert.Equal(t, len(pkg.Replicas), 1)
		assert.Equal(t, pkg.Replicas[0].UUID, "22222222-2222-2222-2222-222222222222")
		assert.Assert(t, pkg.Replicas[0].StoredDate != nil)
		assert.Assert(t, pkg.Replicas[0].StoredDate.Equal(time.Date(2026, 3, 17, 8, 0, 0, 0, time.UTC)))
		assert.Equal(t, pkg.Replicas[0].CurrentLocation.UUID, replicaLoc)
	})

	t.Run("Returns not found error", func(t *testing.T) {
		t.Parallel()

		p := newStoragePipelineForTest(t, &fakeRequestAdapter{
			send: func(context.Context, *kabs.RequestInformation, serialization.ParsableFactory, kabs.ErrorMappings) (serialization.Parsable, error) {
				return nil, &ssclient.ResponseError{StatusCode: http.StatusNotFound, Message: "not found"}
			},
		})

		pkg, err := p.GetStoragePackage(context.Background(), "99999999-9999-4999-8999-999999999999")

		assert.Assert(t, pkg == nil)
		assert.ErrorIs(t, err, ErrStoragePackageNotFound)
	})

	t.Run("Returns upstream status errors", func(t *testing.T) {
		t.Parallel()

		p := newStoragePipelineForTest(t, &fakeRequestAdapter{
			send: func(context.Context, *kabs.RequestInformation, serialization.ParsableFactory, kabs.ErrorMappings) (serialization.Parsable, error) {
				return nil, &ssclient.ResponseError{StatusCode: http.StatusUnauthorized, Message: "nope"}
			},
		})

		pkg, err := p.GetStoragePackage(context.Background(), "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa")

		assert.Assert(t, pkg == nil)
		status, ok := ssclient.StatusCode(err)
		assert.Assert(t, ok)
		assert.Equal(t, status, http.StatusUnauthorized)
	})

	t.Run("Returns parse errors", func(t *testing.T) {
		t.Parallel()

		p := newStoragePipelineForTest(t, &fakeRequestAdapter{
			send: func(context.Context, *kabs.RequestInformation, serialization.ParsableFactory, kabs.ErrorMappings) (serialization.Parsable, error) {
				return nil, errors.New("unexpected EOF")
			},
		})

		pkg, err := p.GetStoragePackage(context.Background(), "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa")

		assert.Assert(t, pkg == nil)
		assert.ErrorContains(t, err, "error querying storage service")
		assert.ErrorContains(t, err, "unexpected EOF")
	})

	t.Run("Respects context cancellation", func(t *testing.T) {
		t.Parallel()

		p := newStoragePipelineForTest(t, &fakeRequestAdapter{
			send: func(ctx context.Context, requestInfo *kabs.RequestInformation, constructor serialization.ParsableFactory, errorMappings kabs.ErrorMappings) (serialization.Parsable, error) {
				<-ctx.Done()
				return nil, ctx.Err()
			},
		})

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		pkg, err := p.GetStoragePackage(ctx, "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa")

		assert.Assert(t, pkg == nil)
		assert.ErrorContains(t, err, "context canceled")
	})

	t.Run("Returns errors on malformed location URI", func(t *testing.T) {
		t.Parallel()

		p := newStoragePipelineForTest(t, &fakeRequestAdapter{
			send: func(ctx context.Context, requestInfo *kabs.RequestInformation, constructor serialization.ParsableFactory, errorMappings kabs.ErrorMappings) (serialization.Parsable, error) {
				pkg := models.NewPackageEscaped()
				pkg.SetUuid(uuidPtr("11111111-1111-1111-1111-111111111111"))
				pkg.SetCurrentLocation(stringPtr("not-a-uri"))
				return pkg, nil
			},
		})

		pkg, err := p.GetStoragePackage(context.Background(), "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa")

		assert.Assert(t, pkg == nil)
		assert.ErrorContains(t, err, "parse storage service package location")
	})

	t.Run("Returns errors on malformed replica URI", func(t *testing.T) {
		t.Parallel()

		p := newStoragePipelineForTest(t, &fakeRequestAdapter{
			send: func(ctx context.Context, requestInfo *kabs.RequestInformation, constructor serialization.ParsableFactory, errorMappings kabs.ErrorMappings) (serialization.Parsable, error) {
				pkg := models.NewPackageEscaped()
				pkg.SetUuid(uuidPtr("11111111-1111-1111-1111-111111111111"))
				pkg.SetReplicas([]string{"definitely-not-a-resource"})
				return pkg, nil
			},
		})

		pkg, err := p.GetStoragePackage(context.Background(), "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa")

		assert.Assert(t, pkg == nil)
		assert.ErrorContains(t, err, "parse storage service replica URI")
	})

	t.Run("Primary-only reconciliation ignores malformed replica URI", func(t *testing.T) {
		t.Parallel()

		p := newStoragePipelineForTest(t, &fakeRequestAdapter{
			send: func(ctx context.Context, requestInfo *kabs.RequestInformation, constructor serialization.ParsableFactory, errorMappings kabs.ErrorMappings) (serialization.Parsable, error) {
				pkg := models.NewPackageEscaped()
				pkg.SetUuid(uuidPtr("11111111-1111-1111-1111-111111111111"))
				storedAt := time.Date(2026, 3, 17, 7, 0, 0, 0, time.UTC)
				pkg.SetStoredDate(&storedAt)
				pkg.SetReplicas([]string{"definitely-not-a-resource"})
				return pkg, nil
			},
		})

		pkg, err := p.GetStoragePackageForReconciliation(context.Background(), "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", RecoveryConfig{})

		assert.NilError(t, err)
		assert.Assert(t, pkg != nil)
		assert.Equal(t, len(pkg.Replicas), 0)
	})

	t.Run("Primary-only reconciliation ignores replica lookup failures", func(t *testing.T) {
		t.Parallel()

		const (
			primaryID = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
			replicaID = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
		)

		p := newStoragePipelineForTest(t, &fakeRequestAdapter{
			send: func(ctx context.Context, requestInfo *kabs.RequestInformation, constructor serialization.ParsableFactory, errorMappings kabs.ErrorMappings) (serialization.Parsable, error) {
				switch id := pathUUIDString(t, requestInfo); id {
				case primaryID:
					pkg := models.NewPackageEscaped()
					pkg.SetUuid(uuidPtr("11111111-1111-1111-1111-111111111111"))
					storedAt := time.Date(2026, 3, 17, 7, 0, 0, 0, time.UTC)
					pkg.SetStoredDate(&storedAt)
					pkg.SetReplicas([]string{"/api/v2/file/" + replicaID + "/"})
					return pkg, nil
				case replicaID:
					return nil, errors.New("temporary replica lookup failure")
				default:
					assert.Assert(t, false, "unexpected request UUID "+id)
					return nil, nil
				}
			},
		})

		pkg, err := p.GetStoragePackageForReconciliation(context.Background(), primaryID, RecoveryConfig{})

		assert.NilError(t, err)
		assert.Assert(t, pkg != nil)
		assert.Equal(t, len(pkg.Replicas), 0)
	})

	t.Run("Rejects invalid package UUID", func(t *testing.T) {
		t.Parallel()

		p := newStoragePipelineForTest(t, &fakeRequestAdapter{})

		pkg, err := p.GetStoragePackage(context.Background(), "not-a-uuid")

		assert.Assert(t, pkg == nil)
		assert.ErrorContains(t, err, `invalid storage package UUID "not-a-uuid"`)
	})
}

func TestDownloadStoragePackage(t *testing.T) {
	t.Parallel()

	t.Run("Returns package stream", func(t *testing.T) {
		t.Parallel()

		const packageID = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"

		p := newStoragePipelineForDownloadTest(t, &fakeRequestAdapter{
			baseURL: "http://example.com/storage-service",
			convertToNativeRequest: func(ctx context.Context, requestInfo *kabs.RequestInformation) (any, error) {
				return http.NewRequestWithContext(ctx, requestInfo.Method.String(), "http://example.com/storage-service/api/v2/file/"+packageID+"/download/", nil)
			},
		}, roundTripFunc(func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, req.URL.Path, "/storage-service/api/v2/file/"+packageID+"/download/")
			return &http.Response{
				StatusCode: http.StatusOK,
				Header: http.Header{
					"Content-Type":   []string{"application/zip"},
					"Content-Length": []string{"4"},
				},
				Body: io.NopCloser(strings.NewReader("test")),
			}, nil
		}))

		stream, err := p.DownloadStoragePackage(context.Background(), packageID)

		assert.NilError(t, err)
		assert.Assert(t, stream != nil)
		assert.Equal(t, stream.StatusCode, http.StatusOK)
		assert.Equal(t, stream.ContentType, "application/zip")
		assert.Equal(t, stream.ContentLength, int64(4))
		body, err := io.ReadAll(stream.Body)
		assert.NilError(t, err)
		assert.DeepEqual(t, body, []byte("test"))
		assert.NilError(t, stream.Body.Close())
	})

	t.Run("Returns not found error", func(t *testing.T) {
		t.Parallel()

		const packageID = "99999999-9999-4999-8999-999999999999"

		p := newStoragePipelineForDownloadTest(t, &fakeRequestAdapter{
			baseURL: "http://example.com/storage-service",
			convertToNativeRequest: func(ctx context.Context, requestInfo *kabs.RequestInformation) (any, error) {
				return http.NewRequestWithContext(ctx, requestInfo.Method.String(), "http://example.com/storage-service/api/v2/file/"+packageID+"/download/", nil)
			},
		}, roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"message":"not found"}`)),
			}, nil
		}))

		stream, err := p.DownloadStoragePackage(context.Background(), packageID)

		assert.Assert(t, stream == nil)
		assert.ErrorIs(t, err, ErrStoragePackageNotFound)
	})

	t.Run("Rejects invalid package UUID", func(t *testing.T) {
		t.Parallel()

		p := newStoragePipelineForDownloadTest(t, &fakeRequestAdapter{}, nil)

		stream, err := p.DownloadStoragePackage(context.Background(), "not-a-uuid")

		assert.Assert(t, stream == nil)
		assert.ErrorContains(t, err, `invalid storage package UUID "not-a-uuid"`)
	})
}

func TestParseStorageServiceURL(t *testing.T) {
	t.Parallel()

	baseURL, username, key, err := parseStorageServiceURL("http://user:key@example.com/storage-service/")

	assert.NilError(t, err)
	assert.DeepEqual(t, baseURL, &url.URL{
		Scheme: "http",
		Host:   "example.com",
		Path:   "/storage-service/",
	})
	assert.Equal(t, username, "user")
	assert.Equal(t, key, "key")
}

func TestDefaultHTTPClientsUseExpectedTimeouts(t *testing.T) {
	t.Parallel()

	assert.Equal(t, defaultArchivematicaHTTPClient(nil).Timeout, 10*time.Second)
	assert.Equal(t, defaultStorageServiceHTTPClient(nil).Timeout, 30*time.Second)
}

func newStoragePipelineForTest(t *testing.T, adapter *fakeRequestAdapter) *Pipeline {
	t.Helper()

	return newStoragePipelineForClientTest(t, adapter, &http.Client{})
}

func newStoragePipelineForDownloadTest(t *testing.T, adapter *fakeRequestAdapter, transport http.RoundTripper) *Pipeline {
	t.Helper()

	return newStoragePipelineForClientTest(t, adapter, &http.Client{Transport: transport})
}

func newStoragePipelineForClientTest(t *testing.T, adapter *fakeRequestAdapter, httpClient *http.Client) *Pipeline {
	t.Helper()

	p, err := NewPipeline(logr.Discard(), Config{
		ID:                "pipeline-1",
		StorageServiceURL: "http://user:key@example.com/storage-service/",
	}, nil, httpClient)
	assert.NilError(t, err)

	client, err := ssclient.New(ssclient.Config{
		BaseURL:    "http://example.com/storage-service",
		Username:   "user",
		Key:        "key",
		HTTPClient: httpClient,
	})
	assert.NilError(t, err)

	if adapter.baseURL == "" {
		adapter.baseURL = "http://example.com/storage-service"
	}
	client.Raw().RequestAdapter = adapter
	p.storageServiceClient = client

	return p
}

type fakeRequestAdapter struct {
	send                       func(context.Context, *kabs.RequestInformation, serialization.ParsableFactory, kabs.ErrorMappings) (serialization.Parsable, error)
	convertToNativeRequest     func(context.Context, *kabs.RequestInformation) (any, error)
	baseURL                    string
	serializationWriterFactory serialization.SerializationWriterFactory
}

func (f *fakeRequestAdapter) Send(ctx context.Context, requestInfo *kabs.RequestInformation, constructor serialization.ParsableFactory, errorMappings kabs.ErrorMappings) (serialization.Parsable, error) {
	if f.send == nil {
		panic("unexpected Send call")
	}
	return f.send(ctx, requestInfo, constructor, errorMappings)
}

func (f *fakeRequestAdapter) SendEnum(context.Context, *kabs.RequestInformation, serialization.EnumFactory, kabs.ErrorMappings) (any, error) {
	panic("unexpected SendEnum call")
}

func (f *fakeRequestAdapter) SendCollection(context.Context, *kabs.RequestInformation, serialization.ParsableFactory, kabs.ErrorMappings) ([]serialization.Parsable, error) {
	panic("unexpected SendCollection call")
}

func (f *fakeRequestAdapter) SendEnumCollection(context.Context, *kabs.RequestInformation, serialization.EnumFactory, kabs.ErrorMappings) ([]any, error) {
	panic("unexpected SendEnumCollection call")
}

func (f *fakeRequestAdapter) SendPrimitive(context.Context, *kabs.RequestInformation, string, kabs.ErrorMappings) (any, error) {
	panic("unexpected SendPrimitive call")
}

func (f *fakeRequestAdapter) SendPrimitiveCollection(context.Context, *kabs.RequestInformation, string, kabs.ErrorMappings) ([]any, error) {
	panic("unexpected SendPrimitiveCollection call")
}

func (f *fakeRequestAdapter) SendNoContent(context.Context, *kabs.RequestInformation, kabs.ErrorMappings) error {
	panic("unexpected SendNoContent call")
}

func (f *fakeRequestAdapter) GetSerializationWriterFactory() serialization.SerializationWriterFactory {
	return f.serializationWriterFactory
}

func (f *fakeRequestAdapter) EnableBackingStore(store.BackingStoreFactory) {}

func (f *fakeRequestAdapter) SetBaseUrl(baseURL string) {
	f.baseURL = baseURL
}

func (f *fakeRequestAdapter) GetBaseUrl() string {
	return f.baseURL
}

func (f *fakeRequestAdapter) ConvertToNativeRequest(ctx context.Context, requestInfo *kabs.RequestInformation) (any, error) {
	if f.convertToNativeRequest == nil {
		panic("unexpected ConvertToNativeRequest call")
	}
	return f.convertToNativeRequest(ctx, requestInfo)
}

func uuidPtr(value string) *uuid.UUID {
	id := uuid.MustParse(value)
	return &id
}

func stringPtr(value string) *string {
	return &value
}

func pathUUIDString(t *testing.T, requestInfo *kabs.RequestInformation) string {
	t.Helper()

	value, ok := requestInfo.PathParameters["uuid"]
	assert.Assert(t, ok)

	return value
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}
