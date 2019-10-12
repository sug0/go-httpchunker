package httpchunker

import "net/http"

// An httpchunker.Chunk represents the corresponding http.Request
// associated with a particular file chunk to be downloaded.
// The error returned by http.NewRequest() should be included
// in the Err field of this struct.
type Chunk struct {
    *http.Request
    Err error
}

// An httpchunker.Provider should provide an httpchunker.Downloader
// with a stream of httpchunker.Chunk structs, generating in a new
// goroutine sending to the returned channel.
type Provider interface {
    ChunkStream() (<-chan Chunk, error)
}
