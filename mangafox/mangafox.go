package mangafox

import (
    "io"
    "fmt"
    "time"
    "sync"
    "bytes"
    "regexp"
    "net/http"
    "crypto/tls"

    "github.com/sug0/go-httpchunker"
)

var reg = regexp.MustCompile(`<img src="([^"]+)" style="margin:0 auto;" id="image" />`)

var client = http.Client{
    Transport: &http.Transport{
        MaxIdleConnsPerHost: 1024,
        TLSHandshakeTimeout: 0 * time.Second,
        TLSClientConfig: &tls.Config{
            InsecureSkipVerify: true,
        },
    },
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
        var buf bytes.Buffer
        defer close(ch)
        defer wg.Wait()
        for page := 1; ; page++ {
            comicPageUrl := fmt.Sprintf("%s/%d.html", p.BaseURL, page)
            rsp, err := client.Get(comicPageUrl)
            if err != nil || rsp.StatusCode != 200 {
                return
            }
            wg.Add(1)
            go func() {
                defer wg.Done()
                func() { // use a new function to close body
                    defer rsp.Body.Close()
                    buf.Reset()
                    io.Copy(&buf, rsp.Body)
                }()
                imageUrl := parseUrl(buf.Bytes())
                if imageUrl == nil {
                    return
                }
                ch <- httpchunker.NewChunk("GET", string(imageUrl), nil)
            }()
        }
    }()
    return ch, nil
}

func parseUrl(p []byte) []byte {
    for {
        line := lineGet(p)
        if line == nil {
            return nil
        }
        sub := reg.FindSubmatch(line)
        if sub != nil {
            return append([]byte("https:"), sub[1]...)
        }
        p = p[len(line)+1:]
    }
}

func lineGet(p []byte) []byte {
    i := bytes.IndexByte(p, '\n')
    if i == -1 {
        return nil
    }
    return p[:i]
}
