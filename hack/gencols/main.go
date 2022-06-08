package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
)

const (
	// How many records we want to generate.
	datasetSize = 2500000

	// Size of each batch that we write to the CSV.
	batchSize = 100
)

func main() {
	rand.Seed(time.Now().UnixNano())

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
		id(),                         // aip_id
		doneStatus,                   // status
		"2019-11-21 17:36:10.738582", // created_at
		"2019-11-21 17:36:10.738582", // started_at
		"2019-11-21 17:42:10.738582", // completed_at
	}
}
