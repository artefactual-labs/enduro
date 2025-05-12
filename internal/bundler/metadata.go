package bundler

import (
	"encoding/csv"
	"fmt"
	"sort"

	"github.com/spf13/afero"
)

// MetadataSet holds the metadata entries of the transfer.
type MetadataSet struct {
	entries map[string][][2]string
	fs      afero.Fs
}

// NewMetadataSet returns a new MetadataSet.
func NewMetadataSet(fs afero.Fs) *MetadataSet {
	return &MetadataSet{
		entries: make(map[string][][2]string),
		fs:      fs,
	}
}

// Entries returns all entries that were created.
func (m *MetadataSet) Entries() map[string][][2]string {
	// MetadataSet doesn't have a mutex yet but once it's used, it should be
	// locked right here.

	// Make a copy so the returned value won't race with future log requests.
	entries := make(map[string][][2]string)
	for k, v := range m.entries {
		entries[k] = v
	}
	return entries
}

func (m *MetadataSet) Add(name, field, value string) {
	m.entries[name] = append(m.entries[name], [2]string{field, value})
}

func (m *MetadataSet) Write() error {
	const (
		path = "/metadata/metadata.csv"
		sep  = ','
	)
	if len(m.entries) == 0 {
		return nil
	}
	f, err := m.fs.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	writer := csv.NewWriter(f)
	writer.Comma = sep
	writer.UseCRLF = false
	defer writer.Flush()

	// Build a list of fields with max. total of occurrences found
	occurrences := map[string]int{}
	for _, entry := range m.entries {
		for _, pair := range entry { // Pair ("dc.title", "title 1")
			var o int
			for _, p := range entry {
				if pair[0] == p[0] {
					o++
				}
			}
			if c, ok := occurrences[pair[0]]; !ok || (ok && o > c) {
				occurrences[pair[0]] = o
			}
		}
	}

	// Build a list of fields
	fields := []string{}
	for field, o := range occurrences {
		for range o {
			fields = append(fields, field)
		}
	}
	sort.Strings(fields)

	// Write header row in CSV.
	_ = writer.Write(append([]string{"filename"}, fields...))

	// Create an slice of filenames sorted alphabetically. We're going to use it
	// so we can iterate over the files in order to generate CSV output in a
	// predicable way.
	names := make([]string, len(m.entries))
	for filename := range m.entries {
		names = append(names, filename)
	}
	sort.Strings(names)

	for _, filename := range names {
		entry, ok := m.entries[filename]
		if !ok {
			continue
		}
		var (
			values  = []string{filename}
			cursors = make(map[string]int)
		)
		// For each known field we either populate a value or an empty string.
		for _, field := range fields {
			var (
				value  string
				subset = entry
				offset = 0
			)
			if pos, ok := cursors[field]; ok {
				pos++
				subset = entry[pos:] // Continue at the next value
				offset = pos
			}
			for index, pair := range subset {
				if pair[0] == field {
					value = pair[1]                 // We have a match
					cursors[field] = index + offset // Memorize position
					break
				}
			}
			values = append(values, value)
		}
		if len(values) > 1 {
			_ = writer.Write(values)
		}
	}

	return nil
}

// ChecksumSet holds the checksums of the files for a sum algorithm.
type ChecksumSet struct {
	sumType string
	values  map[string]string
	fs      afero.Fs
}

func NewChecksumSet(sumType string, fs afero.Fs) *ChecksumSet {
	return &ChecksumSet{
		sumType: sumType,
		values:  make(map[string]string),
		fs:      fs,
	}
}

func (c *ChecksumSet) Add(name, sum string) {
	c.values[name] = sum
}

func (c *ChecksumSet) Write() error {
	const (
		path = "/metadata/checksum.%s"
		sep  = ' '
	)
	if len(c.values) == 0 {
		return nil
	}
	f, err := c.fs.Create(fmt.Sprintf(path, c.sumType))
	if err != nil {
		return err
	}
	defer f.Close()
	writer := csv.NewWriter(f)
	writer.Comma = sep
	writer.UseCRLF = false
	defer writer.Flush()

	for name, sum := range c.values {
		if err := writer.Write([]string{sum, name}); err != nil {
			return err
		}
	}

	return nil
}
