package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"

	"github.com/google/uuid"
)

const (
	// How many records we want to generate.
	datasetSize = 2500000

	// Size of each batch that we write to the CSV.
	batchSize = 100
)

// A list of pipelines that we pretend to know. This is to
// avoid randomness in a column where cardinality is likely
// going to be low.
var pipelineIDs []string = []string{
	"687ec2f4-3ec8-45b0-965a-104cbd77a657",
	"c5b11a76-ffe2-4de3-9dd6-8f62ee8eb023",
	"43fc49ea-f0d5-4ab3-8e5c-be5f6c0129ee",
	"ce13e7d4-7e8a-46fc-8158-73eaee0c21ab",
	"e40e9687-7ada-4e2c-9522-f6917f90f142",
	"d3cd1eb6-a840-4703-ae64-c4aaef07be21",
	"92d1838e-e08b-4189-8936-9af6b98fc63e",
}

var pipelinesCount int = len(pipelineIDs)

func main() {
	w := csv.NewWriter(os.Stdout)
	var c int
	for {
		c++
		if err := w.Write(gen(c)); err != nil {
			log.Println("error writing record to csv:", err)
		}
		if c%batchSize == 0 {
			w.Flush()
		}
		if c == datasetSize {
			break
		}
	}
	if err := w.Error(); err != nil {
		log.Fatal(err)
	}
}

func pipeline() string {
	return pipelineIDs[rand.Intn(pipelinesCount)] //nolint: gosec
}

func id() string {
	return uuid.New().String()
}

func gen(c int) []string {
	i := id()
	const doneStatus string = "2"
	return []string{
		strconv.Itoa(c),
		fmt.Sprintf("DPJ-SIP-%s.tar", i),            // name
		fmt.Sprintf("processing-workflow-%s", id()), // workflow_id
		id(),                         // run_id
		id(),                         // transfer_id
		id(),                         // aip_id
		i,                            // original_id
		pipeline(),                   // pipeline_id
		doneStatus,                   // status
		"2019-11-21 17:36:10.738582", // created_at
		"2019-11-21 17:42:10.738582", // completed_at
	}
}
