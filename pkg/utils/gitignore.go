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

// Package gitignore provides functionality to check if files match .gitignore patterns.
package utils

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// Matcher checks whether files should be ignored based on .gitignore rules.
type Matcher struct {
	basePath string
	patterns []pattern
}

// pattern represents a single gitignore pattern.
type pattern struct {
	original     string
	negated      bool
	dirOnly      bool
	anchored     bool // pattern is anchored to basePath (starts with / or contains /)
	segments     []string
	hasDoubleAst bool
}

// NewChecker creates a new Matcher.
func NewChecker() *Matcher {
	return &Matcher{}
}

// LoadFile loads patterns from a .gitignore file in the given directory.
func (m *Matcher) LoadFile(dir string) error {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	m.basePath = absDir

	gitignorePath := filepath.Join(absDir, ".gitignore")
	file, err := os.Open(gitignorePath)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if p := parseLine(line); p != nil {
			m.patterns = append(m.patterns, *p)
		}
	}

	return scanner.Err()
}

// Match returns true if the given path should be ignored.
// The path should be an absolute path, and fi provides file info.
func (m *Matcher) Match(path string, fi os.FileInfo) bool {
	if m.basePath == "" || len(m.patterns) == 0 {
		return false
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	// Path must be under basePath
	if !strings.HasPrefix(absPath, m.basePath) {
		return false
	}

	// Get relative path from basePath
	relPath := strings.TrimPrefix(absPath, m.basePath)
	relPath = strings.TrimPrefix(relPath, string(filepath.Separator))
	if relPath == "" {
		return false
	}

	// Normalize to forward slashes for matching
	relPath = filepath.ToSlash(relPath)
	isDir := fi.IsDir()

	// Check patterns in order, last match wins
	ignored := false
	for _, p := range m.patterns {
		if p.matches(relPath, isDir) {
			ignored = !p.negated
		}
	}

	return ignored
}

// parseLine parses a single line from a .gitignore file.
func parseLine(line string) *pattern {
	// Remove trailing spaces (unless escaped)
	line = strings.TrimRight(line, " \t")

	// Skip empty lines and comments
	if line == "" || strings.HasPrefix(line, "#") {
		return nil
	}

	p := &pattern{original: line}

	// Check for negation
	if strings.HasPrefix(line, "!") {
		p.negated = true
		line = line[1:]
	}

	// Check for directory-only match
	if strings.HasSuffix(line, "/") {
		p.dirOnly = true
		line = strings.TrimSuffix(line, "/")
	}

	// Check if pattern is anchored (starts with / or contains /)
	if strings.HasPrefix(line, "/") {
		p.anchored = true
		line = strings.TrimPrefix(line, "/")
	} else if strings.Contains(line, "/") {
		p.anchored = true
	}

	// Check for ** patterns
	p.hasDoubleAst = strings.Contains(line, "**")

	// Split into segments
	p.segments = strings.Split(line, "/")

	return p
}

// matches checks if the pattern matches the given relative path.
func (p *pattern) matches(relPath string, isDir bool) bool {
	// Directory-only patterns don't match files
	if p.dirOnly && !isDir {
		return false
	}

	pathSegments := strings.Split(relPath, "/")

	if p.hasDoubleAst {
		return p.matchDoubleAsterisk(pathSegments)
	}

	if p.anchored {
		return p.matchAnchored(pathSegments)
	}

	// Non-anchored pattern: can match at any level
	return p.matchUnanchored(pathSegments)
}

// matchAnchored matches a pattern that is anchored to the base path.
func (p *pattern) matchAnchored(pathSegments []string) bool {
	if len(pathSegments) < len(p.segments) {
		return false
	}

	// Must match from the start
	for i, seg := range p.segments {
		if !matchSegment(seg, pathSegments[i]) {
			return false
		}
	}

	return true
}

// matchUnanchored matches a pattern that can appear anywhere in the path.
func (p *pattern) matchUnanchored(pathSegments []string) bool {
	// Single segment pattern: match against any path component
	if len(p.segments) == 1 {
		for _, pathSeg := range pathSegments {
			if matchSegment(p.segments[0], pathSeg) {
				return true
			}
		}

		return false
	}

	// Multi-segment pattern: try matching at each position
	for start := 0; start <= len(pathSegments)-len(p.segments); start++ {
		matched := true
		for i, seg := range p.segments {
			if !matchSegment(seg, pathSegments[start+i]) {
				matched = false

				break
			}
		}
		if matched {
			return true
		}
	}

	return false
}

// matchDoubleAsterisk handles patterns with ** (matches zero or more directories).
func (p *pattern) matchDoubleAsterisk(pathSegments []string) bool {
	return matchWithDoubleAsterisk(p.segments, pathSegments)
}

// matchWithDoubleAsterisk recursively matches pattern segments against path segments.
func matchWithDoubleAsterisk(patternSegs, pathSegs []string) bool {
	pi, pathi := 0, 0

	for pi < len(patternSegs) {
		if patternSegs[pi] == "**" {
			// ** at end matches everything
			if pi == len(patternSegs)-1 {
				return true
			}

			// Try matching ** against 0, 1, 2, ... path segments
			for skip := 0; skip <= len(pathSegs)-pathi; skip++ {
				if matchWithDoubleAsterisk(patternSegs[pi+1:], pathSegs[pathi+skip:]) {
					return true
				}
			}

			return false
		}

		if pathi >= len(pathSegs) {
			return false
		}

		if !matchSegment(patternSegs[pi], pathSegs[pathi]) {
			return false
		}

		pi++
		pathi++
	}

	return pathi == len(pathSegs)
}

// matchSegment matches a single path segment against a pattern segment using glob rules.
func matchSegment(pattern, name string) bool {
	matched, _ := filepath.Match(pattern, name)

	return matched
}
