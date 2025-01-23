package main

import (
	"compress/bzip2"
	"flag"
	"fmt"
	"io"
	"os"
)

func main() {
	// Parse command line flags
	inputFile := flag.String("file", "", "Path to multistream bzip2 file")
	streamIndex := flag.Int("stream", 0, "Index of bzip2 stream to extract (0-based)")
	flag.Parse()

	if *inputFile == "" {
		fmt.Println("Error: -file argument is required")
		fmt.Println("Usage example: program -file input.bz2 -stream 0")
		flag.Usage()
		os.Exit(1)
	}

	// Check if file exists
	if _, err := os.Stat(*inputFile); os.IsNotExist(err) {
		fmt.Printf("Error: file '%s' does not exist\n", *inputFile)
		os.Exit(1)
	}

	// Open the input file
	f, err := os.Open(*inputFile)
	if err != nil {
		fmt.Printf("Error opening file '%s': %v\n", *inputFile, err)
		os.Exit(1)
	}
	defer f.Close()

	// Skip to desired stream
	var currentStream int
	fmt.Fprintf(os.Stderr, "Seeking to stream %d...\n", *streamIndex)
	
	for currentStream < *streamIndex {
		fmt.Fprintf(os.Stderr, "Skipping stream %d...\n", currentStream)
		
		// Create a new bzip2 reader
		bzReader := bzip2.NewReader(f)
		
		// Read and discard the current stream
		n, err := io.Copy(io.Discard, bzReader)
		if err != nil {
			fmt.Printf("Error skipping stream %d: %v\n", currentStream, err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Skipped %d bytes in stream %d\n", n, currentStream)
		
		currentStream++
	}

	fmt.Fprintf(os.Stderr, "Reading target stream %d...\n", *streamIndex)
	
	// Create bzip2 reader for target stream
	bzReader := bzip2.NewReader(f)

	// Copy decompressed data to stdout
	n, err := io.Copy(os.Stdout, bzReader)
	fmt.Fprintf(os.Stderr, "Decompressed %d bytes from stream %d\n", n, *streamIndex)
	if err != nil {
		fmt.Printf("Error decompressing stream %d: %v\n", *streamIndex, err)
		os.Exit(1)
	}
}
