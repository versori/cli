/*
 * Copyright (c) 2026 Versori Group Inc
 *
 * Use of this software is governed by the Business Source License 1.1
 * included in the LICENSE file at the root of this repository.
 *
 * Change Date: 2030-03-01
 * Change License: Apache License, Version 2.0
 *
 * As of the Change Date, in accordance with the Business Source License,
 * use of this software will be governed by the Apache License, Version 2.0.
 */

package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	headerFile := flag.String("header", "COPYRIGHT_HEADER.txt", "path to the copyright header file")
	rootDir := flag.String("dir", ".", "root directory to search for Go files")
	dryRun := flag.Bool("dry-run", false, "print files that would be modified without changing them")
	flag.Parse()

	headerBytes, err := os.ReadFile(*headerFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading header file %q: %v\n", *headerFile, err)
		os.Exit(1)
	}
	header := strings.TrimRight(string(headerBytes), "\n") + "\n"

	var modified, skipped int

	err = filepath.WalkDir(*rootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			// skip hidden directories and vendor
			name := d.Name()
			if name != "." && (strings.HasPrefix(name, ".") || name == "vendor") {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading %s: %w", path, err)
		}

		var body []byte
		var action string

		start, end, hasCopyright := findCopyrightBlock(content)
		switch {
		case hasCopyright && strings.TrimRight(string(content[start:end]), "\r\n") == strings.TrimRight(header, "\r\n"):
			// header is identical — nothing to do
			skipped++
			return nil
		case hasCopyright:
			// header exists but is outdated — replace it
			body = content[end:]
			action = "update header"
		default:
			// no header at all — prepend
			body = content
			action = "add header"
		}

		if *dryRun {
			fmt.Printf("would %s: %s\n", action, path)
			modified++
			return nil
		}

		fileStats, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("unable to get file stats for %s: %w", path, err)
		}

		newContent := header + "\n" + string(body)
		if err := os.WriteFile(path, []byte(newContent), fileStats.Mode()); err != nil {
			return fmt.Errorf("writing %s: %w", path, err)
		}

		fmt.Printf("%sd: %s\n", action, path)
		modified++
		return nil
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error walking directory: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\ndone: %d file(s) modified, %d file(s) already had header\n", modified, skipped)
}

// findCopyrightBlock looks for a /* ... */ block comment at the very start of
// content (before any package declaration) that contains "Copyright".
// It returns the byte range [start, end) of that block, including any trailing
// newline, so the caller can strip it cleanly.
func findCopyrightBlock(content []byte) (start, end int, found bool) {
	trimmed := bytes.TrimLeft(content, " \t\r\n")
	if !bytes.HasPrefix(trimmed, []byte("/*")) {
		return 0, 0, false
	}

	start = len(content) - len(trimmed)
	closeIdx := bytes.Index(trimmed, []byte("*/"))
	if closeIdx == -1 {
		return 0, 0, false
	}

	block := trimmed[:closeIdx+2]
	if !bytes.Contains(block, []byte("Copyright")) {
		return 0, 0, false
	}

	end = start + closeIdx + 2
	// consume the newline(s) that follow the closing */
	for end < len(content) && (content[end] == '\n' || content[end] == '\r') {
		end++
	}
	return start, end, true
}
