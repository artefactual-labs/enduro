package db

import rice "github.com/GeertJohan/go.rice"

//go:generate rice embed-go

func migrations() (*rice.Box, error) {
	return rice.FindBox("migrations")
}
