package nha

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

var regexTransfer = regexp.MustCompile(
	`^` +
		`(?P<type>(?i:DPJ|EPJ|OTHER|AVL-DPJ|AVL-EPJ|AVL-OTHER))-(?i:SIP)` +
		`[-_]` +
		`(?:` +
		`(?P<journalidentifikator>[a-zA-Z0-9]{8}-[a-zA-Z0-9]{4}-[1-5][a-zA-Z0-9]{3}-[a-zA-Z0-9]{4}-[a-zA-Z0-9]{12})` +
		`|` +
		`(?P<avleveringsidentifikator>[0-9.]+)[-_](?P<date>\d{6,8})` +
		`)` +
		`(?P<ext>(?i:\.xml)?(?i:\.tar|\.zip))?` +
		`$`,
)

type TransferType int

const (
	TransferTypeUnknown TransferType = iota
	TransferTypeDPJ
	TransferTypeEPJ
	TransferTypeOther
	TransferTypeAVLXML
)

func (t TransferType) String() string {
	switch t {
	case TransferTypeDPJ:
		return "DPJ"
	case TransferTypeEPJ:
		return "EPJ"
	case TransferTypeOther:
		return "OTHER"
	case TransferTypeAVLXML:
		return "AVLXML"
	default:
		return ""
	}
}

func (t TransferType) Lower() string {
	return strings.ToLower(t.String())
}

// NameInfo captures attributes from NHA SIP obtained from its name.
type NameInfo struct {
	// journalidentifikator (UUID) or avleveringsidentifikator (?).
	Identifier string

	// Type of NHA SIP.
	Type TransferType

	// Date, only included when using avleveringsidentifikator.
	Date time.Time

	// File extension, optional.
	Extension string

	// Original value as matched.
	Value string
}

// ParseName extracts relevant information from NHA SIPs.
func ParseName(name string) (*NameInfo, error) {
	res := regexTransfer.FindStringSubmatch(name)
	if res == nil {
		return nil, fmt.Errorf("error parsing name: no matches")
	}

	info := &NameInfo{
		Extension: strings.TrimPrefix(res[5], "."),
		Value:     res[0],
	}

	t := strings.ToUpper(res[1])
	switch {
	case t == TransferTypeDPJ.String():
		info.Type = TransferTypeDPJ
	case t == TransferTypeEPJ.String():
		info.Type = TransferTypeEPJ
	case t == TransferTypeOther.String():
		info.Type = TransferTypeOther
	case t == TransferTypeAVLXML.String() || strings.HasPrefix(t, "AVL-"):
		info.Type = TransferTypeAVLXML
	}

	// journalidentifikator
	if res[2] != "" {
		_, err := uuid.Parse(res[2])
		if err != nil {
			return nil, fmt.Errorf("error parsing uuid: %s", err)
		}
		info.Identifier = res[2]
	}

	// avleveringsidentifikator + date
	if res[3] != "" {
		info.Identifier = res[3]
	}
	if res[4] != "" {
		const format = "20060102"
		var err error
		info.Date, err = time.Parse(format, res[4])
		if err != nil {
			return nil, fmt.Errorf("error parsing date: %s", err)
		}
	}

	return info, nil
}

func (i NameInfo) String() string {
	return i.Value
}
