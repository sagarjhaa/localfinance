// Package prompts is the single source of truth for every LLM prompt the
// app uses. Each prompt lives in a `.tmpl` file in this directory so they
// can be reviewed / edited / diffed without spelunking through Go code.
//
// Templates use the standard text/template syntax. Code calls Render
// (or MustRender for the common case) with the template name and a
// map[string]any of fields to substitute.
//
// Why go:embed: the .app distribution bundles the prompts as part of
// the binary, same way internal/webui/dist embeds the React build, so
// there's never an "I can't find the prompts directory" failure at
// install time.
package prompts

import (
	"bytes"
	"embed"
	"fmt"
	"sync"
	"text/template"
)

//go:embed *.tmpl
var fs embed.FS

var (
	cacheMu sync.RWMutex
	cache   = map[string]*template.Template{}

	// funcs are exposed to every template. Add new helpers here as
	// templates need them — keep the surface small so prompts stay
	// readable.
	funcs = template.FuncMap{
		// add1 returns its int argument + 1 — used for 1-based indices
		// in `{{range}}` loops where humans count from 1.
		"add1": func(i int) int { return i + 1 },
	}
)

// Render loads and renders the named template (without the .tmpl
// extension). Returns the rendered string or an error if the template
// is missing or fails to execute.
func Render(name string, data any) (string, error) {
	t, err := load(name)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("prompts: render %q: %w", name, err)
	}
	return buf.String(), nil
}

// MustRender is Render but panics on failure. Use it for prompts where
// a render error means a developer typo in a template we ship — caller
// has no recovery path. Don't use this with user-supplied data that
// could legitimately fail to render.
func MustRender(name string, data any) string {
	s, err := Render(name, data)
	if err != nil {
		panic(err)
	}
	return s
}

func load(name string) (*template.Template, error) {
	cacheMu.RLock()
	if t, ok := cache[name]; ok {
		cacheMu.RUnlock()
		return t, nil
	}
	cacheMu.RUnlock()

	cacheMu.Lock()
	defer cacheMu.Unlock()
	if t, ok := cache[name]; ok {
		return t, nil
	}
	body, err := fs.ReadFile(name + ".tmpl")
	if err != nil {
		return nil, fmt.Errorf("prompts: load %q: %w", name, err)
	}
	// Disable HTML-style escaping by using text/template (already imported).
	// Disable the default delimiters' option-collision pitfalls by leaving
	// them as `{{` / `}}` — none of our prompts contain those literally.
	t, err := template.New(name).Funcs(funcs).Option("missingkey=error").Parse(string(body))
	if err != nil {
		return nil, fmt.Errorf("prompts: parse %q: %w", name, err)
	}
	cache[name] = t
	return t, nil
}
