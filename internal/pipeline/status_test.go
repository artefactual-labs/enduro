package pipeline

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"go.artefactual.dev/amclient"
	"go.artefactual.dev/amclient/amclienttest"
	"go.uber.org/mock/gomock"
	"gotest.tools/v3/assert"
)

func TestTransferStatus(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tid := "ba006a05-0420-48bc-817c-b50af0cc7793"

	tests := map[string]struct {
		fakefn  func(*amclienttest.MockTransferService)
		wantSID *string
		wantErr error
	}{
		"It returns a ErrStatusRetryable when the context deadline is exceeded": {
			fakefn: func(tsfake *amclienttest.MockTransferService) {
				tsfake.
					EXPECT().
					Status(gomock.Eq(ctx), gomock.Eq(tid)).
					Return(
						nil,
						nil,
						context.DeadlineExceeded,
					)
			},
			wantErr: ErrStatusRetryable,
		},
		"It returns a ErrStatusRetryable when a network timeout error is detected": {
			fakefn: func(tsfake *amclienttest.MockTransferService) {
				tsfake.
					EXPECT().
					Status(gomock.Eq(ctx), gomock.Eq(tid)).
					Return(
						nil,
						nil,
						fakeTimeoutError{},
					)
			},
			wantErr: ErrStatusRetryable,
		},
		"It returns a ErrStatusRetryable when the server returns a 400 error (Status API server error)": {
			fakefn: func(tsfake *amclienttest.MockTransferService) {
				tsfake.
					EXPECT().
					Status(gomock.Eq(ctx), gomock.Eq(tid)).
					Return(
						nil,
						&amclient.Response{Response: &http.Response{StatusCode: 400}},
						&amclient.ErrorResponse{Response: &http.Response{StatusCode: 400}},
					)
			},
			wantErr: ErrStatusRetryable,
		},
		"It returns a ErrStatusNonRetryable when the server returns a 4xx error, e.g. 401 Unauthorized": {
			fakefn: func(tsfake *amclienttest.MockTransferService) {
				tsfake.
					EXPECT().
					Status(gomock.Eq(ctx), gomock.Eq(tid)).
					Return(
						nil,
						&amclient.Response{Response: &http.Response{StatusCode: 401}},
						&amclient.ErrorResponse{Response: &http.Response{StatusCode: 401}},
					)
			},
			wantErr: ErrStatusNonRetryable,
		},
		"It returns a ErrStatusRetryable when the server returns a 503 error, e.g. 503 Bad Gateway": {
			fakefn: func(tsfake *amclienttest.MockTransferService) {
				tsfake.
					EXPECT().
					Status(gomock.Eq(ctx), gomock.Eq(tid)).
					Return(
						nil,
						&amclient.Response{Response: &http.Response{StatusCode: 503}},
						&amclient.ErrorResponse{Response: &http.Response{StatusCode: 503}},
					)
			},
			wantErr: ErrStatusRetryable,
		},
		"It returns a ErrStatusNonRetryable when the server returns a 3xx error": {
			fakefn: func(tsfake *amclienttest.MockTransferService) {
				tsfake.
					EXPECT().
					Status(gomock.Eq(ctx), gomock.Eq(tid)).
					Return(
						nil,
						&amclient.Response{Response: &http.Response{StatusCode: 311}},
						&amclient.ErrorResponse{Response: &http.Response{StatusCode: 311}},
					)
			},
			wantErr: ErrStatusNonRetryable,
		},
		"It returns a ErrStatusNonRetryable when the API returns unmanageable statuses": {
			fakefn: func(tsfake *amclienttest.MockTransferService) {
				tsfake.
					EXPECT().
					Status(gomock.Eq(ctx), gomock.Eq(tid)).
					Return(
						&amclient.TransferStatusResponse{Status: "FAILED"},
						&amclient.Response{},
						nil,
					)
			},
			wantErr: ErrStatusNonRetryable,
		},
		"It returns a ErrStatusNonRetryable when the API returns unknown statuses": {
			fakefn: func(tsfake *amclienttest.MockTransferService) {
				tsfake.
					EXPECT().
					Status(gomock.Eq(ctx), gomock.Eq(tid)).
					Return(
						&amclient.TransferStatusResponse{Status: "UNKNOWN"},
						&amclient.Response{},
						nil,
					)
			},
			wantErr: ErrStatusNonRetryable,
		},
		"It returns a ErrStatusNonRetryable when the API reports empty status": {
			fakefn: func(tsfake *amclienttest.MockTransferService) {
				tsfake.
					EXPECT().
					Status(gomock.Eq(ctx), gomock.Eq(tid)).
					Return(
						&amclient.TransferStatusResponse{Status: ""},
						&amclient.Response{},
						nil,
					)
			},
			wantErr: ErrStatusNonRetryable,
		},
		"It returns a ErrStatusNonRetryable when the API reports that the transfer is in backlog": {
			fakefn: func(tsfake *amclienttest.MockTransferService) {
				tsfake.
					EXPECT().
					Status(gomock.Eq(ctx), gomock.Eq(tid)).
					Return(
						&amclient.TransferStatusResponse{
							Status: "COMPLETE",
							SIPID:  "BACKLOG",
						},
						&amclient.Response{},
						nil,
					)
			},
			wantErr: ErrStatusNonRetryable,
		},
		"It returns a ErrStatusNonRetryable when the API reports that the transfer requires user input": {
			fakefn: func(tsfake *amclienttest.MockTransferService) {
				tsfake.
					EXPECT().
					Status(gomock.Eq(ctx), gomock.Eq(tid)).
					Return(
						&amclient.TransferStatusResponse{Status: "USER_INPUT"},
						&amclient.Response{},
						nil,
					)
			},
			wantErr: ErrStatusNonRetryable,
		},
		"It returns a ErrStatusInProgress when the API reports in-progress status": {
			fakefn: func(tsfake *amclienttest.MockTransferService) {
				tsfake.
					EXPECT().
					Status(gomock.Eq(ctx), gomock.Eq(tid)).
					Return(
						&amclient.TransferStatusResponse{Status: "PROCESSING"},
						&amclient.Response{},
						nil,
					)
			},
			wantErr: ErrStatusInProgress,
		},
		"It returns a ErrStatusInProgress when the API reports in-progress status (2)": {
			fakefn: func(tsfake *amclienttest.MockTransferService) {
				tsfake.
					EXPECT().
					Status(gomock.Eq(ctx), gomock.Eq(tid)).
					Return(
						&amclient.TransferStatusResponse{
							Status: "COMPLETE",
							SIPID:  "",
						},
						&amclient.Response{},
						nil,
					)
			},
			wantErr: ErrStatusInProgress,
		},
		"It returns the SIP ID when the API reports completion status": {
			fakefn: func(tsfake *amclienttest.MockTransferService) {
				tsfake.
					EXPECT().
					Status(gomock.Eq(ctx), gomock.Eq(tid)).
					Return(
						&amclient.TransferStatusResponse{
							Status: "COMPLETE",
							SIPID:  "fb901f48-8d38-4e1b-8047-6e03a0079439",
						},
						&amclient.Response{},
						nil,
					)
			},
			wantSID: strPtr("fb901f48-8d38-4e1b-8047-6e03a0079439"),
			wantErr: nil,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			transferFake := amclienttest.NewMockTransferService(ctrl)
			tc.fakefn(transferFake)

			sid, err := TransferStatus(ctx, transferFake, tid)

			if tc.wantSID != nil {
				assert.Equal(t, sid, *tc.wantSID)
			}

			assert.Equal(t, errors.Is(err, tc.wantErr), true)
		})
	}
}

func TestIngestStatus(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	sid := "ba006a05-0420-48bc-817c-b50af0cc7793"

	tests := map[string]struct {
		fakefn  func(*amclienttest.MockIngestService)
		wantErr error
	}{
		"It returns a ErrStatusRetryable when the context deadline is exceeded": {
			fakefn: func(isfake *amclienttest.MockIngestService) {
				isfake.
					EXPECT().
					Status(gomock.Eq(ctx), gomock.Eq(sid)).
					Return(
						nil,
						nil,
						context.DeadlineExceeded,
					)
			},
			wantErr: ErrStatusRetryable,
		},
		"It returns a ErrStatusRetryable when a network timeout error is detected": {
			fakefn: func(tsfake *amclienttest.MockIngestService) {
				tsfake.
					EXPECT().
					Status(gomock.Eq(ctx), gomock.Eq(sid)).
					Return(
						nil,
						nil,
						fakeTimeoutError{},
					)
			},
			wantErr: ErrStatusRetryable,
		},
		"It returns a ErrStatusRetryable when the server returns a 400 error (Status API server error)": {
			fakefn: func(isfake *amclienttest.MockIngestService) {
				isfake.
					EXPECT().
					Status(gomock.Eq(ctx), gomock.Eq(sid)).
					Return(
						nil,
						&amclient.Response{Response: &http.Response{StatusCode: 400}},
						&amclient.ErrorResponse{Response: &http.Response{StatusCode: 400}},
					)
			},
			wantErr: ErrStatusRetryable,
		},
		"It returns a ErrStatusNonRetryable when the server returns a 4xx error, e.g. 401 Unauthorized": {
			fakefn: func(isfake *amclienttest.MockIngestService) {
				isfake.
					EXPECT().
					Status(gomock.Eq(ctx), gomock.Eq(sid)).
					Return(
						nil,
						&amclient.Response{Response: &http.Response{StatusCode: 401}},
						&amclient.ErrorResponse{Response: &http.Response{StatusCode: 401}},
					)
			},
			wantErr: ErrStatusNonRetryable,
		},
		"It returns a ErrStatusRetryable when the server returns a 5xx error, e.g. 503 Service Unavailable": {
			fakefn: func(isfake *amclienttest.MockIngestService) {
				isfake.
					EXPECT().
					Status(gomock.Eq(ctx), gomock.Eq(sid)).
					Return(
						nil,
						&amclient.Response{Response: &http.Response{StatusCode: 503}},
						&amclient.ErrorResponse{Response: &http.Response{StatusCode: 503}},
					)
			},
			wantErr: ErrStatusRetryable,
		},
		"It returns a ErrStatusNonRetryable when the server returns a 3xx error": {
			fakefn: func(isfake *amclienttest.MockIngestService) {
				isfake.
					EXPECT().
					Status(gomock.Eq(ctx), gomock.Eq(sid)).
					Return(
						nil,
						&amclient.Response{Response: &http.Response{StatusCode: 311}},
						&amclient.ErrorResponse{Response: &http.Response{StatusCode: 311}},
					)
			},
			wantErr: ErrStatusNonRetryable,
		},
		"It returns a ErrStatusNonRetryable when the API reports unknown statuses": {
			fakefn: func(isfake *amclienttest.MockIngestService) {
				isfake.
					EXPECT().
					Status(gomock.Eq(ctx), gomock.Eq(sid)).
					Return(
						&amclient.IngestStatusResponse{Status: "UNKNOWN"},
						&amclient.Response{},
						nil,
					)
			},
			wantErr: ErrStatusNonRetryable,
		},
		"It returns a ErrStatusNonRetryable when the API reports empty status": {
			fakefn: func(isfake *amclienttest.MockIngestService) {
				isfake.
					EXPECT().
					Status(gomock.Eq(ctx), gomock.Eq(sid)).
					Return(
						&amclient.IngestStatusResponse{Status: ""},
						&amclient.Response{},
						nil,
					)
			},
			wantErr: ErrStatusNonRetryable,
		},
		"It returns ErrStatusNonRetryable when the API reports USER_INPUT status": {
			fakefn: func(isfake *amclienttest.MockIngestService) {
				isfake.
					EXPECT().
					Status(gomock.Eq(ctx), gomock.Eq(sid)).
					Return(
						&amclient.IngestStatusResponse{Status: "USER_INPUT"},
						&amclient.Response{},
						nil,
					)
			},
			wantErr: ErrStatusNonRetryable,
		},
		"It returns ErrStatusNonRetryable when the API reports FAILED status": {
			fakefn: func(isfake *amclienttest.MockIngestService) {
				isfake.
					EXPECT().
					Status(gomock.Eq(ctx), gomock.Eq(sid)).
					Return(
						&amclient.IngestStatusResponse{Status: "FAILED"},
						&amclient.Response{},
						nil,
					)
			},
			wantErr: ErrStatusNonRetryable,
		},
		"It returns ErrStatusNonRetryable when the API reports REJECTED status": {
			fakefn: func(isfake *amclienttest.MockIngestService) {
				isfake.
					EXPECT().
					Status(gomock.Eq(ctx), gomock.Eq(sid)).
					Return(
						&amclient.IngestStatusResponse{Status: "REJECTED"},
						&amclient.Response{},
						nil,
					)
			},
			wantErr: ErrStatusNonRetryable,
		},
		"It returns ErrStatusInProgress when the API reports in-progress status": {
			fakefn: func(isfake *amclienttest.MockIngestService) {
				isfake.
					EXPECT().
					Status(gomock.Eq(ctx), gomock.Eq(sid)).
					Return(
						&amclient.IngestStatusResponse{Status: "PROCESSING"},
						&amclient.Response{},
						nil,
					)
			},
			wantErr: ErrStatusInProgress,
		},
		"It returns nil when the API reports completion status": {
			fakefn: func(isfake *amclienttest.MockIngestService) {
				isfake.
					EXPECT().
					Status(gomock.Eq(ctx), gomock.Eq(sid)).
					Return(
						&amclient.IngestStatusResponse{Status: "COMPLETE"},
						&amclient.Response{},
						nil,
					)
			},
			wantErr: nil,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			ingestFake := amclienttest.NewMockIngestService(ctrl)
			tc.fakefn(ingestFake)

			err := IngestStatus(ctx, ingestFake, sid)

			assert.Equal(t, errors.Is(err, tc.wantErr), true)
		})
	}
}

func strPtr(str string) *string {
	return &str
}

type fakeTimeoutError struct{}

func (fakeTimeoutError) Timeout() bool   { return true }
func (fakeTimeoutError) Temporary() bool { return true }
func (fakeTimeoutError) Error() string   { return "fake timeout error" }
