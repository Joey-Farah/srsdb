package storage

import (
	"bytes"
	"path/filepath"
	"testing"
)

func TestReadWriteRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	pager, err := Open(path)
	if err != nil {
		t.Fatalf("Open() failed: %v", err)
	}
	want := make([]byte, PageSize)
	want[0] = 42
	err = pager.WritePage(3, want)
	if err != nil {
		t.Fatalf("WritePage() failed: %v", err)
	}

	got, err := pager.ReadPage(3)
	if err != nil {
		t.Fatalf("ReadPage() failed: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("Round-trip mismatch: got %v, want %v", got[:4], want[:4])
	}
}
