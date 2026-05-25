package discovery

import (
	"fmt"
	"io"
	"mime"
	"net/url"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/ramadhantriyant/feedfinder/internal/client"
	"github.com/ramadhantriyant/feedfinder/internal/feed"
	"github.com/ramadhantriyant/feedfinder/internal/model"
)

const debugColor = "\033[90m"
const colorReset = "\033[0m"

func Discover(rawURL, userAgent string, timeout time.Duration, verbose bool) ([]model.FeedResult, error) {
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "https://" + rawURL
	}

	c := client.New(timeout)
	var results []model.FeedResult

	logf := func(format string, a ...any) {
		if verbose {
			fmt.Fprintf(os.Stderr, debugColor+"  [dbg] "+format+colorReset+"\n", a...)
		}
	}

	// Stage 1: Fetch base URL
	logf("Fetching %s", rawURL)
	resp, err := client.DoRequest(c, rawURL, userAgent)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("server returned HTTP %s", resp.Status)
	}

	effectiveURL := resp.Request.URL.String()
	base, err := url.Parse(effectiveURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	// Stage 2: Content-Type sniff — is the URL itself a feed?
	ct := resp.Header.Get("Content-Type")
	logf("Content-Type: %s", ct)
	if mediaType, _, err := mime.ParseMediaType(ct); err == nil && feed.IsFeedMIME(mediaType) {
		logf("URL is itself a feed (Content-Type: %s)", mediaType)
		info, err := feed.ParseInfo(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to parse feed: %w", err)
		}

		results = append(results, model.FeedResult{
			URL:    effectiveURL,
			Source: "Content-Type header (" + mediaType + ")",
			Title:  info.Title,
		})
		return results, nil
	}

	// Stage 3: HTML <link rel="alternate"> / <link rel="self"> discovery
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	body := string(bodyBytes)

	logf("Scanning HTML for <link> tags")
	links := extractLinkTags(body)
	logf("Found %d <link> tags", len(links))

	for _, link := range links {
		isAlternate := hasToken(link.Rel, "alternate")
		isSelf := hasToken(link.Rel, "self")
		if !isAlternate && !isSelf {
			continue
		}
		ltype := strings.ToLower(strings.TrimSpace(link.Type))
		if !slices.Contains(feed.MIMETypes, ltype) || link.Href == "" {
			continue
		}

		resolved := resolveURL(base, link.Href)
		if resolved == "" {
			continue
		}
		logf("Found via <link rel=%q type=%q href=%q>", link.Rel, link.Type, link.Href)

		info, ok := isValidFeed(c, resolved, userAgent)
		if !ok {
			// Still include — the <link> tag is authoritative even if validation
			// fails (server may rate-limit a second request).
			info = model.FeedInfo{}
		}

		source := `HTML <link rel="alternate">`
		if isSelf {
			source = `HTML <link rel="self">`
		}
		results = append(results, model.FeedResult{
			URL:    resolved,
			Source: source + " (type=" + link.Type + ")",
			Title:  info.Title,
		})
	}

	if len(results) > 0 {
		return results, nil
	}

	// Stage 4: Common path probing
	commonPaths := []string{
		"/feed",
		"/feed/",
		"/rss",
		"/rss/",
		"/feed.xml",
		"/rss.xml",
		"/atom.xml",
		"/index.xml",
		"/feeds/posts/default",
		"/blog/feed",
		"/blog/rss",
		"/posts/index.xml",
	}

	logf("Probing %d common feed paths", len(commonPaths))
	for _, path := range commonPaths {
		candidate := resolveURL(base, path)
		if candidate == "" {
			continue
		}
		logf("Probing %s", candidate)
		info, ok := isValidFeed(c, candidate, userAgent)
		if ok {
			logf("Valid feed found at %s (title: %q)", candidate, info.Title)
			results = append(results, model.FeedResult{
				URL:    candidate,
				Source: "Common path probe (" + path + ")",
				Title:  info.Title,
			})
		}
	}

	return results, nil
}
