package interfaces

import (
	"io"
	"time"
)

// FileOpener abstracts access to files stored on the operating system so the
// internal streaming server can serve them without depending on the os package
// directly.
type FileOpener interface {
	Open(path string) (rc io.ReadSeekCloser, size int64, modTime time.Time, err error)
}
