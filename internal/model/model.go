package model

import "encoding/xml"

type FeedResult struct {
	URL    string
	Source string
	Title  string
}

type FeedInfo struct {
	Title    string
	HasItems bool
}

type RSSRoot struct {
	XMLName xml.Name `xml:"rss"`
	Channel struct {
		Title string `xml:"title"`
		Items []struct {
			Title string `xml:"title"`
		} `xml:"item"`
	} `xml:"channel"`
}

type AtomFeed struct {
	XMLName xml.Name `xml:"feed"`
	Title   struct {
		Text string `xml:",chardata"`
	} `xml:"title"`
	Entries []struct {
		Title string `xml:"title"`
	} `xml:"entry"`
}

type HTMLLink struct {
	Rel  string
	Type string
	Href string
}
