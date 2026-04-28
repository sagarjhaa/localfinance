package parse

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/sagarjhaa/statementmd"
)

// dumpStatementMarkdown converts the uploaded file to markdown via the
// statementmd library and writes it to ~/Library/Application Support/
// LocalFinance/parse-debug/. Best-effort and async — never blocks the
// active parse path.
//
// Right now this is a side-by-side artifact for inspection. Once we're
// happy with quality we can flip parseText to use this output directly
// instead of (or alongside) the pdftotext text path.
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
		name := fmt.Sprintf("%s-statementmd-%s.md", time.Now().Format("20060102-150405"), documentID)
		path := filepath.Join(dir, name)
		header := fmt.Sprintf("<!-- statementmd output\ndocument_id: %s\nsource: %s\nlen: %d\nelapsed: %s\n-->\n\n", documentID, filepath.Base(filePath), len(md), dur)
		_ = os.WriteFile(path, []byte(header+md), 0o644)
	}()
}
