package amclient

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTask_Read(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/api/v2beta/task/96acb0a1-525c-456a-9060-51bb84f5f708/", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		fmt.Fprint(w, `{
	"uuid": "96acb0a1-525c-456a-9060-51bb84f5f708",
	"exit_code": 1,
	"file_uuid": "e502c0d9-becf-455a-8b20-091526947a09",
	"file_name": "foobar.txt",
	"time_created": "2019-06-18T00:00:00",
	"time_started": "2019-07-18T00:00:00",
	"time_ended": "2019-08-18T00:00:00",
	"duration": 4294967295
}`)
	})

	payload, _, err := client.Task.Read(ctx, "96acb0a1-525c-456a-9060-51bb84f5f708")

	assert.NoError(t, err)
	assert.Equal(t, &TaskDetailed{
		ID:          "96acb0a1-525c-456a-9060-51bb84f5f708",
		ExitCode:    1,
		FileID:      "e502c0d9-becf-455a-8b20-091526947a09",
		TimeCreated: TaskDateTime{Time: time.Date(2019, time.June, 18, 0, 0, 0, 0, time.UTC)},
		TimeStarted: TaskDateTime{Time: time.Date(2019, time.July, 18, 0, 0, 0, 0, time.UTC)},
		TimeEnded:   TaskDateTime{Time: time.Date(2019, time.August, 18, 0, 0, 0, 0, time.UTC)},
		Filename:    "foobar.txt",
		Duration:    4294967295,
	}, payload)
}
