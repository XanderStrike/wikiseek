package main

import (
	"bufio"
	"bytes"
	"compress/bzip2"
	"encoding/xml"
	"flag"
	"fmt"
	"html/template"
	"io"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Page represents a Wikipedia page XML structure
type Page struct {
	Title    string   `xml:"title"`
	ID       int      `xml:"id"`
	Revision Revision `xml:"revision"`
}

type Revision struct {
	Text string `xml:"text"`
}

// ExtractBzip2Range extracts a range of bytes from a bzip2 file and returns the decompressed data
func ExtractBzip2Range(filename string, startOffset, endOffset int64) ([]byte, error) {
	// Validate parameters
	if startOffset < 0 {
		return nil, fmt.Errorf("start offset must be non-negative")
	}
	if endOffset < 0 {
		return nil, fmt.Errorf("end offset must be non-negative")
	}
	if endOffset > 0 && endOffset <= startOffset {
		return nil, fmt.Errorf("end offset must be greater than start offset")
	}

	// Check if file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return nil, fmt.Errorf("file '%s' does not exist", filename)
	}

	// Open the input file
	f, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("error opening file '%s': %v", filename, err)
	}
	defer f.Close()

	// Seek to the start offset in the compressed file
	_, err = f.Seek(startOffset, 0)
	if err != nil {
		return nil, fmt.Errorf("error seeking to offset %d: %v", startOffset, err)
	}

	// Create a buffer to hold compressed bytes
	compressedData := make([]byte, endOffset-startOffset)

	// Read the compressed bytes into buffer
	bytesToRead := endOffset - startOffset
	fmt.Fprintf(os.Stderr, "Reading %d compressed bytes...\n", bytesToRead)

	n, err := io.ReadFull(f, compressedData)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("error reading compressed data: %v", err)
	}
	fmt.Fprintf(os.Stderr, "Read %d compressed bytes\n", n)

	// Create bzip2 reader for the compressed data
	bzReader := bzip2.NewReader(bytes.NewReader(compressedData))

	// Decompress the data to a buffer
	var buf bytes.Buffer
	written, err := io.Copy(&buf, bzReader)
	if err != nil {
		return nil, fmt.Errorf("error decompressing data: %v", err)
	}

	fmt.Fprintf(os.Stderr, "Decompressed %d bytes\n", written)
	return buf.Bytes(), nil
}

// ExtractPageText parses XML data and returns the text content for a given page ID
func ExtractPageText(data []byte, pageID int) (string, error) {

	decoder := xml.NewDecoder(bytes.NewReader(data))
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("error decoding XML: %v", err)
		}

		if se, ok := token.(xml.StartElement); ok {
			if se.Name.Local == "page" {
				var page Page
				if err := decoder.DecodeElement(&page, &se); err != nil {
					return "", fmt.Errorf("error decoding page: %v", err)
				}
				if page.ID == pageID {
					return page.Revision.Text, nil
				}
			}
		}
	}
	return "", fmt.Errorf("page with ID %d not found", pageID)
}

type IndexEntry struct {
	StartOffset int64
	EndOffset   int64
	PageID      int
	Title       string
}

type PageData struct {
	Error   string
	Content template.HTML
	Query   string
	Results []IndexEntry
	Title   string
}

func loadIndex(filename string) ([]IndexEntry, error) {
	fmt.Printf("Loading compressed index from %s...\n", filename)

	// Open the bzip2 file
	f, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("error opening index file: %v", err)
	}
	defer f.Close()

	// Create bzip2 reader
	bzReader := bzip2.NewReader(f)

	// Pre-allocate slice with estimated size (adjust based on your data)
	allEntries := make([]IndexEntry, 0, 6000000)
	scanner := bufio.NewScanner(bzReader)

	// First pass: collect all entries
	for scanner.Scan() {
		line := scanner.Text()
		
		// Fast string splitting
		offsetStr, rest, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		pageIDStr, title, ok := strings.Cut(rest, ":")
		if !ok {
			continue
		}

		startOffset, _ := strconv.ParseInt(offsetStr, 10, 64)
		pageID, _ := strconv.Atoi(pageIDStr)

		allEntries = append(allEntries, IndexEntry{
			StartOffset: startOffset,
			PageID:      pageID,
			Title:       title,
		})
	}

	sort.Slice(allEntries, func(i, j int) bool {
		return allEntries[i].StartOffset < allEntries[j].StartOffset
	})

	// Second pass: calculate EndOffsets
	for i := 0; i < len(allEntries); i++ {
		entry := allEntries[i]
		// Find next higher start offset
		var nextOffset int64
		for j := i + 1; j < len(allEntries); j++ {
			if allEntries[j].StartOffset > entry.StartOffset {
				nextOffset = allEntries[j].StartOffset
				break
			}
		}
		entry.EndOffset = nextOffset
		allEntries[i] = entry
	}

	fmt.Printf("Index loaded with %d entries\n", len(allEntries))
	return allEntries, nil
}

func searchIndex(entries []IndexEntry, query string) []IndexEntry {
	query = strings.ToLower(query)
	var results []IndexEntry
	for _, entry := range entries {
		if strings.Contains(strings.ToLower(entry.Title), query) {
			results = append(results, entry)
		}
	}
	return results
}

func findPageByTitle(entries []IndexEntry, title string) *IndexEntry {
	// Convert underscores to spaces in the requested title
	searchTitle := strings.ReplaceAll(title, "_", " ")

	for _, entry := range entries {
		if strings.EqualFold(entry.Title, searchTitle) {
			return &entry
		}
	}
	return nil
}

func handlePage(w http.ResponseWriter, r *http.Request, inputFile string, tmpl *template.Template, index []IndexEntry) {
	// Extract the title from the URL path
	title := strings.TrimPrefix(r.URL.Path, "/wiki/")

	entry := findPageByTitle(index, title)
	if entry == nil {
		http.NotFound(w, r)
		return
	}

	data := PageData{
		Title: entry.Title,
	}

	xmlData, err := ExtractBzip2Range(inputFile, entry.StartOffset, entry.EndOffset)
	if err != nil {
		data.Error = err.Error()
	} else {
		text, err := ExtractPageText(xmlData, entry.PageID)
		if err != nil {
			data.Error = err.Error()
		} else {
			cmd := exec.Command("pandoc", "-f", "mediawiki", "-t", "html")
			stdin, err := cmd.StdinPipe()
			if err != nil {
				data.Error = err.Error()
				return
			}
			go func() {
				defer stdin.Close()
				io.WriteString(stdin, text)
			}()
			output, err := cmd.Output()
			if err != nil {
				data.Error = err.Error()
			} else {
				data.Content = template.HTML(output)
			}
		}
	}

	tmpl.Execute(w, data)
}

func handleSearch(w http.ResponseWriter, r *http.Request, searchTmpl *template.Template, index []IndexEntry) {
	data := PageData{}

	if query := r.FormValue("q"); query != "" {
		data.Query = query
		data.Results = searchIndex(index, query)
	}

	searchTmpl.Execute(w, data)
}

func getRandomEntries(entries []IndexEntry, count int) []IndexEntry {
	if len(entries) <= count {
		return entries
	}
	
	// Create a copy of indices to shuffle
	indices := make([]int, len(entries))
	for i := range indices {
		indices[i] = i
	}
	
	// Fisher-Yates shuffle
	for i := len(indices) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		indices[i], indices[j] = indices[j], indices[i]
	}
	
	// Take first count entries
	result := make([]IndexEntry, count)
	for i := 0; i < count; i++ {
		result[i] = entries[indices[i]]
	}
	return result
}

func handleExtract(w http.ResponseWriter, r *http.Request, inputFile string, tmpl *template.Template, index []IndexEntry) {
	data := PageData{
		RandomPages: getRandomEntries(index, 10), // Show 10 random pages
	}
	tmpl.Execute(w, data)
}

func main() {
	rand.Seed(time.Now().UnixNano())

	inputFile := flag.String("file", "", "Path to multistream bzip2 file")
	indexFile := flag.String("index", "", "Path to index file")
	port := flag.String("port", "8080", "Port to run the server on")
	flag.Parse()

	if *inputFile == "" || *indexFile == "" {
		fmt.Println("Error: both -file and -index arguments are required")
		flag.Usage()
		os.Exit(1)
	}

	index, err := loadIndex(*indexFile)
	if err != nil {
		fmt.Printf("Error loading index: %v\n", err)
		os.Exit(1)
	}

	funcMap := template.FuncMap{
		"urlize": func(s string) string {
			return strings.ReplaceAll(s, " ", "_")
		},
	}
	tmpl, err := template.New("index.html").Funcs(funcMap).ParseFiles("templates/index.html")
	if err != nil {
		fmt.Printf("Error parsing template: %v\n", err)
		os.Exit(1)
	}

	searchTmpl, err := template.New("search.html").Funcs(funcMap).ParseFiles("templates/search.html")
	if err != nil {
		fmt.Printf("Error parsing template: %v\n", err)
		os.Exit(1)
	}

	// Serve static files
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	http.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		handleSearch(w, r, searchTmpl, index)
	})

	http.HandleFunc("/wiki/", func(w http.ResponseWriter, r *http.Request) {
		handlePage(w, r, *inputFile, tmpl, index)
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		handleExtract(w, r, *inputFile, tmpl, index)
	})

	fmt.Printf("Server starting on http://localhost:%s\n", *port)
	if err := http.ListenAndServe(":"+*port, nil); err != nil {
		fmt.Printf("Server error: %v\n", err)
		os.Exit(1)
	}
}
