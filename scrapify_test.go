package scrapify_test

import (
	"context"
	"sync"
	"testing"

	"github.com/ricardocastanho/scrapify"
	"github.com/stretchr/testify/assert"
)

// mockScraper is a mock implementation of the IScraper interface for testing.
type mockScraper struct {
	urls      []string
	nextPages []string
	data      string
}

func (m *mockScraper) GetUrls(ctx context.Context, url string) ([]string, []string) {
	return m.urls, m.nextPages
}

func (m *mockScraper) GetData(ctx context.Context, ch chan<- string, data *string, url string) {
	*data = m.data + "-" + url
	ch <- *data
}

// TestScraper_Run_SingleStrategy tests the scraper with a single strategy.
func TestScraper_Run_SingleStrategy(t *testing.T) {
	ctx := context.Background()

	// Mock scraper returning a single URL with no next page
	mock := &mockScraper{
		urls:      []string{"https://example.com/item1"},
		nextPages: []string{},
		data:      "scrapedData",
	}

	var collected []string
	mu := sync.Mutex{}

	// Callback to collect scraped data
	callback := func(data string) {
		mu.Lock()
		defer mu.Unlock()
		collected = append(collected, data)
	}

	strategies := []scrapify.ScraperStrategy[string]{
		{Scraper: mock, Url: "https://example.com/start"},
	}

	s := scrapify.NewScraper[string](strategies, callback, 0)
	s.Run(ctx)

	// Verify that the scraped data was collected correctly
	assert.Len(t, collected, 1)
	assert.Equal(t, "scrapedData-https://example.com/item1", collected[0])
}

// TestScraper_Run_MultipleStrategies tests the scraper with multiple strategies and pagination.
func TestScraper_Run_MultipleStrategies(t *testing.T) {
	ctx := context.Background()

	// First mock scraper returns two URLs and a next page
	mock1 := &mockScraper{
		urls:      []string{"https://example.com/item1", "https://example.com/item2"},
		nextPages: []string{"https://example.com/page2"},
		data:      "scraper1",
	}

	// Second mock scraper (for page2) returns one URL and no next pages
	mock2 := &mockScraper{
		urls:      []string{"https://example.com/item3"},
		nextPages: []string{},
		data:      "scraper2",
	}

	var collected []string
	mu := sync.Mutex{}

	// Callback to collect scraped data
	callback := func(data string) {
		mu.Lock()
		defer mu.Unlock()
		collected = append(collected, data)
	}

	// Multiple strategies with different scrapers
	strategies := []scrapify.ScraperStrategy[string]{
		{Scraper: mock1, Url: "https://example.com/start"},
		{Scraper: mock2, Url: "https://example.com/page2"},
	}

	s := scrapify.NewScraper[string](strategies, callback, 0)
	s.Run(ctx)

	// Verify that all data has been collected
	assert.Len(t, collected, 3)

	// Check that the collected data contains expected items (order may vary)
	expected := map[string]bool{
		"scraper1-https://example.com/item1": true,
		"scraper1-https://example.com/item2": true,
		"scraper2-https://example.com/item3": true,
	}

	for _, item := range collected {
		assert.True(t, expected[item])
	}
}

// TestScraper_Run_MultipleIndependentScrapers tests multiple independent scrapers working together.
func TestScraper_Run_MultipleIndependentScrapers(t *testing.T) {
	ctx := context.Background()

	// Mock scraper 1 returns two URLs and no pagination
	mock1 := &mockScraper{
		urls:      []string{"https://site1.com/item1", "https://site1.com/item2"},
		nextPages: []string{},
		data:      "mock1",
	}

	// Mock scraper 2 returns one URL and no pagination
	mock2 := &mockScraper{
		urls:      []string{"https://site2.com/item1"},
		nextPages: []string{},
		data:      "mock2",
	}

	// Mock scraper 3 returns no URLs but one nextPage
	mock3 := &mockScraper{
		urls:      []string{},
		nextPages: []string{"https://site3.com/page2"},
		data:      "mock3",
	}

	// Mock scraper 4 for page2
	mock4 := &mockScraper{
		urls:      []string{"https://site3.com/item1"},
		nextPages: []string{},
		data:      "mock4",
	}

	var collected []string
	mu := sync.Mutex{}

	callback := func(data string) {
		mu.Lock()
		defer mu.Unlock()
		collected = append(collected, data)
	}

	strategies := []scrapify.ScraperStrategy[string]{
		{Scraper: mock1, Url: "https://site1.com/start"},
		{Scraper: mock2, Url: "https://site2.com/start"},
		{Scraper: mock3, Url: "https://site3.com/start"},
		{Scraper: mock4, Url: "https://site3.com/page2"},
	}

	s := scrapify.NewScraper[string](strategies, callback, 0)
	s.Run(ctx)

	// Verifying the collected data
	assert.Len(t, collected, 4)

	expected := map[string]bool{
		"mock1-https://site1.com/item1": true,
		"mock1-https://site1.com/item2": true,
		"mock2-https://site2.com/item1": true,
		"mock4-https://site3.com/item1": true,
	}

	for _, item := range collected {
		assert.True(t, expected[item], "unexpected item collected: %s", item)
	}
}
