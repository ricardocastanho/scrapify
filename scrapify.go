package scrapify

import (
	"context"
	"sync"
	"time"
)

// IScraper is an interface that defines the methods required for any scraper implementation.
// T is a generic type representing the data being scraped.
type IScraper[T any] interface {
	// GetUrls retrieves the URLs from the current page and the URLs of the next pages for pagination.
	GetUrls(ctx context.Context, url string) ([]string, []string)

	// GetData scrapes the data from a given URL and sends it to the provided channel.
	GetData(ctx context.Context, ch chan<- T, data *T, url string)
}

// Scraper represents the main structure that coordinates scraping jobs across multiple strategies.
// It manages the scraping process, handles concurrency, and invokes a user-defined callback when data is scraped.
type Scraper[T any] struct {
	strategy     []ScraperStrategy[T] // A list of scraping strategies, each with a unique configuration.
	jobs         chan ScraperJob[T]   // Channel that holds scraping jobs to be processed.
	ch           chan T               // Channel through which scraped data is passed.
	wg           sync.WaitGroup       // Synchronizes the goroutines to ensure proper job completion.
	scrapedUrls  map[string]bool      // Tracks URLs that have already been scraped to avoid duplicates.
	callback     func(T)              // User-provided callback function for processing scraped data.
	requestDelay time.Duration        // User-defined delay between requests (default is 0, meaning no delay).
}

// ScraperStrategy defines the strategy for scraping a specific URL with a given scraper implementation.
// T represents the type of data being scraped.
type ScraperStrategy[T any] struct {
	Scraper IScraper[T] // The scraper implementation used to scrape data from the target URL.
	Url     string      // The URL to start scraping from.
}

// ScraperJob represents a job containing the scraper and a list of URLs to process.
// T is the type of data being scraped.
type ScraperJob[T any] struct {
	scraper IScraper[T] // The scraper instance used to perform the scraping.
	urls    []string    // A list of URLs to be processed for scraping.
}

// NewScraper creates a new Scraper instance.
// logger is used for logging, s is the list of strategies to run, callback is the function that processes scraped data, and requestDelay is the optional delay between requests.
func NewScraper[T any](s []ScraperStrategy[T], callback func(T), requestDelay time.Duration) *Scraper[T] {
	return &Scraper[T]{
		strategy:     s,
		jobs:         make(chan ScraperJob[T]),
		ch:           make(chan T),
		scrapedUrls:  make(map[string]bool),
		callback:     callback,
		requestDelay: requestDelay, // Set the delay between requests.
	}
}

// getData is responsible for processing jobs from the jobs channel and invoking the provided scraper.
// It also ensures that the data is sent to the channel and the callback is called when the data is received.
func (s *Scraper[T]) getData(ctx context.Context) {
	go func() {
		for job := range s.jobs {
			for _, url := range job.urls {
				go func(url string) {
					defer s.wg.Done()

					// Skip already scraped URLs to avoid duplication.
					if _, ok := s.scrapedUrls[url]; ok {
						return
					}

					var data T
					// Scrape the data from the URL and send it to the channel.
					job.scraper.GetData(ctx, s.ch, &data, url)
					s.scrapedUrls[url] = true

				}(url)

				// Apply the user-defined delay between requests.
				if s.requestDelay > 0 {
					time.Sleep(s.requestDelay)
				}
			}
		}
	}()

	go func() {
		select {
		case <-ctx.Done():
			return
		default:
			// Continuously process data from the channel and invoke the callback.
			for data := range s.ch {
				s.callback(data)
			}
		}
	}()
}

// runScraper starts the scraping process for a given strategy.
// It handles both the retrieval of data URLs and pagination to new pages.
func (s *Scraper[T]) runScraper(ctx context.Context, strategy ScraperStrategy[T]) {
	defer s.wg.Done()

	// Get URLs from the current page and the next pages for further scraping.
	urls, nextPages := strategy.Scraper.GetUrls(ctx, strategy.Url)
	s.scrapedUrls[strategy.Url] = true
	s.wg.Add(len(urls))
	s.wg.Add(1)

	// Send the URLs to the jobs channel for further processing.
	s.jobs <- ScraperJob[T]{scraper: strategy.Scraper, urls: urls}

	// Process the next pages recursively.
	for _, newUrl := range nextPages {
		if _, ok := s.scrapedUrls[newUrl]; ok {
			continue
		}

		s.wg.Add(1)
		s.scrapedUrls[newUrl] = true

		// Recursively call runScraper to handle pagination.
		go s.runScraper(ctx, ScraperStrategy[T]{Scraper: strategy.Scraper, Url: newUrl})
	}
}

// Run starts the entire scraping process by running each strategy and managing concurrency.
// It waits for all scraping jobs to complete before closing the channels.
func (s *Scraper[T]) Run(ctx context.Context) {
	// Add all strategies to the wait group.
	s.wg.Add(len(s.strategy))

	// Start processing jobs and data.
	s.getData(ctx)

	// Run each scraping strategy in a separate goroutine.
	for i := range s.strategy {
		strategy := s.strategy[i]
		go s.runScraper(ctx, strategy)
	}

	// Wait for all jobs to complete.
	s.wg.Wait()

	// Close the channels after all work is done.
	close(s.jobs)
	close(s.ch)
}
