package httpchunker

import (
    "os"
    "io"
    "log"
    "fmt"
    "sync"
    "time"
    "bytes"
    "errors"
    "runtime"
    "net/http"
    "path/filepath"
)

type Downloader struct {
    logger *log.Logger
    client http.Client
}

var (
    bufPool sync.Pool
    cpPool  sync.Pool
    client  http.Client
)

const maxConns = 1024

func init() {
    bufPool.New = func() interface{} {
        return new(bytes.Buffer)
    }
    cpPool.New = func() interface{} {
        return make([]byte, 4096)
    }
}

func NewDownloader() *Downloader {
    return &Downloader{
        client: http.Client{
            Transport: &http.Transport{
                MaxIdleConnsPerHost: maxConns,
                TLSHandshakeTimeout: 0,
            },
        },
    }
}

func (d *Downloader) WithLogger(logger *log.Logger) {
    d.logger = logger
}

func (d *Downloader) log(format string, args ...interface{}) {
    if d.logger != nil {
        d.logger.Printf(format, args...)
    }
}

func (d *Downloader) Download(workers int, chunks Provider, destPath, filePrefix string) []error {
    const collect = 5 * time.Second

    var errs []error

    if workers < 1 || workers > maxConns {
        return append(errs, ErrInvalidWorkers)
    }

    err := os.Mkdir(destPath, 0777)
    if err != nil && !errors.Is(err, os.ErrExist) {
        err = fmt.Errorf("httpchunker: failed to create dest dir: %w", err)
        return append(errs, err)
    }

    requests, err := chunks.ChunkStream()
    if err != nil {
        err = fmt.Errorf("httpchunker: invalid chunk provider: %w", err)
        return append(errs, err)
    }

    wg := sync.WaitGroup{}
    sem := make(chan struct{}, workers)
    dlErrors := make(chan error, workers)
    ticker := time.NewTicker(collect)
    defer ticker.Stop()

    for part := 1;; part++ {
        select {
        case <-ticker.C:
            runtime.GC()
        case err := <-dlErrors:
            errs = append(errs, err)
        case chk, ok := <-requests:
            if !ok {
                wg.Wait()
                return errs
            }
            if chk.Err != nil {
                go func() {
                    err := fmt.Errorf("httpchunker: download failed: %w", chk.Err)
                    dlErrors <- err
                }()
                continue
            }
            wg.Add(1)
            go func(part int) {
                sem <- struct{}{}
                err := d.downloadPart(part, chk.Request, destPath, filePrefix)
                <-sem
                wg.Done()
                if err != nil {
                    dlErrors <- err
                }
            }(part)
        }
    }
}

func (d *Downloader) downloadPart(part int, req *http.Request, destPath, filePrefix string) error {
    d.log("Downloading part %d\n", part)

    rsp, err := d.client.Do(req)
    if err != nil {
        return fmt.Errorf("httpchunker: download failed: %w", err)
    }
    defer rsp.Body.Close()

    body, err := readBytes(rsp.Body)
    defer returnBuf(body)

    if err != nil {
        return fmt.Errorf("httpchunker: transfer failed: %w", err)
    }

    err = writeFile(
        filepath.Join(destPath, fmt.Sprintf("%s%d", filePrefix, part)),
        body.Bytes(),
    )

    if err != nil {
        return fmt.Errorf("httpchunker: failed to save file: %w", err)
    }
    return nil
}

func writeFile(path string, body []byte) error {
    f, err := os.Create(path)
    if err != nil {
        return err
    }

    cp := cpPool.Get().([]byte)
    _, err = io.CopyBuffer(f, bytes.NewReader(body), cp)
    cpPool.Put(cp)

    f.Close()

    return err
}

func readBytes(r io.Reader) (*bytes.Buffer, error) {
    buf := bufPool.Get().(*bytes.Buffer)
    cp := cpPool.Get().([]byte)

    _, err := io.CopyBuffer(buf, r, cp)
    cpPool.Put(cp)

    return buf, err
}

func returnBuf(buf *bytes.Buffer) {
    buf.Reset()
    bufPool.Put(buf)
}
