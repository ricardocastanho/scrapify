# Scrapify

Scrapify is a flexible and decoupled Go package for scraping paginated web pages. It is designed to work with any type of page that lists items and supports pagination. The package allows users to provide their own scraping logic through a callback mechanism, making it versatile and easy to integrate into various projects.

## Features

- **Flexible Scraping**: Use custom scraper implementations by providing an `IScraper` interface.
- **Paginated Requests**: Automatically handles pagination and scraping of multiple pages.
- **Callback Processing**: Users can define a callback function to process scraped data.
- **Configurable Request Intervals**: Set the interval between requests to manage load and avoid rate limiting.

## Installation

To use Scrapify in your Go project, first install the package with:

```bash
go get github.com/ricardocastanho/scrapify
```

## Usage

Here is a basic example of how to use Scrapify:

1- Define Your Scraper Implementation

Implement the `IScraper` interface with your custom scraping logic.

```go
package main

import (
    "context"
    "fmt"
    "github.com/ricardocastanho/scrapify"
)

type ExampleScraper struct{}

func (e ExampleScraper) GetUrls(ctx context.Context, url string) ([]string, []string) {
    // Implement URL extraction logic here
    return []string{"url1", "url2"}, []string{"nextPageUrl"}
}

func (e ExampleScraper) GetData(ctx context.Context, ch chan<- string, data *string, url string) {
    // Implement data extraction logic here
    *data = "Example data from " + url
    ch <- *data
}
```

2- Create and Run the Scraper

Instantiate the `Scraper` with your `ScraperStrategy` and a callback function.

```go
package main

import (
    "context"
    "fmt"
    "time"
    "github.com/ricardocastanho/scrapify"
)

func main() {
    logger := &Logger{} // Define or use an existing Logger implementation

    strategy := []scrapify.ScraperStrategy[string]{
        {
            Scraper: ExampleScraper{},
            Url:     "https://example.com",
        },
    }

    callback := func(data string) {
        fmt.Println("Processed data:", data)
    }

    scraper := scrapify.NewScraper(logger, strategy, callback, time.Second*2)
    scraper.Run(context.Background())
}
```

## API

### `type Scraper[T any]`

`Scraper` is the main struct for orchestrating the scraping process.

- `func NewScraper[T any](logger *Logger, s []ScraperStrategy[T], callback func(T), interval time.Duration) *Scraper[T]`: Creates a new Scraper instance.

- `func (s *Scraper[T]) Run(ctx context.Context)`: Starts the scraping process.

- `func (s *Scraper[T]) getData(ctx context.Context)`: Handles data extraction and processing.

- `func (s *Scraper[T]) runScraper(ctx context.Context, strategy ScraperStrategy[T])`: Executes the scraping logic for each strategy.

### type IScraper[T any]

`IScraper` is an interface for implementing custom scraping logic.

- `GetUrls(ctx context.Context, url string) ([]string, []string)`: Returns the URLs of the current page and the next pages.

- `GetData(ctx context.Context, ch chan<- T, data *T, url string)`: Performs the data scraping for a given URL and sends the result to the channel.

## Contributing

Feel free to open issues or submit pull requests if you have suggestions or improvements.

## License

This project is licensed under the MIT License. See the LICENSE file for details.
