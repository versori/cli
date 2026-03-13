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
	"sort"
	"strings"
)

func main() {
	skillsDir := "skills"

	outputFlag := flag.String("out", filepath.Join(skillsDir, "AGENTS.md"), "Path to the output file")
	outputFile := *outputFlag

	flag.Parse()

	var files []string

	// First, collect all the relevant markdown files
	err := filepath.Walk(skillsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Only process .md files
		if !strings.HasSuffix(info.Name(), ".md") {
			return nil
		}

		// Skip the output file itself if it already exists
		// We use absolute paths to ensure accurate comparison in case the output is outside the skills dir
		absPath, _ := filepath.Abs(path)
		absOutputPath, _ := filepath.Abs(outputFile)
		if absPath == absOutputPath {
			return nil
		}

		files = append(files, path)
		return nil
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error walking skills directory: %v\n", err)
		os.Exit(1)
	}

	// Sort the files based on the requested priority
	sort.Slice(files, func(i, j int) bool {
		nameI := filepath.Base(files[i])
		nameJ := filepath.Base(files[j])

		// Helper function to assign a priority rank
		rank := func(name string) int {
			if strings.EqualFold(name, "readme.md") {
				return 0
			}
			if strings.EqualFold(name, "skill.md") || strings.EqualFold(name, "skills.md") {
				return 1
			}
			return 2
		}

		rankI := rank(nameI)
		rankJ := rank(nameJ)

		// If ranks are different, sort by rank (lower rank first)
		if rankI != rankJ {
			return rankI < rankJ
		}
		// If ranks are the same, sort alphabetically by path
		return files[i] < files[j]
	})

	var combinedContent bytes.Buffer

	// Now read and combine the files in the sorted order
	for _, path := range files {
		content, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to read file %s: %v\n", path, err)
			os.Exit(1)
		}

		// Add a header for each file to separate them clearly
		combinedContent.WriteString(fmt.Sprintf("\n\n<!-- BEGIN %s -->\n\n", path))
		combinedContent.Write(content)
		combinedContent.WriteString(fmt.Sprintf("\n\n<!-- END %s -->\n", path))

		fmt.Printf("Added: %s\n", path)
	}

	// Ensure the output directory exists
	outDir := filepath.Dir(outputFile)
	if err := os.MkdirAll(outDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output directory %s: %v\n", outDir, err)
		os.Exit(1)
	}

	// Write the combined content to the output file
	err = os.WriteFile(outputFile, combinedContent.Bytes(), 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing to %s: %v\n", outputFile, err)
		os.Exit(1)
	}

	fmt.Printf("Successfully combined skills into %s\n", outputFile)
}
