package httpchunker

// An httpchunker.Provider should provide an httpchunker.Downloader
// with a stream of httpchunker.Chunk structs, generating in a new
// goroutine sending to the returned channel.
type Provider interface {
    ChunkStream() (<-chan Chunk, error)
}
