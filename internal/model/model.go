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
	XMLName xml.Name   `xml:"rss"`
	Channel RSSChannel `xml:"channel"`
}

type RSSChannel struct {
	Title string `xml:"title"`
	Items []struct {
		Title string `xml:"title"`
	} `xml:"item"`
}

type AtomFeed struct {
	XMLName xml.Name    `xml:"feed"`
	Title   AtomText    `xml:"title"`
	Entries []AtomEntry `xml:"entry"`
}

type AtomText struct {
	Text string `xml:",chardata"`
}

type AtomEntry struct {
	Title string `xml:"title"`
}

type HTMLLink struct {
	Rel  string
	Type string
	Href string
}
