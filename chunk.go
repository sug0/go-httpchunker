package httpchunker

import (
    "io"
    "net/http"
)

// An httpchunker.Chunk represents the corresponding http.Request
// associated with a particular file chunk to be downloaded.
// The error returned by http.NewRequest() should be included
// in the Err field of this struct.
type Chunk struct {
    *http.Request
    Err error
}

// Wrapper method for http.NewRequest() that returns
// an httpchunker.Chunk.
func NewChunk(method, url string, body io.Reader) Chunk {
    req, err := http.NewRequest(method, url, body)
    return Chunk{req, err}
}

// Used to setup the httpchunker.Chunk returned by
// httpchunker.NewChunk(), when the error returned by
// http.NewRequest() is nil.
func (chk Chunk) Setup(setup func(req *http.Request)) Chunk {
    if chk.Err == nil {
        setup(chk.Request)
    }
    return chk
}
