# s3put

`s3put` is a small test helper used by the Dagger object-storage smoke tests to
upload files to an S3-compatible endpoint.

The helper exists so the smoke tests can exercise the same object-upload path
against MinIO and SeaweedFS without depending on a provider-specific CLI such as
`mc` or `weed`. It uses the AWS SDK, path-style addressing, explicit static
credentials, and a caller-provided endpoint, which matches the local object
storage services started by Dagger.

Run it from the repository root:

```sh
go run ./hack/s3put -h
```

Example:

```sh
go run ./hack/s3put \
  -endpoint http://127.0.0.1:9000 \
  -region us-west-1 \
  -bucket sips \
  -key transfer.zip \
  -access-key minio \
  -secret-key minio123 \
  -file ./transfer.zip
```

Generate and upload one simple zipped transfer:

```sh
go run ./hack/s3put \
  -endpoint http://127.0.0.1:7460 \
  -region us-west-1 \
  -bucket sips \
  -key issue-681-single.zip \
  -access-key minio \
  -secret-key minio123 \
  -generate-transfer
```

Generate and upload 25 simple zipped transfers:

```sh
go run ./hack/s3put \
  -endpoint http://127.0.0.1:7460 \
  -region us-west-1 \
  -bucket sips \
  -key-prefix issue-681 \
  -count 25 \
  -access-key minio \
  -secret-key minio123 \
  -generate-transfer
```

The multi-upload example creates object keys named `issue-681-001.zip`,
`issue-681-002.zip`, and so on. This is useful when testing pipeline capacity
and queued collection cancellation in the local Enduro development environment.

The command exits with status `0` when the upload succeeds. Any validation,
configuration, or upload failure is written to stderr and exits non-zero.
