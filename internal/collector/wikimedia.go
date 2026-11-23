package collector

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"realtime-dashboard/internal/metrics"
	"time"
)

const wikimediaStreamURL = "https://stream.wikimedia.org/v2/stream/recentchange"
const tryDelay = time.Second * 5

var trackedWikis = map[string]bool{
	"enwiki":        true,
	"enwikiversity": true,
	"enwikiquote":   true,
	"commonswiki":   true,
	"wikidatawiki":  true,
}

type recentChange struct {
	Wiki      string `json:"wiki"`
	Type      string `json:"type"`
	Bot       bool   `json:"bot"`
	Timestamp int64  `json:"timestamp"`
}

type WikimediaSource struct {
	url        string
	client     *http.Client
	retryDelay time.Duration
}

var _ Source = (*WikimediaSource)(nil)

func NewWikimediaSource() *WikimediaSource {
	return &WikimediaSource{
		url:        wikimediaStreamURL,
		client:     &http.Client{},
		retryDelay: tryDelay,
	}
}

func (s *WikimediaSource) Name() string {
	return "wikimedia"
}

func (s *WikimediaSource) Run(ctx context.Context, out chan<- metrics.Event) error {
	for {
		err := s.stream(ctx, out)
		if ctx.Err() != nil {
			return nil
		}
		if err != nil {
			log.Printf("Wikimedia stream ended: %v trying again in %v", err, s.retryDelay)
		}
		select {
		case <-time.After(s.retryDelay):
		case <-ctx.Done():
			return nil
		}
	}
}

func (s *WikimediaSource) stream(ctx context.Context, out chan<- metrics.Event) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("User-Agent", "realtime-dashboard-learning/1.0")
	resp, err := s.client.Do(req)

	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status, code: %d", resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		data, found := bytes.CutPrefix(scanner.Bytes(), []byte("data: "))
		if !found {
			continue
		}
		var wikiData recentChange
		err = json.Unmarshal(data, &wikiData)
		if err != nil {
			continue
		}
		events := toEvents(wikiData, time.Now(), s.Name())

		for _, event := range events {
			sent := send(ctx, out, event)
			if !sent {
				return nil
			}
		}
	}
	if err = scanner.Err(); err != nil {
		return err
	}
	return fmt.Errorf("stream closed")
}

func toEvents(rc recentChange, now time.Time, source string) []metrics.Event {
	if rc.Type != "new" && rc.Type != "edit" {
		return nil
	}

	newEvent := func(name string) metrics.Event {
		return metrics.Event{
			Name:      name,
			Value:     1,
			Source:    source,
			Timestamp: now,
		}
	}
	events := []metrics.Event{newEvent("wiki_edit")}
	if rc.Bot {
		events = append(events, newEvent("bot_edit"))
	}
	if trackedWikis[rc.Wiki] {
		events = append(events, newEvent("edit_"+rc.Wiki))
	}
	return events
}
