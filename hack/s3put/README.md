# s3put

`s3put` is a small test helper used by the Dagger object-storage smoke tests to
upload one file to an S3-compatible endpoint.

The helper exists so the smoke tests can exercise the same object-upload path
against MinIO and SeaweedFS without depending on a provider-specific CLI such as
`mc` or `weed`. It uses the AWS SDK, path-style addressing, explicit static
credentials, and a caller-provided endpoint, which matches the local object
storage services started by Dagger.

Build it from the repository root:

```sh
go build -trimpath -o /tmp/s3put ./hack/s3put
```

Example:

```sh
/tmp/s3put \
  -endpoint http://127.0.0.1:9000 \
  -region us-west-1 \
  -bucket sips \
  -key transfer.zip \
  -access-key minio \
  -secret-key minio123 \
  -file ./transfer.zip
```

The command exits with status `0` when the upload succeeds. Any validation,
configuration, or upload failure is written to stderr and exits non-zero.
