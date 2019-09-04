package amclient

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJobs_List(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/api/v2beta/jobs/e99afef7-90c5-4fd9-bf8f-bed13b3bd4ba/", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		fmt.Fprint(w, `[
	{
		"uuid": "624581dc-ec01-4195-9da3-db0ab0ad1cc3",
		"name": "Check transfer directory for objects",
		"status": "COMPLETE",
		"microservice": "Create SIP from Transfer",
		"link_uuid": "6ce08b95-1b3b-498a-8baa-e595e2ae7466",
		"tasks": [
			{
				"uuid": "491aebbd-457b-4a6e-adf6-87a3a9ee951a",
				"exit_code": 1
			}
		]
	}
]`)
	})

	payload, _, err := client.Jobs.List(ctx, "e99afef7-90c5-4fd9-bf8f-bed13b3bd4ba", &JobsListRequest{
		LinkID: "6ce08b95-1b3b-498a-8baa-e595e2ae7466",
	})

	assert.NoError(t, err)
	assert.Equal(t, []Job{
		{
			ID:           "624581dc-ec01-4195-9da3-db0ab0ad1cc3",
			Name:         "Check transfer directory for objects",
			Status:       JobStatusComplete,
			Microservice: "Create SIP from Transfer",
			LinkID:       "6ce08b95-1b3b-498a-8baa-e595e2ae7466",
			Tasks: []Task{
				{
					ID:       "491aebbd-457b-4a6e-adf6-87a3a9ee951a",
					ExitCode: 1,
				},
			},
		},
	}, payload)
}
