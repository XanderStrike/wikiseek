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

// ExtractBzip2Range extracts a range of bytes from a bzip2 file and writes them to output.xml
func ExtractBzip2Range(filename string, startOffset, endOffset int64) error {
	// Validate parameters
	if startOffset < 0 {
		return fmt.Errorf("start offset must be non-negative")
	}
	if endOffset < 0 {
		return fmt.Errorf("end offset must be non-negative")
	}
	if endOffset > 0 && endOffset <= startOffset {
		return fmt.Errorf("end offset must be greater than start offset")
	}

	// Check if file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return fmt.Errorf("file '%s' does not exist", filename)
	}

	// Open the input file
	f, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("error opening file '%s': %v", filename, err)
	}
	defer f.Close()

	// Seek to the start offset in the compressed file
	_, err = f.Seek(startOffset, 0)
	if err != nil {
		return fmt.Errorf("error seeking to offset %d: %v", startOffset, err)
	}

	// Create a buffer to hold compressed bytes
	compressedData := make([]byte, endOffset-startOffset)

	// Read the compressed bytes into buffer
	bytesToRead := endOffset - startOffset
	fmt.Fprintf(os.Stderr, "Reading %d compressed bytes...\n", bytesToRead)

	n, err := io.ReadFull(f, compressedData)
	if err != nil && err != io.EOF {
		return fmt.Errorf("error reading compressed data: %v", err)
	}
	fmt.Fprintf(os.Stderr, "Read %d compressed bytes\n", n)

	// Create bzip2 reader for the compressed data
	bzReader := bzip2.NewReader(bytes.NewReader(compressedData))

	// Create output file
	outFile, err := os.Create("output.xml")
	if err != nil {
		return fmt.Errorf("error creating output file: %v", err)
	}
	defer outFile.Close()

	// Decompress the data to the output file
	written, err := io.Copy(outFile, bzReader)
	if err != nil {
		return fmt.Errorf("error decompressing data: %v", err)
	}

	fmt.Fprintf(os.Stderr, "Decompressed %d bytes\n", written)
	return nil
}

// ExtractPageText parses XML data and returns the text content for a given page ID
func ExtractPageText(filename string, pageID int) (string, error) {
	// Read the XML file
	data, err := os.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("error reading file: %v", err)
	}

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
}

func loadIndex(filename string) ([]IndexEntry, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	fmt.Printf("Loading index from %s...\n", filename)
	
	var entries []IndexEntry
	scanner := bufio.NewScanner(bytes.NewReader(data))
	var allEntries []IndexEntry
	lineCount := 0

	// First pass: collect all entries
	for scanner.Scan() {
		lineCount++
		if lineCount%100000 == 0 {
			fmt.Printf("Processed %d index entries...\n", lineCount)
		}
		parts := strings.Split(scanner.Text(), ":")
		if len(parts) != 3 {
			continue
		}

		startOffset, _ := strconv.ParseInt(parts[0], 10, 64)
		pageID, _ := strconv.Atoi(parts[1])

		allEntries = append(allEntries, IndexEntry{
			StartOffset: startOffset,
			PageID:      pageID,
			Title:       parts[2],
		})
	}

	fmt.Printf("Sorting %d index entries...\n", len(allEntries))
	sort.Slice(allEntries, func(i, j int) bool {
		return allEntries[i].StartOffset < allEntries[j].StartOffset
	})
	fmt.Printf("Calculating end offsets...\n")

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
		entries = append(entries, entry)
	}

	fmt.Printf("Index loaded with %d entries\n", len(entries))
	return entries, nil
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
		if entry.Title == searchTitle {
			return &entry
		}
	}
	return nil
}

func handlePage(w http.ResponseWriter, r *http.Request, inputFile string, tmpl *template.Template, index []IndexEntry) {
	// Extract the title from the URL path
	title := strings.TrimPrefix(r.URL.Path, "/page/")
	
	entry := findPageByTitle(index, title)
	if entry == nil {
		http.NotFound(w, r)
		return
	}

	data := PageData{}
	
	if err := ExtractBzip2Range(inputFile, entry.StartOffset, entry.EndOffset); err != nil {
		data.Error = err.Error()
	} else {
		text, err := ExtractPageText("output.xml", entry.PageID)
		if err != nil {
			data.Error = err.Error()
		} else {
			if err := os.WriteFile("page.mediawiki", []byte(text), 0644); err != nil {
				data.Error = err.Error()
			} else {
				output, err := exec.Command("pandoc", "-f", "mediawiki", "page.mediawiki").Output()
				if err != nil {
					data.Error = err.Error()
				} else {
					data.Content = template.HTML(output)
				}
			}
		}
	}

	tmpl.Execute(w, data)
}

func handleExtract(w http.ResponseWriter, r *http.Request, inputFile string, tmpl *template.Template, index []IndexEntry) {
	data := PageData{}

	// Handle search
	if query := r.FormValue("search"); query != "" {
		data.Query = query
		data.Results = searchIndex(index, query)
		tmpl.Execute(w, data)
		return
	}

	if r.Method == "POST" {
		startOffset, _ := strconv.ParseInt(r.FormValue("start"), 10, 64)
		endOffset, _ := strconv.ParseInt(r.FormValue("end"), 10, 64)
		pageID, _ := strconv.Atoi(r.FormValue("id"))

		if err := ExtractBzip2Range(inputFile, startOffset, endOffset); err != nil {
			data.Error = err.Error()
		} else {
			text, err := ExtractPageText("output.xml", pageID)
			if err != nil {
				data.Error = err.Error()
			} else {
				if err := os.WriteFile("page.mediawiki", []byte(text), 0644); err != nil {
					data.Error = err.Error()
				} else {
					output, err := exec.Command("pandoc", "-f", "mediawiki", "page.mediawiki").Output()
					if err != nil {
						data.Error = err.Error()
					} else {
						data.Content = template.HTML(output)
					}
				}
			}
		}
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

	tmpl, err := template.ParseFiles("templates/index.html")
	if err != nil {
		fmt.Printf("Error parsing template: %v\n", err)
		os.Exit(1)
	}

	http.HandleFunc("/page/", func(w http.ResponseWriter, r *http.Request) {
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
