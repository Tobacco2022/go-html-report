package main

import (
	"bufio"
	"encoding/json"
	"io"
	"strings"
	"time"
)

// Result represents a test result.
type Result int

// Test result constants
const (
	PASS Result = iota
	FAIL
	SKIP
)

// Report is a collection of package tests.
type Report struct {
	Packages []Package
}

// Package contains the test results of a single package.
type Package struct {
	Name        string
	Duration    time.Duration
	Tests       []*Test
	Benchmarks  []*Benchmark
	CoveragePct string

	// Time is deprecated, use Duration instead.
	Time int // in milliseconds
}

// Test contains the results of a single test.
type Test struct {
	Name     string
	Duration time.Duration
	Result   Result
	Output   []string

	SubtestIndent string

	// Time is deprecated, use Duration instead.
	Time int // in milliseconds
}

// Benchmark contains the results of a single benchmark.
type Benchmark struct {
	Name     string
	Duration time.Duration
	// number of B/op
	Bytes int
	// number of allocs/op
	Allocs int
}

// TestEvent represents a single test event from JSON output
type TestEvent struct {
	Time    string  `json:"Time"`
	Action  string  `json:"Action"`
	Package string  `json:"Package"`
	Test    string  `json:"Test,omitempty"`
	Output  string  `json:"Output,omitempty"`
	Elapsed float64 `json:"Elapsed,omitempty"`
}

var report = &Report{make([]Package, 0)}

// Parse parses go test JSON output from reader r and returns a report with the
// results. An optional pkgName can be given, which is used in case a package
// result line is missing.
func Parse(r io.Reader, pkgName string) error {
	scanner := bufio.NewScanner(r)

	// Map to track tests by their full name (including package)
	testMap := make(map[string]*Test)

	// Map to track packages
	packageMap := make(map[string]*Package)

	// Track test order within packages
	testOrder := make(map[string][]string) // package -> []testName

	for scanner.Scan() {
		line := scanner.Text()

		var event TestEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			// Skip invalid JSON lines
			continue
		}

		// Initialize package if not exists
		if _, exists := packageMap[event.Package]; !exists {
			packageMap[event.Package] = &Package{
				Name:       event.Package,
				Tests:      make([]*Test, 0),
				Benchmarks: make([]*Benchmark, 0),
			}
			testOrder[event.Package] = make([]string, 0)
		}

		switch event.Action {
		case "run":
			// Create new test when it starts running
			if event.Test != "" {
				testKey := event.Package + "::" + event.Test
				if _, exists := testMap[testKey]; !exists {
					testMap[testKey] = &Test{
						Name:   event.Test,
						Result: FAIL, // default to FAIL until we know the result
						Output: make([]string, 0),
					}
					// Track test order
					testOrder[event.Package] = append(testOrder[event.Package], testKey)
				}
			}

		case "output":
			// Append output to the corresponding test
			// Only process non-empty output
			if event.Output != "" && event.Test != "" {
				testKey := event.Package + "::" + event.Test
				if test, exists := testMap[testKey]; exists {
					// Trim trailing newline from output
					output := strings.TrimSuffix(event.Output, "\n")
					if output != "" {
						test.Output = append(test.Output, output)
					}
				}
			}

		case "pass":
			// Mark test as passed
			if event.Test != "" {
				testKey := event.Package + "::" + event.Test
				if test, exists := testMap[testKey]; exists {
					test.Result = PASS
					test.Duration = time.Duration(event.Elapsed * float64(time.Second))
					test.Time = int(test.Duration / time.Millisecond)
				}
			}

		case "fail":
			// Mark test as failed
			if event.Test != "" {
				testKey := event.Package + "::" + event.Test
				if test, exists := testMap[testKey]; exists {
					test.Result = FAIL
					test.Duration = time.Duration(event.Elapsed * float64(time.Second))
					test.Time = int(test.Duration / time.Millisecond)
				}
			}

		case "skip":
			// Mark test as skipped
			if event.Test != "" {
				testKey := event.Package + "::" + event.Test
				if test, exists := testMap[testKey]; exists {
					test.Result = SKIP
					test.Duration = time.Duration(event.Elapsed * float64(time.Second))
					test.Time = int(test.Duration / time.Millisecond)
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	// Build final report by assembling tests into packages in order
	for pkgName, pkg := range packageMap {
		var totalDuration time.Duration

		// Add tests in the order they were encountered
		for _, testKey := range testOrder[pkgName] {
			if test, exists := testMap[testKey]; exists {
				pkg.Tests = append(pkg.Tests, test)
				totalDuration += test.Duration
			}
		}

		pkg.Duration = totalDuration
		pkg.Time = int(totalDuration / time.Millisecond)

		// Only add package if it has tests
		if len(pkg.Tests) > 0 {
			report.Packages = append(report.Packages, *pkg)
		}
	}

	// If no packages were found and pkgName is provided, create an empty package
	if len(report.Packages) == 0 && pkgName != "" {
		report.Packages = append(report.Packages, Package{
			Name:  pkgName,
			Tests: make([]*Test, 0),
		})
	}

	return nil
}

// Failures counts the number of failed tests in this report
func (r *Report) Failures() int {
	count := 0

	for _, p := range r.Packages {
		for _, t := range p.Tests {
			if t.Result == FAIL {
				count++
			}
		}
	}

	return count
}
