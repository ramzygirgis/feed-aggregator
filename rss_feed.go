package main

import (
	"context"
	"net/http"
	"io"
	"encoding/xml"
	"html"
)

type RSSFeed struct {
	Channel struct {
		Title       string    `xml:"title"`
		Link        string    `xml:"link"`
		Description string    `xml:"description"`
		Item        []RSSItem `xml:"item"`
	} `xml:"channel"`
}

type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}


func (r *RSSFeed) decodeHTML() {
	r.Channel.Title = html.UnescapeString(r.Channel.Title)
	r.Channel.Description = html.UnescapeString(r.Channel.Description)
	for i := 0; i < len(r.Channel.Item); i++ {
		r.Channel.Item[i].Title = html.UnescapeString(r.Channel.Item[i].Title)
		r.Channel.Item[i].Description = html.UnescapeString(r.Channel.Item[i].Description)
	}
}


func fetchFeed(ctx context.Context, feedURL string) (*RSSFeed, error) {
	req, err := http.NewRequestWithContext(ctx, feedURL, http.MethodGet)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "gator")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data RSSFeed
	err = xml.Unmarshal(bodyBytes, &data)
	if err != nil {
		return nil, err
	}

	data.decodeHTML()

	return &data, nil
}


