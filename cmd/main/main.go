package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/ramadhantriyant/feedfinder/internal/discovery"
)

var (
	colReset  = "\033[0m"
	colBold   = "\033[1m"
	colGreen  = "\033[32m"
	colYellow = "\033[33m"
	colCyan   = "\033[36m"
	colRed    = "\033[31m"
	colGray   = "\033[90m"
)

func main() {
	var (
		timeoutSec = flag.Int("timeout", 15, "HTTP timeout in seconds")
		userAgent  = flag.String("ua", "", "Custom User-Agent header (optional)")
		verbose    = flag.Bool("v", false, "Verbose debug output")
		noColor    = flag.Bool("no-color", false, "Disable coloured output")
	)
	flag.Usage = func() {
		fmt.Fprint(os.Stderr, colBold+"feedfinder"+colReset+" - discover RSS/Atom/JSON feeds from a URL\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n  feedfinder [flags] <url>\n\nFlags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  feedfinder https://blog.rust-lang.org\n")
		fmt.Fprintf(os.Stderr, "  feedfinder -v -timeout 20 news.ycombinator.com\n")
		fmt.Fprintf(os.Stderr, "  feedfinder -ua \"Mozilla/5.0\" https://example.com\n")
	}
	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}
	rawURL := flag.Arg(0)

	if *noColor {
		colReset, colBold, colGreen, colYellow, colCyan, colRed, colGray = "", "", "", "", "", "", ""
	}

	timeout := time.Duration(*timeoutSec) * time.Second

	fmt.Printf("\n%s%s feedfinder%s - scanning %s%s%s\n\n",
		colBold, colCyan, colReset,
		colBold, rawURL, colReset)

	printStage("Stage 1 + 2", "Fetching URL & checking Content-Type")
	results, err := discovery.Discover(rawURL, *userAgent, timeout, *verbose)

	if err != nil {
		fmt.Fprintf(os.Stderr, "\n%s✗ Error: %s%s\n\n", colRed, err, colReset)
		os.Exit(1)
	}

	if len(results) == 0 {
		fmt.Printf("%s✗ No feeds found.%s\n\n", colYellow, colReset)
		fmt.Printf("  The site may use a non-standard feed path, or may not publish a feed.\n")
		fmt.Printf("  Try passing an explicit feed URL or using -v for debug output.\n\n")
		os.Exit(0)
	}

	fmt.Printf("%s✓ Found %d feed(s):%s\n\n", colGreen, len(results), colReset)
	for i, r := range results {
		fmt.Printf("  %s%d.%s %s%s%s\n", colBold, i+1, colReset, colGreen, r.URL, colReset)
		if r.Title != "" {
			fmt.Printf("     %sTitle:%s  %s\n", colGray, colReset, r.Title)
		}
		fmt.Printf("     %sSource:%s %s\n", colGray, colReset, r.Source)
		fmt.Println()
	}
}

func printStage(stage, desc string) {
	fmt.Printf("  %s%s%s %s\n", colBold, stage, colReset, desc)
}
