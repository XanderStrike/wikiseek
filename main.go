package main

import (
	"compress/bzip2"
	"flag"
	"fmt"
	"io"
	"os"
)

type limitedReader struct {
	r        io.Reader
	limit    int64
	position int64
}

func (l *limitedReader) Read(p []byte) (n int, err error) {
	if l.position >= l.limit {
		return 0, io.EOF
	}
	if l.limit-l.position < int64(len(p)) {
		p = p[:l.limit-l.position]
	}
	n, err = l.r.Read(p)
	l.position += int64(n)
	return
}

type byteCounter struct {
	count int64
}

func (c *byteCounter) Write(p []byte) (n int, err error) {
	c.count += int64(len(p))
	return len(p), nil
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

	// Seek to start offset
	fmt.Fprintf(os.Stderr, "Seeking to offset %d...\n", startOffset)
	_, err = f.Seek(startOffset, 0)
	if err != nil {
		return fmt.Errorf("error seeking to offset %d: %v", startOffset, err)
	}

	// Create bzip2 reader
	bzReader := bzip2.NewReader(f)

	// Wrap with limited reader if end offset specified
	var reader io.Reader = bzReader
	if endOffset > 0 {
		reader = &limitedReader{
			r:        bzReader,
			limit:    endOffset - startOffset,
			position: 0,
		}
	}

	// Create output file
	outFile, err := os.Create("output.xml")
	if err != nil {
		return fmt.Errorf("error creating output file: %v", err)
	}
	defer outFile.Close()

	// Setup output writer and byte counter
	var output io.Writer = outFile
	counter := &byteCounter{count: 0}

	if endOffset > 0 {
		// If end offset specified, limit the number of bytes read
		output = io.MultiWriter(outFile, counter)
	}

	// Copy decompressed data to file
	fmt.Fprintf(os.Stderr, "Reading bzip2 data...\n")
	n, err := io.Copy(output, reader)
	if err != nil {
		return fmt.Errorf("error decompressing data: %v", err)
	}

	if endOffset > 0 && counter.count > (endOffset-startOffset) {
		fmt.Printf("Warning: Read more bytes (%d) than specified range (%d)\n",
			counter.count, endOffset-startOffset)
	}

	fmt.Fprintf(os.Stderr, "Decompressed %d bytes\n", n)
	return nil
}

func main() {
	// Parse command line flags
	inputFile := flag.String("file", "", "Path to multistream bzip2 file")
	startOffset := flag.Int64("start", 0, "Starting byte offset of bzip2 stream")
	endOffset := flag.Int64("end", 0, "Ending byte offset of bzip2 stream (0 means until EOF)")
	flag.Parse()

	if *inputFile == "" {
		fmt.Println("Error: -file argument is required")
		fmt.Println("Usage example: program -file input.bz2 -start 1024 -end 2048")
		flag.Usage()
		os.Exit(1)
	}

	if err := ExtractBzip2Range(*inputFile, *startOffset, *endOffset); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
