package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

type feed struct {
	Entries []entry `xml:"entry"`
}

// apiClient carries a timeout; do() adds the User-Agent. Proxies such as
// Cloudflare treat agent-less requests from datacenter IPs as bots (403).
var apiClient = &http.Client{Timeout: 30 * time.Second}

const userAgent = "blog-announcer/1.0 (https://stdout.alesr.me)"

// do sends req with the announcer User-Agent.
func do(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", userAgent)
	return apiClient.Do(req)
}

// statusError includes a response-body snippet so proxy blocks are diagnosable.
func statusError(op string, resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 200))
	if len(body) > 0 {
		return fmt.Errorf("%s: status %s: %s", op, resp.Status, strings.TrimSpace(string(body)))
	}
	return fmt.Errorf("%s: status %s", op, resp.Status)
}

type entry struct {
	Title     string `xml:"title"`
	Summary   string `xml:"summary"`
	Link      []link `xml:"link"`
	Published string `xml:"published"`
}

type link struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
}

func (e entry) canonicalURL() string {
	for _, l := range e.Link {
		if l.Rel == "" || l.Rel == "alternate" {
			return l.Href
		}
	}
	return ""
}

func main() {
	pds := getenv("PDS_URL", "https://bsky.social")
	handle := os.Getenv("HANDLE")
	password := os.Getenv("APP_PASSWORD")
	feedURL := getenv("FEED_URL", "https://stdout.alesr.me/feed.xml")
	prefix := getenv("POST_PREFIX", "New post: ")

	if handle == "" || password == "" {
		fatal("HANDLE and APP_PASSWORD must be set")
	}

	accessJwt, err := createSession(pds, handle, password)
	if err != nil {
		fatal("createSession: %v", err)
	}

	entries, err := fetchFeed(feedURL)
	if err != nil {
		fatal("fetch feed: %v", err)
	}

	posted, err := postedURIs(pds, accessJwt, handle)
	if err != nil {
		fatal("listRecords: %v", err)
	}

	// process oldest-first so catch-up posts are chronological
	count := 0
	for i := len(entries) - 1; i >= 0; i-- {
		e := entries[i]
		uri := e.canonicalURL()
		if uri == "" {
			continue
		}
		if posted[uri] {
			continue
		}
		text := truncate(prefix+e.Title, 300)
		desc := truncate(stripHTML(e.Summary), 300)
		if desc == "" {
			desc = truncate(e.Title, 300)
		}
		if err := createRecord(pds, accessJwt, handle, text, uri, truncate(e.Title, 300), desc); err != nil {
			fmt.Fprintf(os.Stderr, "post failed for %s: %v\n", uri, err)
			continue
		}
		fmt.Printf("posted: %s\n", uri)
		count++
	}
	if count == 0 {
		fmt.Println("no new posts")
	}
}

func createSession(pds, handle, password string) (string, error) {
	body, _ := json.Marshal(map[string]string{"identifier": handle, "password": password})
	req, err := http.NewRequest(http.MethodPost, pds+"/xrpc/com.atproto.server.createSession", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", statusError("createSession", resp)
	}
	var out struct {
		AccessJwt string `json:"accessJwt"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	return out.AccessJwt, nil
}

func fetchFeed(feedURL string) ([]entry, error) {
	req, err := http.NewRequest(http.MethodGet, feedURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, statusError("fetch feed", resp)
	}
	var f feed
	if err := xml.NewDecoder(resp.Body).Decode(&f); err != nil {
		return nil, err
	}
	return f.Entries, nil
}

func postedURIs(pds, accessJwt, handle string) (map[string]bool, error) {
	u := pds + "/xrpc/com.atproto.repo.listRecords?repo=" + url.QueryEscape(handle) +
		"&collection=app.bsky.feed.post&limit=100"
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessJwt)
	resp, err := do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, statusError("listRecords", resp)
	}
	var out struct {
		Records []struct {
			Value json.RawMessage `json:"value"`
		} `json:"records"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	uris := make(map[string]bool)
	for _, r := range out.Records {
		var v struct {
			Embed *struct {
				External *struct {
					URI string `json:"uri"`
				} `json:"external"`
			} `json:"embed"`
		}
		if err := json.Unmarshal(r.Value, &v); err != nil {
			continue
		}
		if v.Embed != nil && v.Embed.External != nil && v.Embed.External.URI != "" {
			uris[v.Embed.External.URI] = true
		}
	}
	return uris, nil
}

func createRecord(pds, accessJwt, handle, text, uri, title, desc string) error {
	rec := map[string]any{
		"$type":     "app.bsky.feed.post",
		"text":      text,
		"createdAt": time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
		"embed": map[string]any{
			"$type": "app.bsky.embed.external",
			"external": map[string]string{
				"uri":         uri,
				"title":       title,
				"description": desc,
			},
		},
	}
	body, _ := json.Marshal(map[string]any{
		"repo":       handle,
		"collection": "app.bsky.feed.post",
		"record":     rec,
	})
	req, err := http.NewRequest(http.MethodPost, pds+"/xrpc/com.atproto.repo.createRecord", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessJwt)
	resp, err := do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return statusError("createRecord", resp)
	}
	return nil
}

var tagRe = regexp.MustCompile(`<[^>]*>`)

func stripHTML(s string) string {
	return strings.TrimSpace(html.UnescapeString(tagRe.ReplaceAllString(s, "")))
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "bsky-post: "+format+"\n", args...)
	os.Exit(1)
}
