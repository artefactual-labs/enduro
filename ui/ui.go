package ui

import (
	"net/http"

	rice "github.com/GeertJohan/go.rice"
)

//go:generate make -C ../ gen-ui

// Handler creates a HTTP handler for the web content.
func Handler() (http.Handler, error) {
	box, err := rice.FindBox("dist")
	if err != nil {
		return nil, err
	}

	return http.FileServer(box.HTTPBox()), nil
}
