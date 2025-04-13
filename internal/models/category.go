package models

// Category represents a category from Avito.ru
type Category struct {
	Name          string     `json:"name"`
	URL           string     `json:"url"`
	Subcategories []Category `json:"subcategories,omitempty"`
}
