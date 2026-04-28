package parse

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/sagarjhaa/statementmd"
)

// docPrefixRE matches the "doc_<uuid-fragment>_" prefix the upload handler
// adds to incoming files (e.g. "doc_f663272b-64d5-4b_capitaone.pdf").
// Capturing group 1 is the original filename.
var docPrefixRE = regexp.MustCompile(`^doc_[a-f0-9-]+_(.+)$`)

// originalFilename strips the upload-handler prefix and the extension,
// returning a filesystem-safe stem suitable for inclusion in a debug
// filename. Empty input → "unknown".
func originalFilename(filePath string) string {
	base := filepath.Base(filePath)
	if m := docPrefixRE.FindStringSubmatch(base); len(m) == 2 {
		base = m[1]
	}
	stem := strings.TrimSuffix(base, filepath.Ext(base))
	stem = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			return r
		default:
			return '_'
		}
	}, stem)
	if stem == "" {
		return "unknown"
	}
	return stem
}

// statementMarkdown converts a file to markdown via statementmd with a
// 30s budget. Synchronous — used by the active parse path.
func statementMarkdown(filePath string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return statementmd.ConvertContext(ctx, filePath)
}

// dumpStatementMarkdown writes the same markdown to parse-debug/ for
// after-the-fact inspection. Best-effort and async — never blocks the
// active parse path. Useful even though the active path now uses the
// markdown directly, because we still want the file on disk so users
// can compare what the LLM saw vs what it produced.
func dumpStatementMarkdown(filePath, documentID string) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		t0 := time.Now()
		md, err := statementmd.ConvertContext(ctx, filePath)
		dur := time.Since(t0).Round(time.Millisecond)
		if err != nil {
			log.Printf("[%s] statementmd failed in %s: %v", documentID, dur, err)
			return
		}
		log.Printf("[%s] statementmd produced %d chars of markdown in %s", documentID, len(md), dur)

		home, err := os.UserHomeDir()
		if err != nil {
			return
		}
		dir := filepath.Join(home, "Library", "Application Support", "LocalFinance", "parse-debug")
		_ = os.MkdirAll(dir, 0o755)
		name := fmt.Sprintf("%s-%s-statementmd.md", time.Now().Format("20060102-150405"), originalFilename(filePath))
		path := filepath.Join(dir, name)
		header := fmt.Sprintf("<!-- statementmd output\ndocument_id: %s\nsource: %s\nlen: %d\nelapsed: %s\n-->\n\n", documentID, filepath.Base(filePath), len(md), dur)
		_ = os.WriteFile(path, []byte(header+md), 0o644)
	}()
}
