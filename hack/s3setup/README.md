# s3setup

`s3setup` is a Dagger smoke-test helper. It creates the test bucket and, when
requested, writes the bucket notification configuration through the S3 API.

The `minio-latest` smoke permutation uses it because newer MinIO images are
configured with notification targets through environment variables, while the
bucket-to-target notification rule still needs to be created before the S3
watcher test uploads its transfer.
