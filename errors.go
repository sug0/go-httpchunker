package httpchunker

import "errors"

var (
    ErrInvalidWorkers = errors.New("httpchunker: invalid number of workers")
)
