package bagit

import (
	go_bagit "github.com/nyudlts/go-bagit"
)

func Complete(path string) error {
	return go_bagit.ValidateBag(path, false, false)
}

func Valid(path string) error {
	return go_bagit.ValidateBag(path, false, true)
}
