package filestorage

import "io"

// FileStorage provides an abstraction for file persistence operations.
type FileStorage interface {
	Save(filename string, data io.Reader) (string, error)
	Delete(path string) error
}
