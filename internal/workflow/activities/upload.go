package activities

import (
	"context"
	"io"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"gocloud.dev/blob/s3blob"

	"github.com/artefactual-labs/enduro/internal/aipstore"
)

type UploadActivity struct {
	config aipstore.Config
}

func NewUploadActivity(config aipstore.Config) *UploadActivity {
	return &UploadActivity{config: config}
}

func (a *UploadActivity) Execute(ctx context.Context, AIPPath string) error {
	sessOpts := session.Options{}
	sessOpts.Config.WithRegion(a.config.Region)
	sessOpts.Config.WithEndpoint(a.config.Endpoint)
	sessOpts.Config.WithS3ForcePathStyle(a.config.PathStyle)
	sessOpts.Config.WithCredentials(
		credentials.NewStaticCredentials(
			a.config.Key, a.config.Secret, a.config.Token,
		),
	)
	sess, err := session.NewSessionWithOptions(sessOpts)
	if err != nil {
		return err
	}

	bucket, err := s3blob.OpenBucket(ctx, sess, a.config.Bucket, nil)
	if err != nil {
		return err
	}

	defer bucket.Close()

	name := filepath.Base(AIPPath)

	// Open the key "foo.txt" for writing with the default options.
	w, err := bucket.NewWriter(ctx, name, nil)
	if err != nil {
		return err
	}

	f, err := os.Open(AIPPath)
	if err != nil {
		return err
	}

	defer f.Close()

	// TODO: Does this return when the context is canceled?
	_, copyErr := io.Copy(w, f)

	closeErr := w.Close()

	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}

	return nil
}
