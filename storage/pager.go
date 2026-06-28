package storage

import (
	"os"
)

type Pager struct {
	file *os.File
}

func Open(path string) (*Pager, error) {
	openedFile, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	return &Pager{
		file: openedFile}, nil
}
