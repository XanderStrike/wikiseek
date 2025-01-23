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
		flag.Usage()
		os.Exit(1)
	}

	// Open the input file
	f, err := os.Open(*inputFile)
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	// Skip to desired stream
	var currentStream int
	for currentStream < *streamIndex {
		// Create a new bzip2 reader
		bzReader := bzip2.NewReader(f)
		
		// Read and discard the current stream
		_, err := io.Copy(io.Discard, bzReader)
		if err != nil {
			fmt.Printf("Error skipping stream %d: %v\n", currentStream, err)
			os.Exit(1)
		}
		
		currentStream++
	}

	// Create bzip2 reader for target stream
	bzReader := bzip2.NewReader(f)

	// Copy decompressed data to stdout
	_, err = io.Copy(os.Stdout, bzReader)
	if err != nil {
		fmt.Printf("Error decompressing stream %d: %v\n", *streamIndex, err)
		os.Exit(1)
	}
}
