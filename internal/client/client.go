package client

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

func New(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}
}

func DoRequest(client *http.Client, targetURL, userAgent string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, err
	}

	ua := userAgent
	if strings.TrimSpace(ua) == "" {
		ua = "Mozilla/5.0 (compatible; FeedFinder/1.0)"
	}

	req.Header.Set("User-Agent", ua)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml,application/rss+xml,application/atom+xml;q=0.9,*/*;q=0.8")

	return client.Do(req)
}
