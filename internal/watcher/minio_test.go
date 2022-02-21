package watcher_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"gotest.tools/v3/poll"

	"github.com/artefactual-labs/enduro/internal/watcher"
)

func newWatcher(t *testing.T) (*miniredis.Miniredis, watcher.Watcher) {
	t.Helper()

	m, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}

	dur := time.Duration(time.Second)
	config := watcher.MinioConfig{
		Name:             "minio-watcher",
		RedisAddress:     fmt.Sprintf("redis://%s", m.Addr()),
		RedisList:        "minio-events",
		Region:           "eu-south-1",
		Endpoint:         "endpoint",
		PathStyle:        true,
		Profile:          "profile",
		Key:              "key",
		Secret:           "secret",
		Token:            "token",
		Bucket:           "bucket",
		Pipeline:         []string{"am1"},
		RetentionPeriod:  &dur,
		StripTopLevelDir: true,
	}

	var w watcher.Watcher
	w, err = watcher.NewMinioWatcher(context.Background(), &config)
	if err != nil {
		t.Fatal(err)
	}

	return m, w
}

func cleanup(t *testing.T, m *miniredis.Miniredis) {
	m.Close()
}

func TestWatcherReturnsErrWhenNoMessages(t *testing.T) {
	m, w := newWatcher(t)
	defer cleanup(t, m)

	// TODO: slow test, should inject smaller timeout.

	check := func(t poll.LogT) poll.Result {
		_, err := w.Watch(context.Background())

		if err == nil {
			return poll.Error(errors.New("watched did not return an error"))
		}

		if !errors.Is(err, watcher.ErrWatchTimeout) {
			return poll.Error(fmt.Errorf("error not expected: %w", err))
		}

		return poll.Success()
	}

	poll.WaitOn(t, check, poll.WithTimeout(time.Second*3))
}

func TestWatcherReturnsErrOnInvalidMessages(t *testing.T) {
	m, w := newWatcher(t)
	defer cleanup(t, m)

	m.Lpush("minio-events", "{}")

	check := func(t poll.LogT) poll.Result {
		_, err := w.Watch(context.Background())

		if err == nil {
			return poll.Error(errors.New("watched did not return an error"))
		}

		if !strings.Contains(err.Error(), "json: cannot unmarshal object into Go value") {
			return poll.Error(fmt.Errorf("unexpected error: %s", err))
		}

		return poll.Success()
	}

	poll.WaitOn(t, check, poll.WithTimeout(time.Second*3))
}

func TestWatcherReturnsErrOnMessageInWrongBucket(t *testing.T) {
	m, w := newWatcher(t)
	defer cleanup(t, m)

	// Message with a bucket we're not watching.
	m.Lpush("minio-events", `[
	{
		"Event": [
			{
				"awsRegion": "",
				"eventName": "s3:ObjectCreated:Put",
				"eventSource": "minio:s3",
				"eventTime": "2020-04-29T01:00:32Z",
				"eventVersion": "2.0",
				"requestParameters": {
					"accessKey": "12345",
					"region": "",
					"sourceIPAddress": "172.26.0.1"
				},
				"responseElements": {
					"x-amz-request-id": "160A2492E9D053F5",
					"x-minio-deployment-id": "bcc2f9ce-65f2-4558-a455-b8176012f89b",
					"x-minio-origin-endpoint": "http://172.26.0.3:9000"
				},
				"s3": {
					"bucket": {
						"arn": "arn:aws:s3:::one",
						"name": "one",
						"ownerIdentity": {
							"principalId": "36J9X8EZI4KEV1G7EHXA"
						}
					},
					"configurationId": "Config",
					"object": {
						"contentType": "text/plain",
						"eTag": "184826e17f70cb407cafe326f5a48a29",
						"key": "list-email-draft.txt",
						"sequencer": "160A2492EA0BD4B6",
						"size": 1810,
						"userMetadata": {
							"content-type": "text/plain"
						},
						"versionId": "1"
					},
					"s3SchemaVersion": "1.0"
				},
				"source": {
					"host": "172.26.0.1",
					"port": "",
					"userAgent": "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:75.0) Gecko/20100101 Firefox/75.0"
				},
				"userIdentity": {
					"principalId": "36J9X8EZI4KEV1G7EHXA"
				}
			}
		],
		"EventTime": "2020-04-29T01:00:32Z"
	}
]`)

	check := func(t poll.LogT) poll.Result {
		_, err := w.Watch(context.Background())

		if err == nil {
			return poll.Error(errors.New("watched did not return an error"))
		}

		if !errors.Is(err, watcher.ErrBucketMismatch) {
			return poll.Error(fmt.Errorf("error not expected: %w", err))
		}

		return poll.Success()
	}

	poll.WaitOn(t, check, poll.WithTimeout(time.Second*3))
}

func TestWatcherReturnsOnValidMessage(t *testing.T) {
	m, w := newWatcher(t)
	defer cleanup(t, m)

	m.Lpush("minio-events", `[
	{
		"Event": [
			{
				"awsRegion": "",
				"eventName": "s3:ObjectCreated:Put",
				"eventSource": "minio:s3",
				"eventTime": "2020-04-29T01:00:32Z",
				"eventVersion": "2.0",
				"requestParameters": {
					"accessKey": "12345",
					"region": "",
					"sourceIPAddress": "172.26.0.1"
				},
				"responseElements": {
					"x-amz-request-id": "160A2492E9D053F5",
					"x-minio-deployment-id": "bcc2f9ce-65f2-4558-a455-b8176012f89b",
					"x-minio-origin-endpoint": "http://172.26.0.3:9000"
				},
				"s3": {
					"bucket": {
						"arn": "arn:aws:s3:::bucket",
						"name": "bucket",
						"ownerIdentity": {
							"principalId": "36J9X8EZI4KEV1G7EHXA"
						}
					},
					"configurationId": "Config",
					"object": {
						"contentType": "text/plain",
						"eTag": "184826e17f70cb407cafe326f5a48a29",
						"key": "list-email-draft.txt",
						"sequencer": "160A2492EA0BD4B6",
						"size": 1810,
						"userMetadata": {
							"content-type": "text/plain"
						},
						"versionId": "1"
					},
					"s3SchemaVersion": "1.0"
				},
				"source": {
					"host": "172.26.0.1",
					"port": "",
					"userAgent": "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:75.0) Gecko/20100101 Firefox/75.0"
				},
				"userIdentity": {
					"principalId": "36J9X8EZI4KEV1G7EHXA"
				}
			}
		],
		"EventTime": "2020-04-29T01:00:32Z"
	}
]`)

	check := func(t poll.LogT) poll.Result {
		event, err := w.Watch(context.Background())
		if err != nil {
			return poll.Error(fmt.Errorf("watcher return an error unexpectedly: %w", err))
		}
		if event.Bucket != "bucket" || event.Key != "list-email-draft.txt" {
			return poll.Error(fmt.Errorf("received unexpected event attributes (bucket %s, key %s)", event.Bucket, event.Key))
		}

		return poll.Success()
	}

	poll.WaitOn(t, check, poll.WithTimeout(time.Second*3))
}

func TestWatcherReturnsDecodedObjectKey(t *testing.T) {
	m, w := newWatcher(t)
	defer cleanup(t, m)

	// Message with an encoded object key
	m.Lpush("minio-events", `[
	{
		"Event": [
			{
				"s3": {
					"bucket": {
						"name": "bucket"
					},
					"object": {
						"key": "list+%C3%A9mail+draft.txt"
					}
				}
			}
		],
		"EventTime": "2020-04-29T01:00:32Z"
	}
]`)

	check := func(t poll.LogT) poll.Result {
		event, err := w.Watch(context.Background())
		if err != nil {
			return poll.Error(fmt.Errorf("watcher return an error unexpectedly: %w", err))
		}
		if event.Key != "list émail draft.txt" {
			return poll.Error(fmt.Errorf("received unexpected object key %s", event.Key))
		}

		return poll.Success()
	}

	poll.WaitOn(t, check, poll.WithTimeout(time.Second*3))
}

func TestWatcherReturnsErrOnInvalidObjectKey(t *testing.T) {
	m, w := newWatcher(t)
	defer cleanup(t, m)

	// Message with an invalid encoded object key
	m.Lpush("minio-events", `[
	{
		"Event": [
			{
				"s3": {
					"bucket": {
						"name": "bucket"
					},
					"object": {
						"key": "list+%C 3%A9mail+draft.txt"
					}
				}
			}
		],
		"EventTime": "2020-04-29T01:00:32Z"
	}
]`)

	check := func(t poll.LogT) poll.Result {
		_, err := w.Watch(context.Background())

		if err == nil {
			return poll.Error(errors.New("watched did not return an error"))
		}

		// TODO: Check for a custom decode error?

		return poll.Success()
	}

	poll.WaitOn(t, check, poll.WithTimeout(time.Second*3))
}
