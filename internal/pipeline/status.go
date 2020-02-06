package pipeline

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/artefactual-labs/enduro/internal/amclient"
)

var (
	ErrStatusNonRetryable = errors.New("non retryable error")
	ErrStatusRetryable    = errors.New("retryable error")
	ErrStatusInProgress   = errors.New("waitable error")
)

// TransferStatus returns a non-nil error when the transfer is not fully transferred.
func TransferStatus(ctx context.Context, client *amclient.Client, ID string) (string, error) {
	status, _, err := client.Transfer.Status(ctx, ID)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || os.IsTimeout(err) {
			return "", fmt.Errorf("error checking transfer status (%w): %v", ErrStatusRetryable, err)
		}
		if err, ok := err.(*amclient.ErrorResponse); ok {
			if err.Response.StatusCode >= 400 {
				return "", fmt.Errorf("error checking transfer status (%w): %v (%d)", ErrStatusRetryable, err, err.Response.StatusCode)
			} else {
				return "", fmt.Errorf("error checking transfer status (%w): %v (%d)", ErrStatusNonRetryable, err, err.Response.StatusCode)
			}
		}
		return "", fmt.Errorf("error checking transfer status (%w): %v", ErrStatusNonRetryable, err)
	}

	if status.Status == "" {
		return "", fmt.Errorf("error checking transfer status (%w): status is empty", ErrStatusNonRetryable)
	}

	switch {

	// State that we can't handle.
	default:
		fallthrough
	case status.Status == "COMPLETE" && status.SIPID == "BACKLOG":
		// TODO: (not in POC) SIP arrangement.
		fallthrough
	case status.Status == "USER_INPUT":
		// TODO: (not in POC) User interactions with workflow.
		fallthrough
	case status.Status == "FAILED" || status.Status == "REJECTED":
		return "", fmt.Errorf("error checking transfer status (%w): transfer is in a state that we can't handle: %s", ErrStatusNonRetryable, status.Status)

	// Processing state where we want to keep waiting.
	case status.Status == "COMPLETE" && status.SIPID == "":
		// It is possible (due to https://github.com/archivematica/Issues/issues/690),
		// that AM tells us that the transfer completed but the SIPID field is
		// not populated.
		fallthrough
	case status.Status == "PROCESSING":
		return "", ErrStatusInProgress

	// Success!
	case status.Status == "COMPLETE":
		return status.SIPID, nil

	}
}

// IngestStatus returns a non-nil error when the SIP is not fully ingested.
func IngestStatus(ctx context.Context, client *amclient.Client, ID string) error {
	status, _, err := client.Ingest.Status(ctx, ID)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || os.IsTimeout(err) {
			return fmt.Errorf("error checking ingest status (%w): %v", ErrStatusRetryable, err)
		}
		if err, ok := err.(*amclient.ErrorResponse); ok {
			if err.Response.StatusCode >= 400 {
				return fmt.Errorf("error checking ingest status (%w): %v (%d)", ErrStatusRetryable, err, err.Response.StatusCode)
			} else {
				return fmt.Errorf("error checking ingest status (%w): %v (%d)", ErrStatusNonRetryable, err, err.Response.StatusCode)
			}
		}
		return fmt.Errorf("error checking ingest status (%w): %v", ErrStatusNonRetryable, err)
	}

	if status.Status == "" {
		return fmt.Errorf("error checking ingest status (%w): status is empty", ErrStatusNonRetryable)
	}

	switch {

	default:
		fallthrough
	case status.Status == "USER_INPUT" || status.Status == "FAILED" || status.Status == "REJECTED":
		// TODO: (not in POC) User interactions with workflow.
		return fmt.Errorf("error checking ingest status (%w): ingest is in a state that we can't handle: %s", ErrStatusNonRetryable, status.Status)
	case status.Status == "PROCESSING":
		return ErrStatusInProgress
	case status.Status == "COMPLETE":
		return nil

	}
}
