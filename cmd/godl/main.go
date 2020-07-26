package main

import (
    "os"
    "log"
    "fmt"

    "github.com/sug0/go-httpchunker"
    "github.com/sug0/go-httpchunker/byterange"
)

func main() {
    p := byterange.ChunkProvider{
        URL: os.Args[1],
        ChunkSize: 2*1024*1024,
    }

    d := httpchunker.NewDownloader()
    d.WithLogger(log.New(os.Stderr, os.Args[0]+": ", log.LstdFlags))

    errs := d.Download(24, p, httpchunker.Filename{"part_", "out"})
    if errs != nil {
        panic(joinErrors(errs))
    }
}

func joinErrors(errs []error) error {
    err := errs[0]
    for i := 1; i < len(errs); i++ {
        err = fmt.Errorf("%s\n%s", err, errs[i])
    }
    return fmt.Errorf("%s\n", err)
}
