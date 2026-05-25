package feed

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"

	"github.com/ramadhantriyant/feedfinder/internal/model"
)

var MIMETypes = []string{
	"application/rss+xml",
	"application/atom+xml",
	"application/feed+json",
	"application/xml",
	"text/xml",
}

func IsFeedMIME(mediaType string) bool {
	return mediaType == "application/rss+xml" ||
		mediaType == "application/atom+xml" ||
		mediaType == "application/feed+json"
}

func ParseInfo(r io.Reader) (model.FeedInfo, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return model.FeedInfo{}, err
	}

	var rss model.RSSRoot
	if err := xml.Unmarshal(data, &rss); err == nil && rss.XMLName.Local == "rss" {
		return model.FeedInfo{
			Title:    strings.TrimSpace(rss.Channel.Title),
			HasItems: len(rss.Channel.Items) > 0,
		}, nil
	}

	var atom model.AtomFeed
	if err := xml.Unmarshal(data, &atom); err == nil && atom.XMLName.Local == "feed" {
		return model.FeedInfo{
			Title:    strings.TrimSpace(atom.Title.Text),
			HasItems: len(atom.Entries) > 0,
		}, nil
	}

	s := strings.TrimSpace(string(data))
	if strings.HasPrefix(s, "{") && strings.Contains(s, `"version"`) &&
		strings.Contains(s, `"jsonfeed.org"`) {
		title := ""
		if _, after, ok := strings.Cut(s, `"title"`); ok {
			rest := strings.TrimLeft(after, ` \t:`)
			if len(rest) > 0 && rest[0] == '"' {
				if t, _, ok := strings.Cut(rest[1:], `"`); ok {
					title = t
				}
			}
		}
		return model.FeedInfo{Title: title, HasItems: true}, nil
	}

	return model.FeedInfo{}, fmt.Errorf("not a recognised feed format")
}
