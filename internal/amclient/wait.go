package amclient

import (
	"context"
	"time"

	"github.com/cenkalti/backoff/v3"
	"github.com/pkg/errors"
)

const (
	// maxWait determines for how long is our retry strategy willing to wait
	// for the underlying operation to complete.
	maxWait = time.Hour * 8

	// workflowLinkStoreAIPID is the workflow link ID for "Store the AIP".
	// There's some obvious API leaking here. On the bright side, link IDs are
	// unlikely to change as opposed to link labels which may change more
	// frequently.
	workflowLinkStoreAIPID = "20515483-25ed-4133-b23e-5bb14cab8e22"
)

// WaitUntilStored blocks until the AIP generated after a transfer is confirmed
// to be stored. The implementation is based on TransferService.Status and
// JobsService.List.
//
// The retry gives up as soon as one of the following events occur:
// * The caller cancels the context.
// * The total retry period exceeds maxWait.
//
// TODO: we may want to give up earlier in case of specific errors, e.g. if
// TransferService.Status returns errors repeatedly?
func WaitUntilStored(ctx context.Context, c *Client, transferID string) (SIPID string, err error) {
	expBackoff := backoff.NewExponentialBackOff()
	expBackoff.MaxElapsedTime = maxWait
	ctxBackoff := backoff.WithContext(expBackoff, ctx)

	err = backoff.Retry(func() error {

		ctx, cancel := context.WithTimeout(ctx, time.Duration(time.Second*1))
		defer cancel()

		// Retrieve SIPID.
		if SIPID == "" {
			resp, _, err := c.Transfer.Status(ctx, transferID)
			if err != nil {
				return errors.Wrap(err, "TransferService.Status request failed")
			}
			sid, ok := resp.SIP()
			if !ok {
				return errors.New("SIP not created yet")
			}
			SIPID = sid
		}

		// Retrieve status.
		jobs, _, err := c.Jobs.List(ctx, SIPID, &JobsListRequest{
			LinkID: workflowLinkStoreAIPID,
		})
		if err != nil {
			return errors.Wrap(err, "JobsService.List request failed")
		}
		var match *Job
		for _, job := range jobs {
			job := job
			if job.LinkID == workflowLinkStoreAIPID {
				match = &job
				break
			}
		}
		var notStoredYetErr = errors.New("AIP not stored yet")
		if match == nil {
			return notStoredYetErr
		}
		switch match.Status {
		case JobStatusFailed:
			return backoff.Permanent(errors.New("AIP store operation failed"))
		case JobStatusComplete:
			return nil
		default:
			return notStoredYetErr
		}

	}, ctxBackoff)

	return SIPID, err
}
