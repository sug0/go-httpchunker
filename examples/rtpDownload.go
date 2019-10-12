package main

import (
    "os"
    "log"
    "fmt"
    "flag"
    "bytes"
    "net/http"
    "io/ioutil"

    "github.com/sug0/go-httpchunker"
)

type reqProvider struct{}

var (
    vDestDir  string
    vPrefix   string
    vPlaylist string
    vWorkers  int
)

func init() {
    flag.StringVar(&vDestDir, "o", "out", "Dir to save the video chunks to.")
    flag.StringVar(&vPrefix, "p", "", "The URL prefix for the video chunks.")
    flag.StringVar(&vPlaylist, "i", "", "The URL of the video chunks playlist.")
    flag.IntVar(&vWorkers, "workers", 8, "The number of goroutine workers.")
    flag.Parse()

    if vPlaylist == "" || vPrefix == "" {
        panic("Wrong params. Enter `-help` flag for usage.")
    }
}

func main() {
    d := httpchunker.NewDownloader()
    d.WithLogger(log.New(os.Stderr, "rtpDownloader: ", log.LstdFlags))

    errs := d.Download(vWorkers, reqProvider{}, vDestDir, "part_")
    if errs != nil {
        panic(joinErrors(errs))
    }
}

func joinErrors(errs []error) error {
    err := errs[0]
    for i := 1; i < len(errs); i++ {
        err = fmt.Errorf("«%w», followed by «%w»", errs[i], err)
    }
    return err
}

func (reqProvider) ChunkStream() (<-chan httpchunker.Chunk, error) {
    rsp, err := http.Get(vPlaylist)
    if err != nil {
        return nil, err
    }

    p, err := ioutil.ReadAll(rsp.Body)
    if err != nil {
        return nil, err
    }
    rsp.Body.Close()

    ch := make(chan httpchunker.Chunk, 4)

    go func() {
        for {
            chunk := lineGet(p)
            if chunk == nil {
                close(ch)
                return
            }
            if chunk[0] != '#' {
                url := fmt.Sprintf("%s/%s", vPrefix, chunk)
                ch <- httpchunker.NewChunk("GET", url, nil)
            }
            p = p[len(chunk)+1:]
        }
    }()

    return ch, nil
}

func lineGet(p []byte) []byte {
    i := bytes.IndexByte(p, '\n')
    if i == -1 {
        return nil
    }
    return p[:i]
}
