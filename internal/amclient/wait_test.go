package amclient_test

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/artefactual-labs/enduro/internal/amclient"
)

func ExampleWaitUntilStored() {
	ctx := context.Background()
	client := amclient.NewClient(http.DefaultClient, "http://127.0.0.1:62080/api", "test", "test")

	// Start transfer.
	ctxTimeout, cancel := context.WithTimeout(ctx, time.Hour*2)
	defer cancel()
	payload, _, err := client.Package.Create(ctxTimeout, &amclient.PackageCreateRequest{
		Name:             "images",
		Type:             "standard",
		Path:             "/home/archivematica/archivematica-sampledata/SampleTransfers/Images",
		ProcessingConfig: "automated",
	})
	if err != nil {
		log.Fatal("Package.Create failed: ", err)
	}

	// Wait until the AIP is stored.
	SIPID, err := amclient.WaitUntilStored(ctx, client, payload.ID)
	if err != nil || SIPID == "" {
		log.Fatal("WaitUntilStored failed: ", err)
	}

	fmt.Printf("Transfer stored successfully! AIP %s", SIPID)
}
