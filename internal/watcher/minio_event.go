package watcher

// MinioEvent represents the event delivered by Minio (S3) via Redis.
//
// For reference:
//
//	{
//	    "eventVersion": "2.0",
//	    "eventSource": "minio:s3",
//	    "awsRegion": "",
//	    "eventTime": "2019-10-01T15:28:22Z",
//	    "eventName": "s3:ObjectCreated:CompleteMultipartUpload",
//	    "userIdentity": {
//	        "principalId": "36J9X8EZI4KEV1G7EHXA"
//	    },
//	    "requestParameters": {
//	        "accessKey": "36J9X8EZI4KEV1G7EHXA",
//	        "region": "",
//	        "sourceIPAddress": "172.20.0.1"
//	    },
//	    "responseElements": {
//	        "content-length": "291",
//	        "x-amz-request-id": "15C98F7AC9D60CA6",
//	        "x-minio-deployment-id": "bcc2f9ce-65f2-4558-a455-b8176012f89b",
//	        "x-minio-origin-endpoint": "http://172.20.0.5:9000"
//	    },
//	    "s3": {
//	        "s3SchemaVersion": "1.0",
//	        "configurationId": "Config",
//	        "bucket": {
//	            "name": "sips",
//	            "ownerIdentity": {
//	                "principalId": "36J9X8EZI4KEV1G7EHXA"
//	            },
//	            "arn": "arn:aws:s3:::sips"
//	        },
//	        "object": {
//	            "key": "y25.gif",
//	            "size": 100,
//	            "eTag": "b0814df70de0779da2b0b3f9c676c64d-1",
//	            "contentType": "image/gif",
//	            "userMetadata": {
//	                "X-Minio-Internal-actual-size": "100",
//	                "content-type": "image/gif"
//	            },
//	            "versionId": "1",
//	            "sequencer": "15C98F7ACA94598C"
//	        }
//	    },
//	    "source": {
//	        "host": "172.20.0.1",
//	        "port": "",
//	        "userAgent": "MinIO (linux; amd64) minio-go/v6.0.32 mc/DEVELOPMENT.GOGET"
//	    }
//	}
type MinioEvent struct {
	Name string       `json:"eventName"`
	S3   MinioEventS3 `json:"s3"`
}

func (e MinioEvent) String() string {
	return e.Name
}

type MinioEventS3 struct {
	Bucket MinioEventS3Bucket `json:"bucket"`
	Object MinioEventS3Object `json:"object"`
}

type MinioEventS3Bucket struct {
	Name string `json:"name"`
}

type MinioEventS3Object struct {
	Key string `json:"key"`
}
