package httpchunker

import (
    "os"
    "io"
    "log"
    "fmt"
    "sync"
    "bytes"
    "errors"
    "net/http"
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

// Max number of connections kept alive by an httpchunker.Downloader.
const MaxConns = 1024

func init() {
    bufPool.New = func() interface{} {
        return new(bytes.Buffer)
    }
    cpPool.New = func() interface{} {
        return make([]byte, 4096)
    }
}

// Creates a new httpchunker.Downloader and returns it.
func NewDownloader() *Downloader {
    return &Downloader{
        client: http.Client{
            Transport: &http.Transport{
                MaxIdleConnsPerHost: MaxConns,
                TLSHandshakeTimeout: 0,
            },
        },
    }
}

// Sets the logger used by httpchunker.Downloader. It can be nil,
// to disable all logging.
func (d *Downloader) WithLogger(logger *log.Logger) {
    d.logger = logger
}

func (d *Downloader) log(format string, args ...interface{}) {
    if d.logger != nil {
        d.logger.Printf(format, args...)
    }
}

// Perform the actual download.
func (d *Downloader) Download(workers int, chunks Provider, pn Partnamer) []error {
    var errs []error

    if workers < 1 || workers > MaxConns {
        return append(errs, ErrInvalidWorkers)
    }

    err := os.Mkdir(pn.Destpath(), 0777)
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

    for part := 1;; part++ {
        select {
        case err := <-dlErrors:
            errs = append(errs, err)
        case chk, ok := <-requests:
            if !ok {
                // drain errors channel
                close(dlErrors)
                for err := range dlErrors {
                    errs = append(errs, err)
                }
                wg.Wait()
                return errs
            }
            if chk.Err != nil {
                wg.Add(1)
                go func(err error) {
                    err = fmt.Errorf("httpchunker: download failed: %w", err)
                    dlErrors <- err
                    wg.Done()
                }(chk.Err)
                continue
            }
            wg.Add(1)
            go func(part int) {
                sem <- struct{}{}
                err := d.downloadPart(part, chk.Request, pn)
                if err != nil {
                    dlErrors <- err
                }
                <-sem
                wg.Done()
            }(part)
        }
    }
}

func (d *Downloader) downloadPart(part int, req *http.Request, pn Partnamer) error {
    d.log("Downloading part %d\n", part)

    rsp, err := d.client.Do(req)
    switch {
    case err != nil:
        return fmt.Errorf("httpchunker: download failed: %w", err)
    case (rsp.StatusCode/200) != 1:
        rsp.Body.Close()
        return fmt.Errorf("httpchunker: %s: %w", rsp.Status, ErrHTTPStatus)
    }
    defer rsp.Body.Close()

    body, err := readBytes(rsp.Body)
    defer returnBuf(body)

    bodyBytes := body.Bytes()

    switch {
    case err != nil:
        return fmt.Errorf("httpchunker: transfer failed: %w", err)
    case len(bodyBytes) == 0:
        return fmt.Errorf("httpchunker: %s: %w", pn.Filename(part), ErrEmptyBody)
    }

    err = writeFile(PathOf(part, pn), bodyBytes)

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
