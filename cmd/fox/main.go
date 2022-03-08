package main

import (
    "os"
    "log"
    "fmt"

    "github.com/sug0/go-httpchunker"
    "github.com/sug0/go-httpchunker/mangafox"
)

func main() {
    if len(os.Args) < 2 {
        fmt.Println("Error: Provide a base url, e.g. http://m.fanfox.net/manga/immortal_regis/v01/c001/")
        os.Exit(1)
    }

    p := mangafox.ChunkProvider{BaseURL: os.Args[1]}

    d := httpchunker.NewDownloader()
    d.WithLogger(log.New(os.Stderr, os.Args[0]+": ", log.LstdFlags))

    errs := d.Download(24, p, httpchunker.Filename{
        Prefix: "page",
        Suffix: ".jpeg",
        Dest: "out",
    })
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
