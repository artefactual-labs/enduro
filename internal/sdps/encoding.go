package sdps

import (
	"encoding/csv"
	"os"

	"github.com/jszwec/csvutil"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

// They seem to be using Windows-1252!
var decoder = charmap.Windows1252

// csvdecoder returns a CSV decoder suited for SDPS batches.
func csvdecoder(path string) (*csvutil.Decoder, func() error, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}

	closer := func() error { return f.Close() }

	reader := csv.NewReader(transform.NewReader(f, decoder.NewDecoder()))

	decoder, err := csvutil.NewDecoder(reader)
	if err != nil {
		return nil, nil, err
	}

	return decoder, closer, nil
}
