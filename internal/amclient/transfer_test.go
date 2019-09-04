package amclient

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTransfer_Start(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/api/transfer/start_transfer/", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")
		fmt.Fprint(w, `{"message": "Copy successful", "path": "/var/foobar"}`)
	})

	payload, _, err := client.Transfer.Start(ctx, &TransferStartRequest{
		Name:  "foobar",
		Paths: []string{"a.jpg", "b.jpg"},
		Type:  "standard",
	})
	if err != nil {
		t.Errorf("Transfer.Start returned error: %v", err)
	}
	if want, got := "Copy successful", payload.Message; want != got {
		t.Errorf("Transfer.Start(): Message = %v, want %v", got, want)
	}
}

func TestTransfer_Approve(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/api/transfer/approve/", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")
		fmt.Fprint(w, `{"message": "Approval successful.", "uuid": "eaedbee3-2b02-4e40-baa0-3ef92c5fd17e"}`)
	})

	payload, _, err := client.Transfer.Approve(ctx, &TransferApproveRequest{
		Directory: "Foobar",
		Type:      "standard",
	})
	if err != nil {
		t.Errorf("Transfer.Approve returned error: %v", err)
	}
	if want, got := "Approval successful.", payload.Message; want != got {
		t.Errorf("Transfer.Approve(): Message = %v, want %v", got, want)
	}
	if want, got := "eaedbee3-2b02-4e40-baa0-3ef92c5fd17e", payload.UUID; want != got {
		t.Errorf("Transfer.Approve(): UUID = %v, want %v", got, want)
	}
}

func TestTransfer_Unapproved(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/api/transfer/unapproved/", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		fmt.Fprint(w, `{
			"message": "Fetched unapproved transfers successfully.",
			"results": [
				{
					"type": "standard",
					"directory": "/var/foobar1",
					"uuid": "eaedbee3-2b02-4e40-baa0-3ef92c5fd17e"
				},
				{
					"type": "standard",
					"directory": "/var/foobar2",
					"uuid": "433f20e4-a0e4-484b-8fb4-ec9b3cda4cfc"
				}
			]
		}`)
	})

	payload, _, err := client.Transfer.Unapproved(ctx, &TransferUnapprovedRequest{})
	if err != nil {
		t.Errorf("Transfer.Unapproved() returned error: %v", err)
	}
	if want, got := 2, len(payload.Results); want != got {
		t.Errorf("Transfer.Unapproved() len(Results) %v, want %v", got, want)
	}
}

func TestTransfer_Status(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/api/transfer/status/52dd0c01-e803-423a-be5f-b592b5d5d61c/", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		fmt.Fprint(w, `{
	"status": "COMPLETE",
	"name": "imgs",
	"sip_uuid": "41699e73-ec9e-4240-b153-71f4155e7da4",
	"microservice": "Microservice group name",
	"directory": "imgs-52dd0c01-e803-423a-be5f-b592b5d5d61c",
	"path": "/var/archivematica/sharedDirectory/watchedDirectories/SIPCreation/completedTransfers/imgs-52dd0c01-e803-423a-be5f-b592b5d5d61c/",
	"message": "Fetched status for 52dd0c01-e803-423a-be5f-b592b5d5d61c successfully.",
	"type": "transfer",
	"uuid": "52dd0c01-e803-423a-be5f-b592b5d5d61c"
}`)
	})

	payload, _, err := client.Transfer.Status(ctx, "52dd0c01-e803-423a-be5f-b592b5d5d61c")
	if err != nil {
		t.Errorf("Transfer.Status() returned error: %v", err)
	}

	assert.NoError(t, err)
	assert.Equal(t, &TransferStatusResponse{
		ID:           "52dd0c01-e803-423a-be5f-b592b5d5d61c",
		Status:       "COMPLETE",
		Name:         "imgs",
		SIPID:        "41699e73-ec9e-4240-b153-71f4155e7da4",
		Microservice: "Microservice group name",
		Directory:    "imgs-52dd0c01-e803-423a-be5f-b592b5d5d61c",
		Path:         "/var/archivematica/sharedDirectory/watchedDirectories/SIPCreation/completedTransfers/imgs-52dd0c01-e803-423a-be5f-b592b5d5d61c/",
		Message:      "Fetched status for 52dd0c01-e803-423a-be5f-b592b5d5d61c successfully.",
		Type:         "transfer",
	}, payload)
}
