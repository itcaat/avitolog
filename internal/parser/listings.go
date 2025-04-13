package parser

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
	"github.com/itcaat/avitolog/internal/models"
)

var (
	// Regex to extract item ID from URL or data attributes
	itemIDRegex = regexp.MustCompile(`_(\d+)$|/(\d+)$`)
	// Regex to extract price value
	priceRegex = regexp.MustCompile(`[\d\s,.]+`)
	// Regex to detect if the URL is a catalog page
	catalogRegex = regexp.MustCompile(`/catalog/`)

	// Rate limiting
	minRequestInterval = 3 * time.Second
	lastRequestTime    = time.Now().Add(-minRequestInterval)
	maxRetries         = 3
)

// waitForRateLimit ensures we don't send requests too quickly
func waitForRateLimit() {
	elapsed := time.Since(lastRequestTime)
	if elapsed < minRequestInterval {
		sleepTime := minRequestInterval - elapsed
		log.Printf("Rate limiting: Waiting %v before next request", sleepTime)
		time.Sleep(sleepTime)
	}
	lastRequestTime = time.Now()
}

// GetListings fetches listings from a given category URL
func GetListings(categoryURL string, limit int) ([]models.Listing, error) {
	// Check if this is a catalog URL and handle it differently if needed
	if catalogRegex.MatchString(categoryURL) {
		return handleCatalogPage(categoryURL, limit)
	}

	var listings []models.Listing

	c := colly.NewCollector(
		colly.AllowedDomains("www.avito.ru", "avito.ru"),
		colly.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
		colly.MaxDepth(1),
	)

	// Set up retry mechanism
	c.SetRequestTimeout(30 * time.Second)

	// Randomize delay between requests
	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		RandomDelay: 5 * time.Second,
		Delay:       3 * time.Second,
	})

	// Add debugging callbacks
	c.OnRequest(func(r *colly.Request) {
		log.Println("Visiting", r.URL)
		// Respect rate limiting
		waitForRateLimit()
	})

	c.OnError(func(r *colly.Response, err error) {
		log.Println("Error:", err)
		if r.StatusCode == 429 {
			log.Println("Rate limited, waiting longer before retry")
			time.Sleep(10 * time.Second)

			// Try to retry with a different user agent
			retries := 0
			for retries < maxRetries {
				retries++
				log.Printf("Retry %d of %d...", retries, maxRetries)
				time.Sleep(5 * time.Second * time.Duration(retries))

				// Alternate user agents
				userAgents := []string{
					"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
					"Mozilla/5.0 (iPhone; CPU iPhone OS 13_2_3 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/13.0.3 Mobile/15E148 Safari/604.1",
					"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/92.0.4515.107 Safari/537.36",
				}

				r.Request.Headers.Set("User-Agent", userAgents[retries%len(userAgents)])

				// Retry the request
				err = c.Request("GET", r.Request.URL.String(), nil, r.Request.Ctx, nil)
				if err == nil {
					return
				}

				log.Printf("Retry %d failed: %v", retries, err)
			}
		}
	})

	c.OnResponse(func(r *colly.Response) {
		log.Printf("Received response from listings page, size: %d bytes\n", len(r.Body))
	})

	// Parse listings from search results
	c.OnHTML("div[data-marker='catalog-serp']", func(e *colly.HTMLElement) {
		log.Println("Found listings container")

		// Look for item cards with different possible selectors
		itemSelectors := []string{
			"div[data-marker='item']",
			"div[data-marker='item-card']",
			"div.item",
			"div.item-card",
			"div.iva-item-root",
		}

		for _, selector := range itemSelectors {
			count := 0
			e.ForEach(selector, func(_ int, item *colly.HTMLElement) {
				if limit > 0 && count >= limit {
					return
				}

				listing := parseListing(item)
				if listing.ID != "" && listing.Title != "" {
					listing.CategoryURL = categoryURL
					listings = append(listings, listing)
					count++
				}
			})

			if count > 0 {
				log.Printf("Found %d listings using selector: %s\n", count, selector)
				break
			}
		}
	})

	// If no specific item container found, use a more general approach
	c.OnHTML("body", func(e *colly.HTMLElement) {
		if len(listings) > 0 {
			return // Skip if we already found listings
		}

		// Try to find any element that might be a listing
		log.Println("Trying alternative method to find listings")

		count := 0
		e.DOM.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
			if limit > 0 && count >= limit {
				return
			}

			href, exists := s.Attr("href")
			if !exists {
				return
			}

			// Check if this link points to an item page
			if strings.Contains(href, "/item/") {
				// This might be a listing
				title := strings.TrimSpace(s.Text())
				if title == "" {
					// Try to find title in child elements
					title = strings.TrimSpace(s.Find("h3, h4, h2, div.title, div.snippet-title").First().Text())
				}

				if title != "" {
					listing := models.Listing{
						Title: title,
						URL:   normalizeURL(href),
					}

					// Try to extract ID from URL
					matches := itemIDRegex.FindStringSubmatch(href)
					if len(matches) > 1 {
						if matches[1] != "" {
							listing.ID = matches[1]
						} else if matches[2] != "" {
							listing.ID = matches[2]
						}
					}

					// Look for price near this element
					priceText := strings.TrimSpace(s.Find("span.price, div.price, *[data-marker='item-price']").First().Text())
					if priceText != "" {
						listing.Price = parsePrice(priceText)
					}

					listing.CategoryURL = categoryURL
					listings = append(listings, listing)
					count++
				}
			}
		})

		log.Printf("Found %d listings using alternative method\n", count)
	})

	// Wait for rate limiting before starting
	waitForRateLimit()

	err := c.Visit(categoryURL)
	if err != nil {
		return nil, fmt.Errorf("error visiting category page: %w", err)
	}

	c.Wait()

	// If we found any listings, try to fetch more details for each
	if len(listings) > 0 {
		enrichedListings := make([]models.Listing, 0, len(listings))
		for i, listing := range listings {
			// Only fetch details if we have a URL
			if listing.URL != "" {
				log.Printf("Fetching details for listing %d of %d", i+1, len(listings))

				// Respect rate limiting for each detail request
				waitForRateLimit()

				// Fetch detailed information for this listing
				enriched, err := GetListingDetails(listing)
				if err != nil {
					log.Printf("Error fetching details for listing %s: %v", listing.ID, err)
					enrichedListings = append(enrichedListings, listing)
				} else {
					enrichedListings = append(enrichedListings, enriched)
				}
			} else {
				enrichedListings = append(enrichedListings, listing)
			}
		}
		return enrichedListings, nil
	}

	return listings, nil
}

// handleCatalogPage handles the special case of catalog pages
func handleCatalogPage(catalogURL string, limit int) ([]models.Listing, error) {
	log.Println("Handling catalog page:", catalogURL)
	var listings []models.Listing
	var itemURLs []string

	c := colly.NewCollector(
		colly.AllowedDomains("www.avito.ru", "avito.ru"),
		colly.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
		colly.MaxDepth(1),
	)

	// Set up retry mechanism
	c.SetRequestTimeout(30 * time.Second)

	// Rate limiting
	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		RandomDelay: 5 * time.Second,
		Delay:       3 * time.Second,
	})

	c.OnRequest(func(r *colly.Request) {
		log.Println("Visiting catalog:", r.URL)
		// Respect rate limiting
		waitForRateLimit()
	})

	c.OnError(func(r *colly.Response, err error) {
		log.Println("Error:", err)
		if r.StatusCode == 429 {
			log.Println("Rate limited, waiting longer before retry")
			time.Sleep(10 * time.Second)

			// Try to retry with a different user agent
			retries := 0
			for retries < maxRetries {
				retries++
				log.Printf("Retry %d of %d...", retries, maxRetries)
				time.Sleep(5 * time.Second * time.Duration(retries))

				// Alternate user agents
				userAgents := []string{
					"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
					"Mozilla/5.0 (iPhone; CPU iPhone OS 13_2_3 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/13.0.3 Mobile/15E148 Safari/604.1",
					"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/92.0.4515.107 Safari/537.36",
				}

				r.Request.Headers.Set("User-Agent", userAgents[retries%len(userAgents)])

				// Retry the request
				err = c.Request("GET", r.Request.URL.String(), nil, r.Request.Ctx, nil)
				if err == nil {
					return
				}

				log.Printf("Retry %d failed: %v", retries, err)
			}
		}
	})

	c.OnResponse(func(r *colly.Response) {
		log.Printf("Received catalog response, size: %d bytes\n", len(r.Body))
	})

	// Extract regular listings if any
	c.OnHTML("div.items-items, div.catalog-items", func(e *colly.HTMLElement) {
		log.Println("Found catalog items container")

		// Try multiple selectors for items
		itemSelectors := []string{
			"div[data-item-id]",
			"div.item-wrapper",
			"div.catalog-item",
			"div.item",
		}

		for _, selector := range itemSelectors {
			e.ForEach(selector, func(_ int, s *colly.HTMLElement) {
				if limit > 0 && len(itemURLs) >= limit {
					return
				}

				// Find the URL to the item
				href := s.ChildAttr("a[href]", "href")
				if href == "" {
					s.ForEach("a[href]", func(_ int, a *colly.HTMLElement) {
						if href == "" {
							h := a.Attr("href")
							if strings.Contains(h, "/item/") {
								href = h
							}
						}
					})
				}

				if href != "" {
					href = normalizeURL(href)
					itemURLs = append(itemURLs, href)
				}
			})

			if len(itemURLs) > 0 {
				log.Printf("Found %d item URLs using selector: %s\n", len(itemURLs), selector)
				break
			}
		}
	})

	// Extract catalog cards for models
	c.OnHTML("div.catalog-card, div.catalog-list-item, div.item-panel", func(e *colly.HTMLElement) {
		if limit > 0 && len(itemURLs) >= limit {
			return
		}

		href := e.ChildAttr("a[href]", "href")
		if href != "" {
			href = normalizeURL(href)
			itemURLs = append(itemURLs, href)
		}
	})

	// If no items found, look for any links that might be items or subcategories
	c.OnHTML("body", func(e *colly.HTMLElement) {
		if len(itemURLs) > 0 {
			return // Skip if we already found items
		}

		log.Println("Using fallback method for catalog page")

		// First priority: Find links that point to item pages
		e.DOM.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
			if limit > 0 && len(itemURLs) >= limit {
				return
			}

			href, _ := s.Attr("href")
			if strings.Contains(href, "/item/") {
				href = normalizeURL(href)
				itemURLs = append(itemURLs, href)
			}
		})

		// Second priority: Find links that might be subcategories
		if len(itemURLs) == 0 {
			e.DOM.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
				if limit > 0 && len(itemURLs) >= limit {
					return
				}

				href, _ := s.Attr("href")

				// Skip external links and already processed ones
				if !strings.HasPrefix(href, "/") && !strings.Contains(href, "avito.ru") {
					return
				}

				// Skip certain types of links
				if strings.Contains(href, "/favorites") ||
					strings.Contains(href, "/profile") ||
					strings.Contains(href, "/auth") ||
					strings.Contains(href, "/support") ||
					strings.Contains(href, "/stat") {
					return
				}

				// If we get here, this might be a subcategory or item
				href = normalizeURL(href)

				// Skip the current URL
				if href == catalogURL {
					return
				}

				log.Printf("Found potential subcategory or item: %s\n", href)
				itemURLs = append(itemURLs, href)
			})
		}

		log.Printf("Found %d potential items or subcategories with fallback method\n", len(itemURLs))
	})

	// Wait for rate limiting before starting
	waitForRateLimit()

	err := c.Visit(catalogURL)
	if err != nil {
		return nil, fmt.Errorf("error visiting catalog page: %w", err)
	}

	c.Wait()

	// Process found URLs (could be direct items or subcategories)
	if len(itemURLs) > 0 {
		log.Printf("Processing %d URLs from catalog\n", len(itemURLs))
		for i, url := range itemURLs {
			if limit > 0 && len(listings) >= limit {
				break
			}

			log.Printf("Processing catalog URL %d of %d: %s\n", i+1, len(itemURLs), url)

			// Respect rate limiting
			waitForRateLimit()

			// Check if this is an item URL or potentially a subcategory
			if strings.Contains(url, "/item/") {
				// This is an item URL
				listing := models.Listing{
					URL:         url,
					CategoryURL: catalogURL,
				}

				// Try to extract ID from URL
				matches := itemIDRegex.FindStringSubmatch(url)
				if len(matches) > 1 {
					if matches[1] != "" {
						listing.ID = matches[1]
					} else if matches[2] != "" {
						listing.ID = matches[2]
					}
				}

				// Fetch details for this listing
				enriched, err := GetListingDetails(listing)
				if err != nil {
					log.Printf("Error fetching details for URL %s: %v", url, err)
					if listing.ID != "" {
						listings = append(listings, listing)
					}
				} else {
					listings = append(listings, enriched)
				}
			} else {
				// This might be a subcategory or another type of page
				// Try to parse it as a category page to extract items
				subListings, err := GetListings(url, 1) // Only get 1 item from each potential subcategory
				if err != nil {
					log.Printf("Error processing potential subcategory %s: %v", url, err)
					continue
				}

				if len(subListings) > 0 {
					log.Printf("Found %d listings in subcategory %s\n", len(subListings), url)
					for _, listing := range subListings {
						if limit > 0 && len(listings) >= limit {
							break
						}
						listings = append(listings, listing)
					}
				}
			}

			// Add a delay between requests to be nice to the server
			time.Sleep(3 * time.Second)
		}
	}

	return listings, nil
}

// GetListingDetails fetches detailed information for a specific listing
func GetListingDetails(listing models.Listing) (models.Listing, error) {
	if listing.URL == "" {
		return listing, fmt.Errorf("listing URL is empty")
	}

	c := colly.NewCollector(
		colly.AllowedDomains("www.avito.ru", "avito.ru"),
		colly.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
		colly.MaxDepth(1),
	)

	// Set up retry mechanism
	c.SetRequestTimeout(30 * time.Second)

	c.OnRequest(func(r *colly.Request) {
		log.Println("Visiting listing page:", r.URL)
		// Respect rate limiting
		waitForRateLimit()
	})

	c.OnError(func(r *colly.Response, err error) {
		log.Println("Error visiting listing page:", err)
	})

	// Extract title if we don't have it
	if listing.Title == "" {
		c.OnHTML("h1", func(e *colly.HTMLElement) {
			listing.Title = strings.TrimSpace(e.Text)
		})
	}

	// Parse listing details
	c.OnHTML("body", func(e *colly.HTMLElement) {
		// Extract description
		description := e.DOM.Find("div[data-marker='item-description'], div.item-description").Text()
		listing.Description = strings.TrimSpace(description)

		// Extract images
		e.DOM.Find("div.gallery-img-wrapper img, div.photo-slider-image-wrapper img").Each(func(_ int, s *goquery.Selection) {
			if src, exists := s.Attr("src"); exists && src != "" {
				listing.ImageURLs = append(listing.ImageURLs, src)
			} else if srcset, exists := s.Attr("srcset"); exists && srcset != "" {
				// Take the first image from srcset
				parts := strings.Split(srcset, " ")
				if len(parts) > 0 {
					listing.ImageURLs = append(listing.ImageURLs, parts[0])
				}
			} else if dataSrc, exists := s.Attr("data-src"); exists && dataSrc != "" {
				listing.ImageURLs = append(listing.ImageURLs, dataSrc)
			}
		})

		// Extract location
		location := e.DOM.Find("div[data-marker='item-address'], div.item-address").Text()
		listing.Location = strings.TrimSpace(location)

		// Extract price if we don't have it
		if listing.Price.Value == 0 {
			priceText := e.DOM.Find("span.price-value, div.item-price, *[data-marker='item-price']").Text()
			if priceText != "" {
				listing.Price = parsePrice(priceText)
			}
		}

		// Extract publish date
		dateText := e.DOM.Find("div[data-marker='item-date'], div.item-date").Text()
		if dateText != "" {
			listing.PublishedAt = parseDate(dateText)
		}

		// Extract attributes
		attributes := make(map[string]string)
		e.DOM.Find("div.item-params, ul.item-params-list li").Each(func(_ int, s *goquery.Selection) {
			text := strings.TrimSpace(s.Text())
			if text != "" {
				parts := strings.Split(text, ":")
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0])
					value := strings.TrimSpace(parts[1])
					attributes[key] = value
				}
			}
		})

		// If any attributes were found, add them
		if len(attributes) > 0 {
			listing.Attributes = attributes
		}
	})

	// Wait for rate limiting before starting
	waitForRateLimit()

	err := c.Visit(listing.URL)
	if err != nil {
		return listing, fmt.Errorf("error visiting listing page: %w", err)
	}

	c.Wait()
	return listing, nil
}

// parseListing extracts listing information from an item card
func parseListing(item *colly.HTMLElement) models.Listing {
	listing := models.Listing{
		Attributes: make(map[string]string),
	}

	// Extract ID
	id := item.Attr("data-item-id")
	if id == "" {
		id = item.Attr("id")
	}

	if id == "" {
		// Try to extract ID from URLs or other attributes
		href := item.ChildAttr("a", "href")
		if href != "" {
			matches := itemIDRegex.FindStringSubmatch(href)
			if len(matches) > 1 {
				if matches[1] != "" {
					id = matches[1]
				} else if matches[2] != "" {
					id = matches[2]
				}
			}
		}
	}
	listing.ID = id

	// Extract title
	title := strings.TrimSpace(item.ChildText("h3.title, div.title, a.title, *[data-marker='item-title']"))
	if title == "" {
		// Try more general selectors
		title = strings.TrimSpace(item.DOM.Find("h3, h2, a.snippet-link").First().Text())
	}
	listing.Title = title

	// Extract URL
	url := item.ChildAttr("a[href]", "href")
	if url == "" {
		url = item.ChildAttr("a", "href")
	}
	if url == "" {
		item.DOM.Find("a").Each(func(_ int, s *goquery.Selection) {
			if href, exists := s.Attr("href"); exists {
				if strings.Contains(href, "/item/") {
					url = href
					return
				}
			}
		})
	}
	listing.URL = normalizeURL(url)

	// Extract price
	priceText := strings.TrimSpace(item.ChildText("span.price, div.price, *[data-marker='item-price']"))
	if priceText == "" {
		priceText = strings.TrimSpace(item.DOM.Find(".price, .snippet-price, .price-text").First().Text())
	}

	if priceText != "" {
		listing.Price = parsePrice(priceText)
	}

	// Extract location
	location := strings.TrimSpace(item.ChildText("div.geo-georeferences, *[data-marker='item-address']"))
	if location == "" {
		location = strings.TrimSpace(item.DOM.Find(".geo-georeferences, .item-address, .snippet-address").First().Text())
	}
	listing.Location = location

	// Extract image URL
	imageURL := item.ChildAttr("img", "src")
	if imageURL != "" {
		listing.ImageURLs = []string{imageURL}
	} else {
		// Try to find images with data-src attribute
		dataSrc := item.ChildAttr("img", "data-src")
		if dataSrc != "" {
			listing.ImageURLs = []string{dataSrc}
		}
	}

	return listing
}

// parsePrice extracts price information from text
func parsePrice(priceText string) models.Price {
	price := models.Price{
		Text: priceText,
	}

	// Default to RUB
	price.Currency = "RUB"

	// Check for currency symbols
	if strings.Contains(priceText, "$") {
		price.Currency = "USD"
	} else if strings.Contains(priceText, "€") {
		price.Currency = "EUR"
	}

	// Extract numeric value
	matches := priceRegex.FindString(priceText)
	if matches != "" {
		// Clean up the string
		valueStr := strings.ReplaceAll(matches, " ", "")
		valueStr = strings.ReplaceAll(valueStr, ",", ".")

		// Parse as float
		value, err := strconv.ParseFloat(valueStr, 64)
		if err == nil {
			price.Value = value
		}
	}

	return price
}

// parseDate attempts to parse a date string from Avito into a time.Time
func parseDate(dateStr string) time.Time {
	// Avito may use relative dates like "сегодня", "вчера" or specific dates
	dateStr = strings.ToLower(strings.TrimSpace(dateStr))

	now := time.Now()

	if strings.Contains(dateStr, "сегодня") {
		// Today
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	} else if strings.Contains(dateStr, "вчера") {
		// Yesterday
		return time.Date(now.Year(), now.Month(), now.Day()-1, 0, 0, 0, 0, now.Location())
	}

	// Try various date formats
	formats := []string{
		"02 января 2006",
		"02 января",
		"02 янв 2006",
		"02 янв",
		"02.01.2006",
		"02.01.06",
	}

	for _, format := range formats {
		t, err := time.Parse(format, dateStr)
		if err == nil {
			// If year is not specified, use current year
			if t.Year() == 0 {
				t = time.Date(now.Year(), t.Month(), t.Day(), 0, 0, 0, 0, now.Location())
			}
			return t
		}
	}

	// Default to current time if parsing fails
	return now
}

// ParseItemsFromHTML extracts advertisement items (title, URL, price) from HTML content
func ParseItemsFromHTML(htmlContent string) ([]models.Listing, error) {
	var listings []models.Listing

	// Create a goquery document from the HTML content
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("error parsing HTML: %w", err)
	}

	// Look for item containers using various selectors that might match Avito's structure
	var itemSelectors = []string{
		"div[data-marker='item']",
		"div[data-marker='item-card']",
		"div.iva-item-root",
		"div.styles-item-m0DD4",
		"div.js-item",
		"div.item",
		"div.item-card",
	}

	// Try each selector until we find items
	found := false
	for _, selector := range itemSelectors {
		items := doc.Find(selector)
		if items.Length() > 0 {
			log.Printf("Found %d items using selector: %s\n", items.Length(), selector)
			
			items.Each(func(i int, item *goquery.Selection) {
				listing := models.Listing{
					Attributes: make(map[string]string),
				}

				// Extract ID from data attribute or URL
				id, exists := item.Attr("data-item-id")
				if !exists {
					// Try to extract from href attribute
					itemURLNode := item.Find("a[href*='/item/']").First()
					if itemURLNode.Length() > 0 {
						href, exists := itemURLNode.Attr("href")
						if exists {
							matches := itemIDRegex.FindStringSubmatch(href)
							if len(matches) > 1 {
								if matches[1] != "" {
									id = matches[1]
								} else if matches[2] != "" {
									id = matches[2]
								}
							}
						}
					}
				}
				listing.ID = id

				// Extract title
				titleSelectors := []string{
					"h3[itemprop='name']",
					"*[data-marker='item-title']",
					"div.title",
					"h3.title",
					"a.title",
					"div.snippet-title",
				}
				
				for _, titleSelector := range titleSelectors {
					titleNode := item.Find(titleSelector).First()
					if titleNode.Length() > 0 {
						listing.Title = strings.TrimSpace(titleNode.Text())
						break
					}
				}

				// If no title found yet, look for links with text
				if listing.Title == "" {
					item.Find("a").Each(func(_ int, a *goquery.Selection) {
						if listing.Title == "" && strings.TrimSpace(a.Text()) != "" {
							href, exists := a.Attr("href")
							if exists && strings.Contains(href, "/item/") {
								listing.Title = strings.TrimSpace(a.Text())
							}
						}
					})
				}

				// Extract URL
				urlNode := item.Find("a[href*='/item/']").First()
				if urlNode.Length() > 0 {
					href, exists := urlNode.Attr("href")
					if exists {
						listing.URL = normalizeURL(href)
					}
				}

				// Extract price
				priceSelectors := []string{
					"*[data-marker='item-price']",
					"span.price-text-_YGDY",
					"span.price",
					"div.price",
					"span[itemprop='price']",
					"div.snippet-price",
				}
				
				for _, priceSelector := range priceSelectors {
					priceNode := item.Find(priceSelector).First()
					if priceNode.Length() > 0 {
						priceText := strings.TrimSpace(priceNode.Text())
						if priceText != "" {
							listing.Price = parsePrice(priceText)
							break
						}
					}
				}

				// Only add if we have at least a title or URL
				if listing.Title != "" || listing.URL != "" {
					listings = append(listings, listing)
				}
			})
			
			found = true
			break
		}
	}

	// If no items found with specific selectors, try a more general approach
	if !found || len(listings) == 0 {
		log.Println("No items found with specific selectors, trying fallback approach")
		
		// Look for any link that might be an item
		doc.Find("a[href]").Each(func(_ int, a *goquery.Selection) {
			href, exists := a.Attr("href")
			if exists && strings.Contains(href, "/item/") {
				title := strings.TrimSpace(a.Text())
				
				// If no text in the anchor itself, look for text in children
				if title == "" {
					title = strings.TrimSpace(a.Find("h3, div.title, span.title").First().Text())
				}

				// Skip if no title found
				if title == "" {
					return
				}

				listing := models.Listing{
					Title: title,
					URL:   normalizeURL(href),
				}

				// Extract ID from URL
				matches := itemIDRegex.FindStringSubmatch(href)
				if len(matches) > 1 {
					if matches[1] != "" {
						listing.ID = matches[1]
					} else if matches[2] != "" {
						listing.ID = matches[2]
					}
				}

				// Look for price near this element
				// Either a sibling or a child within the parent container
				parent := a.Parent()
				priceText := strings.TrimSpace(parent.Find("span.price, div.price, *[data-marker='item-price']").First().Text())
				if priceText != "" {
					listing.Price = parsePrice(priceText)
				}

				listings = append(listings, listing)
			}
		})
	}

	return listings, nil
}
