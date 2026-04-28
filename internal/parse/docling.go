package parse

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// extractDoclingMarkdown runs the `docling` CLI on the file and returns its
// markdown output. Docling produces structured markdown (with proper tables,
// headings, and reading order) which is generally easier for an LLM to parse
// than column-aligned plaintext from `pdftotext -layout`.
//
// Returns ("", nil) if docling isn't installed — callers treat that as
// "skip, fall back to pdftotext."
func extractDoclingMarkdown(ctx context.Context, filePath string) (string, error) {
	binPath := findDoclingBinary()
	if binPath == "" {
		return "", nil
	}

	outDir, err := os.MkdirTemp("", "docling-out-")
	if err != nil {
		return "", fmt.Errorf("docling tempdir: %w", err)
	}
	defer os.RemoveAll(outDir)

	cmd := exec.CommandContext(ctx, binPath,
		"--to", "md",
		"--output", outDir,
		filePath,
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("docling failed: %w (%s)", err, strings.TrimSpace(stderr.String()))
	}

	// Docling writes <basename>.md into the output dir.
	base := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
	mdPath := filepath.Join(outDir, base+".md")
	data, err := os.ReadFile(mdPath)
	if err != nil {
		// Fallback: take any .md file in the dir.
		entries, _ := os.ReadDir(outDir)
		for _, e := range entries {
			if strings.HasSuffix(strings.ToLower(e.Name()), ".md") {
				data, err = os.ReadFile(filepath.Join(outDir, e.Name()))
				break
			}
		}
		if err != nil {
			return "", fmt.Errorf("docling output not found: %w", err)
		}
	}
	return string(data), nil
}

// findDoclingBinary returns the path to the docling CLI, or "" if missing.
// Checks DOCLING_BIN env first, then PATH, then a known venv location.
func findDoclingBinary() string {
	if v := os.Getenv("DOCLING_BIN"); v != "" {
		if _, err := os.Stat(v); err == nil {
			return v
		}
	}
	if p, err := exec.LookPath("docling"); err == nil {
		return p
	}
	for _, candidate := range []string{
		"/tmp/docling-venv/bin/docling",
		"/opt/docling-venv/bin/docling",
		os.ExpandEnv("$HOME/.local/bin/docling"),
	} {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
}

// dumpDoclingComparison runs docling alongside the active text path and
// writes both outputs to parse-debug/ so we can compare quality. This is
// best-effort — docling failures or absence are silently skipped.
func dumpDoclingComparison(filePath, documentID string) {
	if findDoclingBinary() == "" {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		t0 := time.Now()
		md, err := extractDoclingMarkdown(ctx, filePath)
		dur := time.Since(t0).Round(time.Millisecond)
		if err != nil {
			log.Printf("[%s] docling extraction failed in %s: %v", documentID, dur, err)
			return
		}
		log.Printf("[%s] docling produced %d chars of markdown in %s", documentID, len(md), dur)

		home, err := os.UserHomeDir()
		if err != nil {
			return
		}
		dir := filepath.Join(home, "Library", "Application Support", "LocalFinance", "parse-debug")
		_ = os.MkdirAll(dir, 0o755)
		name := fmt.Sprintf("%s-docling-%s.md", time.Now().Format("20060102-150405"), documentID)
		path := filepath.Join(dir, name)
		header := fmt.Sprintf("<!-- docling output\ndocument_id: %s\nsource: %s\nlen: %d\nelapsed: %s\n-->\n\n", documentID, filepath.Base(filePath), len(md), dur)
		_ = os.WriteFile(path, []byte(header+md), 0o644)
		log.Printf("[%s] docling dump written to %s", documentID, path)
	}()
}
