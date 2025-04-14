package main

import (
	"fmt"
	"log"

	"github.com/itcaat/avitolog/internal/parser"
)

func main() {
	fmt.Println("Starting Avitolog parser...")

	// Get categories from Avito
	categories, err := parser.GetCategories()
	if err != nil {
		log.Fatalf("Error getting categories: %v", err)
	}

	// Display found categories
	fmt.Printf("Found %d main categories\n", len(categories))
	for i, category := range categories {
		fmt.Printf("\n%d. %s (%s)\n", i+1, category.Name, category.URL)

		// Limit the number of listings to fetch per category
		listingsLimit := 5

		// Fetch listings for this category
		fmt.Printf("   Fetching listings for %s...\n", category.Name)
		listings, err := parser.GetListings(category.URL, listingsLimit)
		if err != nil {
			log.Printf("   Error fetching listings for %s: %v", category.Name, err)
			continue
		}

		// Display the listings
		fmt.Printf("   Found %d listings\n", len(listings))
		for j, listing := range listings {
			fmt.Printf("   %d.%d. %s\n", i+1, j+1, listing.Title)
			fmt.Printf("      URL: %s\n", listing.URL)

			// Print price info if available
			if listing.Price.Value > 0 {
				fmt.Printf("      Price: %.2f %s\n", listing.Price.Value, listing.Price.Currency)
			} else if listing.Price.Text != "" {
				fmt.Printf("      Price: %s\n", listing.Price.Text)
			}

			// Print location if available
			if listing.Location != "" {
				fmt.Printf("      Location: %s\n", listing.Location)
			}
		}

		// Check if the category has subcategories
		if len(category.Subcategories) > 0 {
			fmt.Printf("\n   Subcategories for %s:\n", category.Name)

			// For each subcategory, fetch a smaller number of listings
			subListingsLimit := 2

			for k, subcategory := range category.Subcategories {
				fmt.Printf("   %d.%d. %s (%s)\n", i+1, k+1, subcategory.Name, subcategory.URL)

				// Fetch listings for this subcategory
				fmt.Printf("      Fetching listings for %s...\n", subcategory.Name)
				subListings, err := parser.GetListings(subcategory.URL, subListingsLimit)
				if err != nil {
					log.Printf("      Error fetching listings for %s: %v", subcategory.Name, err)
					continue
				}

				// Display the listings
				fmt.Printf("      Found %d listings\n", len(subListings))
				for l, subListing := range subListings {
					fmt.Printf("      %d.%d.%d. %s\n", i+1, k+1, l+1, subListing.Title)
					fmt.Printf("         URL: %s\n", subListing.URL)

					// Print price info if available
					if subListing.Price.Value > 0 {
						fmt.Printf("         Price: %.2f %s\n", subListing.Price.Value, subListing.Price.Currency)
					} else if subListing.Price.Text != "" {
						fmt.Printf("         Price: %s\n", subListing.Price.Text)
					}
				}
			}
		}

		fmt.Println("\n-------------------------------------------")
	}
}
