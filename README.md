# WikiSeek

WikiSeek is a Go-based web application that serves as a local Wikipedia browser and search engine. It allows you to host and browse Wikipedia content from a compressed Wikipedia XML dump file, providing fast search capabilities and article rendering.

## Features

- Browse Wikipedia articles with rendered HTML output
- Fast full-text search through article titles
- Random article suggestions
- Clean, responsive web interface
- Efficient handling of large compressed Wikipedia dumps
- Markdown-to-HTML conversion of Wikipedia markup

## Requirements

- Go 1.x
- pandoc (for converting Wikipedia markup to HTML)
- A Wikipedia XML dump file in bzip2 format
- The corresponding index file

## Installation

1. Install Go from https://golang.org/
2. Install pandoc: `brew install pandoc` (macOS) or `apt-get install pandoc` (Linux)
3. Clone this repository
4. Download a Wikipedia dump file and its index

## Usage

Run the server with:

```bash
go run main.go -file path/to/wiki.xml.bz2 -index path/to/index.bz2 -port 8080
```

Then visit http://localhost:8080 in your browser.

### Command Line Options

- `-file`: Path to the Wikipedia XML dump file (bzip2 compressed)
- `-index`: Path to the index file (bzip2 compressed)
- `-port`: Port to run the server on (default: 8080)

## Features

### Article Viewing
- Articles are rendered with full HTML formatting
- Internal links are preserved and clickable
- Clean typography and layout

### Search
- Fast title-based search
- Search results show article titles with direct links
- Case-insensitive matching

### Homepage
- Shows 10 random articles for discovery
- Search box for quick access
- Clean, minimal interface

## Technical Details

WikiSeek uses:
- Go's built-in HTTP server
- bzip2 compression handling
- XML parsing for Wikipedia dump format
- Pandoc for markup conversion
- HTML templating
- Static file serving

## License

This project is open source and available under the MIT License.
