package main

import (
	"fmt"
	"log"

	"github.com/itcaat/avitolog/internal/parser"
)

func main() {
	fmt.Println("Starting Avito.ru parser...")

	categories, err := parser.GetCategories()
	if err != nil {
		log.Fatalf("Error fetching categories: %v", err)
	}

	fmt.Printf("\n====== AVITO CATEGORY STRUCTURE ======\n\n")
	fmt.Printf("Found %d main categories\n\n", len(categories))

	// Print categories in a structured format
	for i, cat := range categories {
		fmt.Printf("%d. %s\n", i+1, cat.Name)
		fmt.Printf("   URL: %s\n", cat.URL)

		if len(cat.Subcategories) > 0 {
			fmt.Printf("   Subcategories (%d):\n", len(cat.Subcategories))
			for j, subcat := range cat.Subcategories {
				fmt.Printf("     %d.%d %s\n", i+1, j+1, subcat.Name)
				fmt.Printf("         URL: %s\n", subcat.URL)
			}
		} else {
			fmt.Println("   No subcategories found")
		}
		fmt.Println() // Add empty line between categories
	}

	fmt.Println("Parsing completed!")
}
