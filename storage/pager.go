package storage

import (
	"os"
)

type Pager struct {
	file *os.File
}

const PageSize = 4096

func Open(path string) (*Pager, error) {
	openedFile, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	return &Pager{
		file: openedFile,
	}, nil
}

// Uses a receiver, sort of like using this or self to perform the function on its OWN file
// Single return value means you're not required to parenthesize it
func (p *Pager) WritePage(k int, data []byte) error {
	offset := (int64(k) * PageSize)
	_, err := p.file.WriteAt(data, offset)
	if err != nil {
		return err
	}
	return nil
}

func (p *Pager) ReadPage(k int) ([]byte, error) {
	// bucket in memory that fits the incoming page bytes
	buffer := make([]byte, PageSize)
	offset := (int64(k) * PageSize)
	_, err := p.file.ReadAt(buffer, offset)
	if err != nil {
		return nil, err
	}
	return buffer, nil

}
