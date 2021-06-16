package pipeline

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/artefactual-labs/enduro/internal/amclient"
)

var (
	ErrStatusNonRetryable = errors.New("non retryable error")
	ErrStatusRetryable    = errors.New("retryable error")
	ErrStatusInProgress   = errors.New("waitable error")
)

// processStatusError enriches errors returned by Transfer.Status and
// Ingest.Status and determines whether they should be retried.
func processStatusError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("%v (%w)", err, ErrStatusRetryable)
	}

	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		return fmt.Errorf("network error (%w): %v", ErrStatusRetryable, netErr)
	}

	if amErr, ok := err.(*amclient.ErrorResponse); ok {
		switch {
		case amErr.Response.StatusCode == 400:
			fallthrough // Server error that Archivematica masks as a 400.
		case amErr.Response.StatusCode >= 500:
			return fmt.Errorf("server error (%w): %v (%d)", ErrStatusRetryable, err, amErr.Response.StatusCode)
		case amErr.Response.StatusCode >= 401 && amErr.Response.StatusCode < 500:
			return fmt.Errorf("server error (%w): %v (%d)", ErrStatusNonRetryable, err, amErr.Response.StatusCode)
		}
	}

	return fmt.Errorf("unknown error (%w): %v (%T)", ErrStatusNonRetryable, err, err)
}

// TransferStatus returns a non-nil error when the transfer is not fully transferred.
func TransferStatus(ctx context.Context, transferService amclient.TransferService, ID string) (string, error) {
	status, _, err := transferService.Status(ctx, ID)

	if err := processStatusError(err); err != nil {
		return "", fmt.Errorf("error checking transfer status: %w", err)
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
func IngestStatus(ctx context.Context, ingestService amclient.IngestService, ID string) error {
	status, _, err := ingestService.Status(ctx, ID)

	if err := processStatusError(err); err != nil {
		return fmt.Errorf("error checking ingest status: %w", err)
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
