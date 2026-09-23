package main

import (
	"context"
	"encoding/xml"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"gotest.tools/v3/assert"
)

func TestSetupBucketRegion(t *testing.T) {
	for _, region := range []string{"us-east-1", "us-west-1"} {
		t.Run(region, func(t *testing.T) {
			var body []byte
			var requests int
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				assert.Equal(t, r.Method, http.MethodPut)
				assert.Equal(t, r.URL.Path, "/sips")
				assert.Equal(t, r.URL.RawQuery, "")
				var err error
				body, err = io.ReadAll(r.Body)
				assert.Check(t, err == nil)
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			client := s3.New(s3.Options{
				Region:       region,
				BaseEndpoint: aws.String(server.URL),
				UsePathStyle: true,
				Credentials:  credentials.NewStaticCredentialsProvider("test", "test-secret", ""),
			})
			assert.NilError(t, setupBucket(context.Background(), client, "sips", ""))
			assert.Equal(t, requests, 1)
			if region == "us-east-1" {
				assert.Equal(t, len(body), 0)
			} else {
				var config struct{ LocationConstraint string }
				assert.NilError(t, xml.Unmarshal(body, &config))
				assert.Equal(t, config.LocationConstraint, region)
			}
		})
	}
}
