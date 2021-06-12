package db

import rice "github.com/GeertJohan/go.rice"

//go:generate make -C ../../ gen-migrations

func migrations() (*rice.Box, error) {
	return rice.FindBox("migrations")
}
