package main

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestParseWikipediaTag(t *testing.T) {
	language, title, ok := parseWikipediaTag("ta:திருவள்ளுவர் சிலை")
	if !ok || language != "ta" || title != "திருவள்ளுவர் சிலை" {
		t.Fatalf("parseWikipediaTag() = %q, %q, %v", language, title, ok)
	}
	for _, value := range []string{"", "place-name", ":title", "en:"} {
		if _, _, ok := parseWikipediaTag(value); ok {
			t.Errorf("parseWikipediaTag(%q) unexpectedly succeeded", value)
		}
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestFetchWikipediaDetailsUsesArticleAndReturnsSummaryAndImage(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/api/rest_v1/page/summary/Thiruvalluvar_Statue" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		if r.Header.Get("User-Agent") == "" {
			t.Error("missing identifying User-Agent")
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"extract":"A famous statue.","thumbnail":{"source":"https://upload.wikimedia.org/statue.jpg"}}`)),
			Header:     make(http.Header),
			Request:    r,
		}, nil
	})}

	details, err := fetchWikipediaDetails(context.Background(), client, []attraction{{Wikipedia: "en:Thiruvalluvar Statue"}}, "https://en.wikipedia.org")
	if err != nil {
		t.Fatal(err)
	}
	got := details["en:Thiruvalluvar Statue"]
	if got.Description != "A famous statue." || got.ImageURL != "https://upload.wikimedia.org/statue.jpg" {
		t.Fatalf("unexpected Wikipedia details: %+v", got)
	}
}

func TestApplyWikipediaDetailsPreservesExistingDescription(t *testing.T) {
	places := []attraction{{Wikipedia: "en:Sample", Description: "Curated description"}}
	applyWikipediaDetails(places, map[string]wikipediaDetails{
		"en:Sample": {Description: "Wikipedia summary", ImageURL: "https://upload.wikimedia.org/sample.jpg"},
	})
	if places[0].Description != "Curated description" || places[0].WikipediaImageURL != "https://upload.wikimedia.org/sample.jpg" {
		t.Fatalf("Wikipedia enrichment overwrote or missed data: %+v", places[0])
	}
}

func TestApplyWikipediaDetailsDoesNotAddNonEnglishDescription(t *testing.T) {
	places := []attraction{{Wikipedia: "fr:Mémorial Gandhi"}}
	applyWikipediaDetails(places, map[string]wikipediaDetails{
		"fr:Mémorial Gandhi": {Description: "Résumé français", ImageURL: "https://upload.wikimedia.org/sample.jpg"},
	})
	if places[0].Description != "" || places[0].WikipediaImageURL == "" {
		t.Fatalf("unexpected non-English enrichment: %+v", places[0])
	}
}
