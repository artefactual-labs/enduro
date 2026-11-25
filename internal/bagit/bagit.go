package bagit

import (
	go_bagit "github.com/nyudlts/go-bagit"
)

func Complete(path string) error {
	bag, err := go_bagit.GetExistingBag(path)
	if err != nil {
		return err
	}

	return bag.ValidateBag(false, false)
}

func Valid(path string) error {
	bag, err := go_bagit.GetExistingBag(path)
	if err != nil {
		return err
	}

	return bag.ValidateBag(false, true)
}

// ConvertedLayout records which top-level transfer entries came from data/.
// Receipt activities use it to distinguish submitted content from bag tags.
type ConvertedLayout struct {
	PayloadRoots []string
}
