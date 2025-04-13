package parser

import (
	"fmt"
	"log"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
	"github.com/itcaat/avitolog/internal/models"
)

const (
	baseURL = "https://www.avito.ru"
)

// GetCategories fetches all main categories and their subcategories from Avito.ru
func GetCategories() ([]models.Category, error) {
	var categories []models.Category
	var categoryMap = make(map[string]models.Category) // Use a map to avoid duplicates

	c := colly.NewCollector(
		colly.AllowedDomains("www.avito.ru", "avito.ru"),
		colly.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
	)

	// Add debugging callbacks
	c.OnRequest(func(r *colly.Request) {
		log.Println("Visiting", r.URL)
	})

	c.OnError(func(_ *colly.Response, err error) {
		log.Println("Error:", err)
	})

	c.OnResponse(func(r *colly.Response) {
		log.Printf("Received response from Avito.ru, size: %d bytes\n", len(r.Body))
	})

	// Find categories from visual rubricator grid
	c.OnHTML("div.visual-rubricator-grid-s6aQm a.visual-rubricator-gridItem-MiBU_", func(e *colly.HTMLElement) {
		catName := cleanText(e.DOM.Find("p").Text())
		href := e.Attr("href")
		if href != "" && catName != "" {
			url := normalizeURL(href)
			// Create or update the category
			if existingCat, found := categoryMap[url]; found {
				// Already exists, just ensure name is filled
				if existingCat.Name == "" {
					existingCat.Name = catName
					categoryMap[url] = existingCat
				}
			} else {
				categoryMap[url] = models.Category{
					Name:          catName,
					URL:           url,
					Subcategories: []models.Category{},
				}
			}
			log.Printf("Found category in visual grid: %s (%s)\n", catName, url)
		}
	})

	// Find categories from dropdown catalog menu
	c.OnHTML("div.index-module-nav-catalogs-_9ZX2 div.index-module-nav-catalog-item-a9Xx9 a", func(e *colly.HTMLElement) {
		catName := cleanText(e.Text)
		href := e.Attr("href")
		if href != "" && catName != "" {
			url := normalizeURL(href)
			categoryMap[url] = models.Category{
				Name:          catName,
				URL:           url,
				Subcategories: []models.Category{},
			}
			log.Printf("Found category in dropdown menu: %s (%s)\n", catName, url)
		}
	})

	// Find categories in mini-menu
	c.OnHTML("div.top-rubricator-hide-PSmtS a", func(e *colly.HTMLElement) {
		catName := cleanText(e.Text)
		href := e.Attr("href")
		if href != "" && catName != "" {
			url := normalizeURL(href)
			// Skip if already exists
			if _, found := categoryMap[url]; !found {
				categoryMap[url] = models.Category{
					Name:          catName,
					URL:           url,
					Subcategories: []models.Category{},
				}
				log.Printf("Found category in mini-menu: %s (%s)\n", catName, url)
			}
		}
	})

	// Find more categories from top navigation
	c.OnHTML("ul.index-module-nav-stRnY li.index-module-nav-item-queVi a", func(e *colly.HTMLElement) {
		catName := cleanText(e.Text)
		href := e.Attr("href")
		if strings.HasPrefix(href, "/") && catName != "" {
			url := normalizeURL(href)
			// Skip if already exists
			if _, found := categoryMap[url]; !found {
				categoryMap[url] = models.Category{
					Name:          catName,
					URL:           url,
					Subcategories: []models.Category{},
				}
				log.Printf("Found category in top nav: %s (%s)\n", catName, url)
			}
		}
	})

	// Find service links in mini-menu (may contain additional categories)
	c.OnHTML("div.service-item-QPvjs", func(e *colly.HTMLElement) {
		parent := e.DOM.Parent()
		if href, exists := parent.Attr("href"); exists {
			catName := cleanText(e.DOM.Find("span").Text())
			if href != "" && catName != "" {
				url := normalizeURL(href)
				// Skip if already exists
				if _, found := categoryMap[url]; !found {
					categoryMap[url] = models.Category{
						Name:          catName,
						URL:           url,
						Subcategories: []models.Category{},
					}
					log.Printf("Found category in services menu: %s (%s)\n", catName, url)
				}
			}
		}
	})

	// Start scraping
	err := c.Visit(baseURL)
	if err != nil {
		return nil, fmt.Errorf("error visiting %s: %w", baseURL, err)
	}

	c.Wait()

	// If we found any categories, visit each to find subcategories
	if len(categoryMap) > 0 {
		// Convert map to slice
		for _, cat := range categoryMap {
			categories = append(categories, cat)
		}

		// Process each category to find subcategories
		for i := range categories {
			if strings.Contains(categories[i].URL, "/all/") {
				subcats, err := getSubcategories(categories[i].URL)
				if err != nil {
					log.Printf("Error getting subcategories for %s: %v", categories[i].URL, err)
					continue
				}
				categories[i].Subcategories = subcats
			}
		}
	}

	return categories, nil
}

// getSubcategories fetches subcategories from a category page
func getSubcategories(categoryURL string) ([]models.Category, error) {
	var subcategories []models.Category

	c := colly.NewCollector(
		colly.AllowedDomains("www.avito.ru", "avito.ru"),
		colly.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
		colly.MaxDepth(1),
	)

	// Look for subcategory links in sidebar
	c.OnHTML("div[data-marker='category-map']", func(e *colly.HTMLElement) {
		log.Println("Found category map")

		// Find all subcategory links
		e.DOM.Find("a").Each(func(_ int, s *goquery.Selection) {
			name := cleanText(s.Text())
			href, exists := s.Attr("href")
			if exists && name != "" {
				subcategories = append(subcategories, models.Category{
					Name: name,
					URL:  normalizeURL(href),
				})
				log.Printf("Found subcategory: %s (%s)\n", name, href)
			}
		})
	})

	// Look for alternative subcategory structure
	c.OnHTML("ul.rubricator-list", func(e *colly.HTMLElement) {
		log.Println("Found rubricator list")

		e.DOM.Find("li a").Each(func(_ int, s *goquery.Selection) {
			name := cleanText(s.Text())
			href, exists := s.Attr("href")
			if exists && name != "" {
				// Don't add duplicates
				isDuplicate := false
				for _, existingSub := range subcategories {
					if existingSub.URL == normalizeURL(href) {
						isDuplicate = true
						break
					}
				}

				if !isDuplicate {
					subcategories = append(subcategories, models.Category{
						Name: name,
						URL:  normalizeURL(href),
					})
					log.Printf("Found subcategory in rubricator: %s (%s)\n", name, href)
				}
			}
		})
	})

	// Visit the category page
	err := c.Visit(categoryURL)
	if err != nil {
		return nil, fmt.Errorf("error visiting category page: %w", err)
	}

	c.Wait()
	return subcategories, nil
}

// cleanText removes extra whitespace and normalizes text
func cleanText(text string) string {
	// Replace newlines and multiple spaces with single spaces
	cleaned := strings.ReplaceAll(text, "\n", " ")
	cleaned = strings.Join(strings.Fields(cleaned), " ")
	return strings.TrimSpace(cleaned)
}

// normalizeURL ensures the URL is absolute
func normalizeURL(href string) string {
	if strings.HasPrefix(href, "http") {
		return href
	}

	if strings.HasPrefix(href, "//") {
		return "https:" + href
	}

	if strings.HasPrefix(href, "/") {
		return baseURL + href
	}

	// Try to parse the URL to handle other cases
	parsedURL, err := url.Parse(href)
	if err != nil {
		return baseURL + "/" + href
	}

	// If parsed successfully but is relative
	if !parsedURL.IsAbs() {
		return baseURL + "/" + href
	}

	return href
}
