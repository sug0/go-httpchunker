package httpchunker

import "errors"

// All the errors returned by this package.
var (
    // An inappropriate number of workers was used. The number of workers
    // should be in the range [1 .. httpchunker.MaxConns].
    ErrInvalidWorkers = errors.New("httpchunker: invalid number of workers")

    // The returned HTTP status is not 200.
    ErrHTTPStatus = errors.New("httpchunker: invalid HTTP status")

    // Empty response body.
    ErrEmptyBody = errors.New("httpchunker: body is empty")
)
