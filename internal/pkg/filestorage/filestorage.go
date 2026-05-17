//go:generate go run go.uber.org/mock/mockgen -destination=../../mocks/filestorage_mock.go -package=mocks github.com/go-park-mail-ru/2026_1_KISS/internal/pkg/filestorage FileStorage
package filestorage

import "io"

// FileStorage provides an abstraction for file persistence operations.
type FileStorage interface {
	Save(filename string, data io.Reader) (string, error)
	Delete(path string) error
	Open(storageKey string) (io.ReadCloser, error)
}
