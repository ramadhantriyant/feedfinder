package discovery

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/ramadhantriyant/feedfinder/internal/client"
	"github.com/ramadhantriyant/feedfinder/internal/feed"
	"github.com/ramadhantriyant/feedfinder/internal/model"
)

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

func extractLinkTags(body string) []model.HTMLLink {
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

func hasToken(list, token string) bool {
	list = strings.ToLower(strings.TrimSpace(list))
	token = strings.ToLower(token)
	for t := range strings.FieldsSeq(list) {
		if t == token {
			return true
		}
	}
	return false
}

func resolveURL(base *url.URL, href string) string {
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
