package ui

import (
	"net/http"

	rice "github.com/GeertJohan/go.rice"
)

//go:generate rice embed-go

// Handler creates a HTTP handler for the web content.
func Handler() (http.Handler, error) {
	box, err := rice.FindBox("dist")
	if err != nil {
		return nil, err
	}

	return http.FileServer(box.HTTPBox()), nil
}
