package main

import (
	"compress/bzip2"
	"flag"
	"fmt"
	"io"
	"os"
)



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

	// Create bzip2 reader
	bzReader := bzip2.NewReader(f)

	// Create output file
	outFile, err := os.Create("output.xml")
	if err != nil {
		return fmt.Errorf("error creating output file: %v", err)
	}
	defer outFile.Close()

	// Skip to start offset by reading and discarding bytes
	fmt.Fprintf(os.Stderr, "Skipping to offset %d...\n", startOffset)
	if startOffset > 0 {
		_, err = io.CopyN(io.Discard, bzReader, startOffset)
		if err != nil {
			return fmt.Errorf("error skipping to start offset: %v", err)
		}
	}

	// Read exactly (endOffset - startOffset) bytes
	bytesToRead := endOffset - startOffset
	fmt.Fprintf(os.Stderr, "Reading %d bytes...\n", bytesToRead)
	
	n, err := io.CopyN(outFile, bzReader, bytesToRead)
	if err != nil && err != io.EOF {
		return fmt.Errorf("error reading data: %v", err)
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
