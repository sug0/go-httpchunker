package httpchunker

import "net/http"

type Chunk struct {
    *http.Request
    Err error
}

type Provider interface {
    ChunkStream() (<-chan Chunk, error)
}
