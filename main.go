package main

import (
	"bytes"
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
