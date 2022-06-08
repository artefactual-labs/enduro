package sdps

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"strings"
)

type EASStructureType int

const (
	EASStructureTypeA EASStructureType = iota
	EASStructureTypeB
	EASStructureTypeC
)

func (t EASStructureType) String() string {
	switch t {
	case EASStructureTypeA:
		return "A"
	case EASStructureTypeB:
		return "B"
	case EASStructureTypeC:
		return "C"
	}
	return ""
}

func (t EASStructureType) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.String())
}

func (t *EASStructureType) UnmarshalCSV(b []byte) (err error) {
	s := string(b)
	switch s {
	case "A":
		*t = EASStructureTypeA
	case "B":
		*t = EASStructureTypeB
	case "C":
		*t = EASStructureTypeC
	default:
		return fmt.Errorf("unknown struct type: %q", s)
	}
	return nil
}

// SecurityGrading is the security grading of a batch.
type SecurityGrading int

const (
	SecurityGradingNonSecret SecurityGrading = iota
	SecurityGradingUnclassified
	SecurityGradingRestricted
	SecurityGradingConfidential
)

func (sg SecurityGrading) String() string {
	switch sg {
	case SecurityGradingNonSecret:
		return "Non-Secret"
	case SecurityGradingUnclassified:
		return "Unclassified"
	case SecurityGradingRestricted:
		return "Restricted"
	case SecurityGradingConfidential:
		return "Confidential"
	}
	return ""
}

func (sg SecurityGrading) MarshalJSON() ([]byte, error) {
	return json.Marshal(sg.String())
}

func (sg *SecurityGrading) UnmarshalCSV(b []byte) (err error) {
	s := string(b)
	switch s {
	case "Non-Secret":
		*sg = SecurityGradingNonSecret
	case "Unclassified":
		*sg = SecurityGradingUnclassified
	case "Restricted":
		*sg = SecurityGradingRestricted
	case "Confidential":
		*sg = SecurityGradingConfidential
	default:
		return fmt.Errorf("unknown security grading: %q", s)
	}
	return nil
}

type Item interface {
	Validate() error
	fmt.Stringer
}

var (
	ErrValidateEmptyIdentifier             = errors.New("identifier is empty")
	ErrValidateIncompatibleSecurityGrading = errors.New("incompatible security grading")
)

// File implements Item.
type File struct {
	batch *Batch

	BatchNumber     string `json:"batchNumber"`
	FolderRef       string `json:"folderRef"`
	RecordID        string `json:"recordID"`
	Representation  string `json:"representation"`
	FileSysID       string `json:"fileSysID"`
	ParentFileSysID string `json:"parentFileSysID"`
	FileOrder       string `json:"fileOrder"`
	ChecksumMD5     string `json:"checksumMD5"`
	ChecksumSHA256  string `json:"checksumSHA256"`
	FilePathName    string `json:"filePathName"`
	Note1           string `json:"note1"`
	Note2           string `json:"note2"`
}

var _ Item = (*File)(nil)

func (i File) Validate() error {
	if i.String() == "" {
		return ErrValidateEmptyIdentifier
	}

	return nil
}

func (i File) String() string {
	return i.FilePathName
}

// Record implements Item.
type Record struct {
	batch *Batch

	BatchNumber           string          `json:"batchNumber"`
	FolderRef             string          `json:"folderRef"`
	RecordID              string          `json:"recordID"`
	RecordTitle           string          `json:"recordTitle"`
	RecordDate            string          `json:"recordDate"`
	RecordSecurityGrading SecurityGrading `json:"recordSecurityGrading"`
	AuthorName            string          `json:"authorName"`
	AuthorDesignation     string          `json:"authorDesignation"`
	FiledBy               string          `json:"eFiledBy"`
	FiledDesignation      string          `json:"filedDesignation"`
	FiledDateTime         string          `json:"filedDateTime"`
	KeywordsRemarks       string          `json:"keywordsRemarks"`
	Note1                 string          `json:"note1"`
	Note2                 string          `json:"note2"`
}

var _ Item = (*Record)(nil)

func (i Record) Validate() error {
	if i.RecordSecurityGrading > i.batch.BatchSecurityGrading {
		return fmt.Errorf("%w - record=%s batch=%s", ErrValidateIncompatibleSecurityGrading, i.RecordSecurityGrading, i.batch.BatchSecurityGrading)
	}

	if i.String() == "" {
		return ErrValidateEmptyIdentifier
	}

	return nil
}

func (i Record) String() string {
	return i.RecordID
}

// Folder implements Item.
type Folder struct {
	batch *Batch

	BatchNumber           string          `json:"batchNumber"`
	AuthorityNumber       string          `json:"authorityNumber"`
	ReviewDate            string          `json:"reviewDate"`
	RecordSeriesNumber    string          `json:"recordSeriesNumber"`
	RecordSeriesTitle     string          `json:"recordSeriesTitle"`
	AppraisedSystemID     string          `json:"appraisedSystemID"`
	FolderRef             string          `json:"folderRef"`
	FolderTitle           string          `json:"folderTitle"`
	FolderSecurityGrading SecurityGrading `json:"folderSecurityGrading"`
	FolderTransferFormat  string          `json:"folderTransferFormat"`
	FolderFromDate        string          `json:"folderFromDate"`
	FolderToDate          string          `json:"folderToDate"`
	KeywordsRemarks       string          `json:"keywordsRemarks"`
	Note1                 string          `json:"note1"`
	Note2                 string          `json:"note2"`
}

var _ Item = (*Folder)(nil)

func (i Folder) Validate() error {
	if i.FolderSecurityGrading > i.batch.BatchSecurityGrading {
		return fmt.Errorf("%w - folder=%s batch=%s", ErrValidateIncompatibleSecurityGrading, i.FolderSecurityGrading, i.batch.BatchSecurityGrading)
	}

	if i.String() == "" {
		return ErrValidateEmptyIdentifier
	}

	return nil
}

func (i Folder) String() string {
	return i.FolderRef
}

// Batch is a compound of contents.
type Batch struct {
	path  string
	items []Item

	// Batch is a non-thread-safe iterator, idx is the position of the iterator.
	idx int

	TransferringAgencyID         string           `json:"transferringAgencyID"`
	TransferringAgency           string           `json:"transferringAgency"`
	TransferringAgencyDepartment string           `json:"transferringAgencyDepartment"`
	CreatingAgency               string           `json:"creatingAgency"`
	BatchNumber                  string           `json:"batchNumber"`
	ArchivingDateTime            string           `json:"archivingDateTime"`
	AgencySystem                 string           `json:"agencySystem"`
	TransferDate                 string           `json:"transferDate"`
	RecordType                   string           `json:"recordType"`
	EASStructureType             EASStructureType `json:"easStructureType"`
	BatchSecurityGrading         SecurityGrading  `json:"batchSecurityGrading"`
	SubmittedBy                  string           `json:"submittedBy"`
	EmailSubmitter               string           `json:"emailSubmitter"`
	Designation                  string           `json:"designation"`
	Remarks                      string           `json:"remarks"`
}

// OpenBatch returns a Batch given its path.
func OpenBatch(path string) (*Batch, error) {
	b := &Batch{
		path:  path,
		items: []Item{},
		idx:   -1,
	}

	if err := b.load(path); err != nil {
		return nil, fmt.Errorf("error loading batch: %v", err)
	}

	return b, nil
}

func (b *Batch) load(path string) error {
	res, err := ioutil.ReadDir(path)
	if err != nil {
		return fmt.Errorf("cannot read dir: %v (%s)", err, path)
	}

	// We have to find the manifest.
	var manifest string
	for _, item := range res {
		if item.IsDir() {
			continue
		}
		name := item.Name()
		switch strings.ToLower(name) {
		case "folder.csv", "record.csv", "file.csv", "batch.csv":
			manifest = filepath.Join(path, name)
		default:
			continue
		}
	}
	if manifest == "" {
		return fmt.Errorf("manifest not found in %s", path)
	}

	dec, closer, err := csvdecoder(manifest)
	if err != nil {
		return fmt.Errorf("cannot create manifest decoder: %v (%s)", err, path)
	}
	defer func() { _ = closer() }()

	t := strings.ToLower(filepath.Base(manifest))

	if t == "batch.csv" {
		err := dec.Decode(b)
		if err != nil {
			return fmt.Errorf("error decoding manifest item: %v", err)
		}
		return b.load(filepath.Join(path, b.BatchNumber))
	}

	for {
		var item Item
		switch t {
		case "folder.csv":
			item = &Folder{batch: b}
		case "record.csv":
			item = &Record{batch: b}
		case "file.csv":
			item = &File{batch: b}
		}

		err := dec.Decode(item)
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("error decoding manifest item: %v", err)
		}

		if err := item.Validate(); err != nil {
			return fmt.Errorf("invalid entry: %v (%s)", err, manifest)
		}

		// Collect the item.
		b.items = append(b.items, item)

		// Keep going down the hierarchy.
		if t != "file.csv" {
			p := filepath.Join(path, item.String())
			if err = b.load(p); err != nil {
				return err
			}
		}
	}

	return nil
}

type BatchIterator interface {
	// Next advances the iterator and returns whether
	// the next call to the item method will return a
	// non-nil item.
	//
	// Next should be called prior to any call to the
	// iterator's item retrieval method after the
	// iterator has been obtained or reset.
	//
	// The order of iteration is implementation
	// dependent.
	Next() bool

	// Reset returns the iterator to its start position.
	Reset()

	// Item returns the current item of the iterator.
	Item() Item
}

// Item returns the current item of the iterator.
func (b *Batch) Item() Item {
	if b.idx >= len(b.items) || b.idx < 0 {
		return nil
	}
	return b.items[b.idx]
}

// Next moves the iterator forward one position.
func (b *Batch) Next() bool {
	if uint(b.idx)+1 < uint(len(b.items)) {
		b.idx++
		return true
	}
	b.idx = len(b.items)
	return false
}

// Reset returns the iterator to its initial state.
func (b *Batch) Reset() {
	b.idx = -1
}
