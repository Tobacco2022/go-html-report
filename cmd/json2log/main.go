package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// Event represents a single test event from go test -json output
type Event struct {
	Action  string  `json:"Action"`
	Test    string  `json:"Test"`
	Package string  `json:"Package"`
	Output  string  `json:"Output"`
	Elapsed float64 `json:"Elapsed"`
}

func main() {
	var writer io.Writer = os.Stdout

	// Check if output file is specified
	if len(os.Args) > 1 {
		outputFile := os.Args[1]
		file, err := os.Create(outputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
			os.Exit(1)
		}
		defer file.Close()
		writer = file
	}

	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		line := scanner.Bytes()
		var e Event

		if err := json.Unmarshal(line, &e); err != nil {
			// Skip lines that are not valid JSON
			continue
		}

		// Only output the "output" action, as it contains all formatted text
		// from go test including RUN, PASS, FAIL, SKIP, PAUSE, CONT messages
		if e.Action == "output" {
			fmt.Fprint(writer, e.Output)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		os.Exit(1)
	}
}
