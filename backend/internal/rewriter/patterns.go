package rewriter

import (
	"os"
	"regexp"
	"strings"
)

// patchFile reads the file at path, applies all renames line by line, and
// optionally writes the modified content back.  Returns the list of patches
// (one entry per modified line).
func patchFile(path string, renames []Rename, dryRun bool) ([]Patch, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(data), "\n")
	var patches []Patch
	modified := false

	for i, line := range lines {
		newLine := line
		for _, r := range renames {
			newLine = applyRename(newLine, r.OldName, r.NewName)
		}
		if newLine != line {
			patches = append(patches, Patch{
				File:    path,
				Line:    i + 1,
				OldLine: line,
				NewLine: newLine,
			})
			lines[i] = newLine
			modified = true
		}
	}

	if modified && !dryRun {
		err = os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0644)
		if err != nil {
			return patches, err
		}
	}
	return patches, nil
}

// applyRename replaces all occurrences of oldName with newName in a single
// source line.  It targets common reference patterns:
//
//  1. ES import / export declarations
//  2. CommonJS require() calls
//  3. CSS url() references (with or without quotes)
//  4. HTML src=, href=, poster=, data-src= attributes
//  5. Markdown image links  ![alt](path)
//
// The function returns the original line unchanged when no pattern matches,
// preventing false positives.
func applyRename(line, oldName, newName string) string {
	escaped := regexp.QuoteMeta(oldName)

	// 1. ES import / export: import x from '…/logo.jpg' or export * from "…/logo.jpg"
	reImport := regexp.MustCompile(
		`((?:import|export)\s[^"'` + "`" + `]*?["'` + "`" + `][^"'` + "`" + `]*?)` +
			escaped +
			`(["'` + "`" + `])`,
	)

	// 2. require('…/logo.jpg')
	reRequire := regexp.MustCompile(
		`(require\s*\(\s*["'` + "`" + `][^"'` + "`" + `]*?)` +
			escaped +
			`(["'` + "`" + `]\s*\))`,
	)

	// 3. CSS url(…) — with or without quotes
	reURL := regexp.MustCompile(
		`(url\s*\(\s*["']?[^"')]*?)` +
			escaped +
			`(["']?\s*\))`,
	)

	// 4. HTML attributes: src=, href=, poster=, data-src=
	reAttr := regexp.MustCompile(
		`((?:src|href|poster|data-src)\s*=\s*["'][^"']*)` +
			escaped +
			`(["'])`,
	)

	// 5. Markdown image: ![alt](…/logo.jpg)
	reMarkdown := regexp.MustCompile(
		`(\!\[[^\]]*\]\([^)]*)` +
			escaped +
			`(\))`,
	)

	result := line
	result = reImport.ReplaceAllString(result, "${1}"+newName+"${2}")
	result = reRequire.ReplaceAllString(result, "${1}"+newName+"${2}")
	result = reURL.ReplaceAllString(result, "${1}"+newName+"${2}")
	result = reAttr.ReplaceAllString(result, "${1}"+newName+"${2}")
	result = reMarkdown.ReplaceAllString(result, "${1}"+newName+"${2}")
	return result
}
