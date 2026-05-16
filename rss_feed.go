package main

import (
	"context"
	"net/http"
	"io"
	"encoding/xml"
)

type RSSFeed struct {

}


func fetchFeed(ctx context.Context, feedURL string) (*RSSFeed, error) {
	req, err := http.NewRequestWithContext(ctx feedURL, http.MethodGet)
	if err != nil {
		return &RSSFeed{}, err
	}
	req.Header.Set("User-Agent", "gator")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return &RSSFeed{}, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return &RSSFeed{}, err
	}

	var data RSSFeed
	err = xml.Unmarshal(bodyBytes, &data)
	if err != nil {
		return &RSSFeed{}, err
	}

	return &data, nil
}
