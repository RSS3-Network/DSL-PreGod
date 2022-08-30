package ipfs

import (
	"io"
	"net/http"
	"strings"
	"time"
)

func GetDirectURL(url string) string {
	if s := strings.Split(url, "/ipfs/"); len(s) == 2 {
		url = "https://ipfs.rss3.page/ipfs/" + s[1]
	}

	return strings.Replace(url, "ipfs://", "https://ipfs.rss3.page/ipfs/", 1)
}

func GetFileByURL(url string) ([]byte, error) {
	var err error
	var response *http.Response

	var i int64
	for i = 0; i < 3; i++ {
		response, err = http.Get(GetDirectURL(url))
		if err == nil {
			break
		}
		time.Sleep(time.Duration(i*3) * time.Second)
	}

	defer response.Body.Close()

	return io.ReadAll(response.Body)
}
