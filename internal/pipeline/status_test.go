package pipeline

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/artefactual-labs/enduro/internal/amclient"
	amclientfake "github.com/artefactual-labs/enduro/internal/amclient/fake"
	"github.com/golang/mock/gomock"
	"gotest.tools/v3/assert"
)

func TestTransferStatus(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tid := "ba006a05-0420-48bc-817c-b50af0cc7793"

	tests := map[string]struct {
		fakefn  func(*amclientfake.MockTransferService)
		wantSID *string
		wantErr error
	}{
		"It returns a ErrStatusRetryable when the context deadline is exceeded": {
			fakefn: func(tsfake *amclientfake.MockTransferService) {
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
		"It returns a ErrStatusRetryable when the server returns a 4xx/5xx error, e.g. 401 Unauthorized": {
			fakefn: func(tsfake *amclientfake.MockTransferService) {
				tsfake.
					EXPECT().
					Status(gomock.Eq(ctx), gomock.Eq(tid)).
					Return(
						nil,
						&amclient.Response{Response: &http.Response{StatusCode: 401}},
						&amclient.ErrorResponse{Response: &http.Response{StatusCode: 401}},
					)
			},
			wantErr: ErrStatusRetryable,
		},
		"It returns a ErrStatusNonRetryable when the server returns a 3xx error": {
			fakefn: func(tsfake *amclientfake.MockTransferService) {
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
			fakefn: func(tsfake *amclientfake.MockTransferService) {
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
			fakefn: func(tsfake *amclientfake.MockTransferService) {
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
			fakefn: func(tsfake *amclientfake.MockTransferService) {
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
			fakefn: func(tsfake *amclientfake.MockTransferService) {
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
			fakefn: func(tsfake *amclientfake.MockTransferService) {
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
			fakefn: func(tsfake *amclientfake.MockTransferService) {
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
			fakefn: func(tsfake *amclientfake.MockTransferService) {
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
			fakefn: func(tsfake *amclientfake.MockTransferService) {
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
		tc := tc

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			tsfake := amclientfake.NewMockTransferService(ctrl)
			amc := amclient.NewClient(nil, "", "", "")
			amc.Transfer = tsfake

			tc.fakefn(tsfake)

			sid, err := TransferStatus(ctx, amc, tid)

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
		fakefn  func(*amclientfake.MockIngestService)
		wantErr error
	}{
		"It returns a ErrStatusRetryable when the context deadline is exceeded": {
			fakefn: func(isfake *amclientfake.MockIngestService) {
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
		"It returns a ErrStatusRetryable when the server returns a 4xx/5xx error, e.g. 401 Unauthorized": {
			fakefn: func(isfake *amclientfake.MockIngestService) {
				isfake.
					EXPECT().
					Status(gomock.Eq(ctx), gomock.Eq(sid)).
					Return(
						nil,
						&amclient.Response{Response: &http.Response{StatusCode: 401}},
						&amclient.ErrorResponse{Response: &http.Response{StatusCode: 401}},
					)
			},
			wantErr: ErrStatusRetryable,
		},
		"It returns a ErrStatusNonRetryable when the server returns a 3xx error": {
			fakefn: func(isfake *amclientfake.MockIngestService) {
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
			fakefn: func(isfake *amclientfake.MockIngestService) {
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
			fakefn: func(isfake *amclientfake.MockIngestService) {
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
			fakefn: func(isfake *amclientfake.MockIngestService) {
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
			fakefn: func(isfake *amclientfake.MockIngestService) {
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
			fakefn: func(isfake *amclientfake.MockIngestService) {
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
			fakefn: func(isfake *amclientfake.MockIngestService) {
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
			fakefn: func(isfake *amclientfake.MockIngestService) {
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
		tc := tc

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			isfake := amclientfake.NewMockIngestService(ctrl)
			amc := amclient.NewClient(nil, "", "", "")
			amc.Ingest = isfake

			tc.fakefn(isfake)

			err := IngestStatus(ctx, amc, sid)

			assert.Equal(t, errors.Is(err, tc.wantErr), true)
		})
	}
}

func strPtr(str string) *string {
	return &str
}
