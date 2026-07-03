package storage

import (
	"os"
)

type Pager struct {
	file *os.File
}

const PageSize = 4096

func Open(path string) (*Pager, error) {
	openedFile, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return &Pager{
		file: openedFile,
	}, nil
}

func (p *Pager) WritePage(k int, data []byte) error {
	offset := (int64(k) * PageSize)
	_, err := p.file.WriteAt(data, offset)
	if err != nil {
		return err
	}
	return nil
}
