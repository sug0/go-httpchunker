package byterange

import (
    "fmt"
    "strconv"
    "net/http"

    "github.com/sug0/go-httpchunker"
)

type ChunkProvider struct {
    ChunkSize uint64
    URL       string
}

func (p ChunkProvider) ChunkStream() (<-chan httpchunker.Chunk, error) {
    rsp, err := http.Head(p.URL)
    if err != nil {
        return nil, fmt.Errorf("byterange: HEAD failed: %w", err)
    }

    header := rsp.Header
    possibilities := header["Accept-Ranges"]

    for i := 0; i < len(possibilities); i++ {
        if possibilities[i] == "bytes" {
            goto acceptable
        }
    }

    return nil, fmt.Errorf("byterange: no byte range support")

acceptable:
    max, err := strconv.ParseUint(header.Get("Content-Length"), 10, 64)
    if err != nil {
        return nil, fmt.Errorf("byterange: error parsing length: %w", err)
    }

    // determine best chunk size automatically
    if p.ChunkSize == 0 {
        // for now pick 1KiB chunks
        p.ChunkSize = 1024
    }

    ch := make(chan httpchunker.Chunk, 4)

    go func() {
        var exit bool
        var off uint64
        step := p.ChunkSize
        url := p.URL
        for !exit {
            if off + step >= max {
                step = max - off
                exit = true
            }
            ch <- httpchunker.NewChunk("GET", url, nil).Setup(func(req *http.Request) {
                req.Header.Set("Connection", "Keep-Alive")
                req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", off, off+step-1))
            })
            off += step
        }
        close(ch)
    }()

    return ch, nil
}
