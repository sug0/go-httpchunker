package mangafox

import (
    "io"
    "fmt"
    "time"
    "sync"
    "bytes"
    "regexp"
    "strconv"
    "net/http"
    "crypto/tls"

    "github.com/sug0/go-httpchunker"
)

var urlReg = regexp.MustCompile(`<img src="([^"]+)" style="margin:0 auto;" id="image" />`)
var limReg = regexp.MustCompile(`<option value="[^"]+">(\w+)</option>`)

var client = http.Client{
    Transport: &http.Transport{
        MaxIdleConnsPerHost: 1024,
        TLSHandshakeTimeout: 0 * time.Second,
        TLSClientConfig: &tls.Config{
            InsecureSkipVerify: true,
        },
    },
}

var bufPool = sync.Pool{
    New: func() interface{} { return new(bytes.Buffer) },
}

type ChunkProvider struct {
    // The base URL without the trailing page number,
    // e.g. http://m.fanfox.net/manga/immortal_regis/v01/c001/
    BaseURL string
}

func (p ChunkProvider) ChunkStream() (<-chan httpchunker.Chunk, error) {
    if p.BaseURL[len(p.BaseURL)-1] == '/' {
        // truncate slash
        p.BaseURL = p.BaseURL[:len(p.BaseURL)-1]
    }
    ch := make(chan httpchunker.Chunk, 24)
    go func() {
        var wg sync.WaitGroup

        defer close(ch)
        defer wg.Wait()

        buf := p.fetchPage(1)
        if buf == nil {
            return
        }
        limit, ok := parseLimit(buf)
        if !ok {
            return
        }
        wg.Add(limit)

        go func(buf *bytes.Buffer) {
            defer wg.Done()
            imageUrl, ok := parsePageUrl(buf)
            if ok {
                ch <- httpchunker.NewChunk("GET", imageUrl, nil)
            }
        }(buf)

        for page := 2; page <= limit; page++ {
            go func(page int) {
                defer wg.Done()
                buf := p.fetchPage(page)
                if buf == nil {
                    return
                }
                imageUrl, ok := parsePageUrl(buf)
                if !ok {
                    return
                }
                ch <- httpchunker.NewChunk("GET", imageUrl, nil)
            }(page)
        }
    }()
    return ch, nil
}

func (p ChunkProvider) fetchPage(page int) *bytes.Buffer {
    buf := getBuffer()
    comicPageUrl := fmt.Sprintf("%s/%d.html", p.BaseURL, page)
    rsp, err := client.Get(comicPageUrl)
    if err != nil || rsp.StatusCode != 200 {
        return nil
    }
    defer rsp.Body.Close()
    _, err = io.Copy(buf, rsp.Body)
    if err != nil {
        return nil
    }
    return buf
}

func parsePageUrl(buf *bytes.Buffer) (string, bool) {
    defer returnBuffer(buf)
    imageUrl := parseUrl(buf.Bytes())
    if imageUrl == nil {
        return "", false
    }
    return string(imageUrl), true
}

func getBuffer() *bytes.Buffer {
    return bufPool.Get().(*bytes.Buffer)
}

func returnBuffer(buf *bytes.Buffer) {
    buf.Reset()
    bufPool.Put(buf)
}

func parseUrl(p []byte) []byte {
    for {
        line := lineGet(p)
        if line == nil {
            return nil
        }
        sub := urlReg.FindSubmatch(line)
        if sub != nil {
            return append([]byte("https:"), sub[1]...)
        }
        p = p[len(line)+1:]
    }
}

func parseLimit(buf *bytes.Buffer) (int, bool) {
    matches := limReg.FindAllSubmatch(buf.Bytes(), -1)
    if matches == nil {
        return 0, false
    }
    l, err := strconv.ParseUint(string(matches[len(matches)-1][1]), 10, 32)
    if err != nil {
        return 0, false
    }
    return int(l), true
}

func lineGet(p []byte) []byte {
    i := bytes.IndexByte(p, '\n')
    if i == -1 {
        return nil
    }
    return p[:i]
}
