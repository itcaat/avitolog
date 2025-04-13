# Avitolog - Avito.ru Parser

Avitolog is a Go application for parsing product listings from Avito.ru, a popular Russian classified ads website.

## Features

- Fetches main categories from Avito.ru
- Parses listings within each category
- Detailed information extraction from individual listings
- Saves data in JSON format
- Rate limiting to be respectful to the website

## Usage

### Building the application

```bash
go build -o avitolog ./cmd/avitolog
```

### Running the application

#### Fetch categories only

```bash
./avitolog
```

This will fetch main categories from Avito.ru and save them to `categories.json`.

#### Fetch categories and listings

```bash
./avitolog -listings
```

This will fetch categories and up to 10 listings per category, saving them in a structured directory format.

### Command line options

- `-listings`: Fetch listings in addition to categories
- `-limit N`: Limit the number of listings per category (default: 10, use 0 for no limit)
- `-output DIR`: Directory to save output files (default: current directory)
- `-categories FILE`: Path to categories JSON file (default: categories.json)

Examples:

```bash
# Fetch up to 50 listings per category and save to data directory
./avitolog -listings -limit 50 -output ./data

# Use existing categories file and fetch listings
./avitolog -listings -categories ./my-categories.json
```

## Output Structure

```
├── categories.json
├── Category_1
│   ├── listings.json
│   └── Subcategory_1
│       └── listings.json
└── Category_2
    ├── listings.json
    └── Subcategory_2
        └── listings.json
```

## License

MIT

## Disclaimer

This tool is for educational purposes only. Always comply with Avito.ru's terms of service when using this tool.