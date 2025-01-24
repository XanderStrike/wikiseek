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

func ExtractBzip2Range(filename string, startOffset, endOffset int64) ([]byte, error) {
	if startOffset < 0 || endOffset < 0 || (endOffset > 0 && endOffset <= startOffset) {
		return nil, fmt.Errorf("invalid offset values")
	}

	f, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("opening file: %v", err)
	}
	defer f.Close()

	if _, err := f.Seek(startOffset, 0); err != nil {
		return nil, fmt.Errorf("seeking to offset: %v", err)
	}

	compressedData := make([]byte, endOffset-startOffset)
	if _, err := io.ReadFull(f, compressedData); err != nil && err != io.EOF {
		return nil, fmt.Errorf("reading compressed data: %v", err)
	}

	bzReader := bzip2.NewReader(bytes.NewReader(compressedData))
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, bzReader); err != nil {
		return nil, fmt.Errorf("decompressing data: %v", err)
	}

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

// OffsetPair stores unique start/end offset combinations
type OffsetPair struct {
	Start int64
	End   int64
}

type IndexEntry struct {
	Offsets *OffsetPair // Reference to shared offset pair
	PageID  int
	Title   string
}

// offsetCache helps deduplicate common offset pairs
type offsetCache struct {
	pairs map[uint64]*OffsetPair
}

func newOffsetCache() *offsetCache {
	return &offsetCache{
		pairs: make(map[uint64]*OffsetPair),
	}
}

func (oc *offsetCache) getOrCreate(start, end int64) *OffsetPair {
	// Create a unique key from the two int64s
	key := uint64(start)<<32 | uint64(uint32(end))
	
	if pair, ok := oc.pairs[key]; ok {
		return pair
	}
	
	pair := &OffsetPair{Start: start, End: end}
	oc.pairs[key] = pair
	return pair
}

type PageData struct {
	Error       string
	Content     template.HTML
	Query       string
	Results     []IndexEntry
	Title       string
	RandomPages []IndexEntry
}

func loadIndex(filename string) ([]IndexEntry, error) {
	fmt.Print("Loading index")
	f, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("opening index file: %v", err)
	}
	defer f.Close()

	bzReader := bzip2.NewReader(f)
	allEntries := make([]IndexEntry, 0, 6000000)
	offsets := newOffsetCache()
	scanner := bufio.NewScanner(bzReader)
	count := 0
	for scanner.Scan() {
		count++
		if count%1000000 == 0 {
			fmt.Print(".")
		}
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

		// Skip special namespace entries
		if strings.HasPrefix(title, "File:") ||
			strings.HasPrefix(title, "Category:") ||
			strings.HasPrefix(title, "Wikipedia:") ||
			strings.HasPrefix(title, "Draft:") ||
			strings.HasPrefix(title, "Portal:") ||
			strings.HasPrefix(title, "Template:") {
			continue
		}

		startOffset, _ := strconv.ParseInt(offsetStr, 10, 64)
		pageID, _ := strconv.Atoi(pageIDStr)

		allEntries = append(allEntries, IndexEntry{
			Offsets: offsets.getOrCreate(startOffset, 0), // EndOffset will be set later
			PageID:  pageID,
			Title:   title,
		})
	}

	sort.Slice(allEntries, func(i, j int) bool {
		return allEntries[i].Offsets.Start < allEntries[j].Offsets.Start
	})

	// Second pass: calculate EndOffsets
	for i := 0; i < len(allEntries); i++ {
		entry := &allEntries[i]
		// Find next higher start offset
		var nextOffset int64
		for j := i + 1; j < len(allEntries); j++ {
			if allEntries[j].Offsets.Start > entry.Offsets.Start {
				nextOffset = allEntries[j].Offsets.Start
				break
			}
		}
		// Update the offset pair
		entry.Offsets = offsets.getOrCreate(entry.Offsets.Start, nextOffset)
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

	// Try case sensitive match first
	for _, entry := range entries {
		if entry.Title == searchTitle {
			return &entry
		}
	}

	// Fall back to case insensitive match
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

	xmlData, err := ExtractBzip2Range(inputFile, entry.Offsets.Start, entry.Offsets.End)
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

	// Pick random indices directly
	result := make([]IndexEntry, count)
	seen := make(map[int]bool, count)

	for i := 0; i < count; {
		idx := rand.Intn(len(entries))
		if !seen[idx] {
			result[i] = entries[idx]
			seen[idx] = true
			i++
		}
	}

	return result
}

func handleExtract(w http.ResponseWriter, r *http.Request, inputFile string, tmpl *template.Template, index []IndexEntry) {
	data := PageData{
		RandomPages: getRandomEntries(index, 25),
	}
	tmpl.Execute(w, data)
}

func main() {
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
