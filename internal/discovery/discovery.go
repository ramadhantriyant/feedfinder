package discovery

import (
	"fmt"
	"io"
	"mime"
	"net/http"
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

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("server returned HTTP %d", resp.StatusCode)
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
		info, _ := feed.ParseInfo(resp.Body)
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
	links := ExtractLinkTags(body)
	logf("Found %d <link> tags", len(links))

	for _, link := range links {
		isAlternate := HasToken(link.Rel, "alternate")
		isSelf := HasToken(link.Rel, "self")
		if !isAlternate && !isSelf {
			continue
		}
		ltype := strings.ToLower(strings.TrimSpace(link.Type))
		if !slices.Contains(feed.MIMETypes, ltype) || link.Href == "" {
			continue
		}

		resolved := ResolveURL(base, link.Href)
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
		candidate := ResolveURL(base, path)
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

func isValidFeed(c *http.Client, feedURL, userAgent string) (model.FeedInfo, bool) {
	resp, err := client.DoRequest(c, feedURL, userAgent)
	if err != nil {
		return model.FeedInfo{}, false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return model.FeedInfo{}, false
	}
	info, err := feed.ParseInfo(resp.Body)
	if err != nil {
		return model.FeedInfo{}, false
	}
	return info, info.HasItems || info.Title != ""
}

func ExtractLinkTags(body string) []model.HTMLLink {
	var links []model.HTMLLink

	lower := strings.ToLower(body)
	start := 0

	for {
		idx := strings.Index(lower[start:], "<link")
		if idx < 0 {
			break
		}
		idx += start

		end := strings.IndexAny(lower[idx:], ">")
		if end < 0 {
			break
		}
		end += idx + 1

		tag := body[idx:end]
		link := parseTagAttrs(tag)
		if link.Href != "" {
			links = append(links, link)
		}
		start = end
	}

	return links
}

func parseTagAttrs(tag string) model.HTMLLink {
	var link model.HTMLLink
	lower := strings.ToLower(tag)

	link.Rel = attrValue(lower, tag, "rel")
	link.Type = attrValue(lower, tag, "type")
	link.Href = attrValue(lower, tag, "href")
	return link
}

func attrValue(lowerTag, origTag, attr string) string {
	needle := attr + "="
	idx := strings.Index(lowerTag, needle)
	if idx < 0 {
		return ""
	}
	idx += len(needle)
	if idx >= len(origTag) {
		return ""
	}
	rest := origTag[idx:]
	if len(rest) == 0 {
		return ""
	}
	var val string
	if rest[0] == '"' || rest[0] == '\'' {
		q := rest[0]
		end := strings.IndexByte(rest[1:], q)
		if end < 0 {
			return ""
		}
		val = rest[1 : end+1]
	} else {
		end := strings.IndexAny(rest, " \t\r\n>")
		if end < 0 {
			val = rest
		} else {
			val = rest[:end]
		}
	}
	return strings.TrimSpace(val)
}

func HasToken(list, token string) bool {
	list = strings.ToLower(strings.TrimSpace(list))
	token = strings.ToLower(token)
	for t := range strings.FieldsSeq(list) {
		if t == token {
			return true
		}
	}
	return false
}

func ResolveURL(base *url.URL, href string) string {
	href = strings.TrimSpace(href)
	if href == "" {
		return ""
	}
	parsed, err := url.Parse(href)
	if err != nil {
		return ""
	}
	return base.ResolveReference(parsed).String()
}
