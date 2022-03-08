package httpchunker

import (
    "fmt"
    "path/filepath"
)

// A simple Partnamer implementation.
type Filename struct {
    Prefix string
    Suffix string
    Dest   string
}

// An interface used to fetch the name a particular
// file part should be given.
type Partnamer interface {
    Destpath() string
    Filename(part int) string
}

// Utility function that returns the full path of
// a file given its httpchunker.FilePath.
func PathOf(part int, p Partnamer) string {
    return filepath.Join(p.Destpath(), p.Filename(part))
}

// Implements Partnamer.
func (fn Filename) Destpath() string {
    return fn.Dest
}

// Implements Partnamer.
func (fn Filename) Filename(part int) string {
    return fmt.Sprintf("%s%08d%s", fn.Prefix, part, fn.Suffix)
}
