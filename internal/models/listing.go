package models

import "time"

// Listing represents an individual listing from Avito.ru
type Listing struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	Description string            `json:"description,omitempty"`
	Price       Price             `json:"price"`
	URL         string            `json:"url"`
	ImageURLs   []string          `json:"imageUrls,omitempty"`
	Location    string            `json:"location,omitempty"`
	CategoryID  string            `json:"categoryId,omitempty"`
	CategoryURL string            `json:"categoryUrl,omitempty"`
	PublishedAt time.Time         `json:"publishedAt,omitempty"`
	Attributes  map[string]string `json:"attributes,omitempty"`
}

// Price represents a price with currency information
type Price struct {
	Value    float64 `json:"value"`
	Currency string  `json:"currency"`
	Text     string  `json:"text"`
}
