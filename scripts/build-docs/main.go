// Command build-docs converts site/md/*.md into the static doc pages under
// site/docs/<slug>/index.html using a single shared template. Run via
// `make docs` from the repo root.
//
// The generator is intentionally a standalone Go module so it does not pull
// goldmark into the main updu binary. It has zero options: edit the markdown
// files in site/md/ and rerun.
package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	gmhtml "github.com/yuin/goldmark/renderer/html"
)

// page is one item rendered into the sidebar and on disk.
type page struct {
	Slug  string // directory name under site/docs/, e.g. "http"; empty for the index
	Title string // shown in the sidebar
	Group string // sidebar grouping: "" (top), "Monitors", "Advanced monitors"
}

// pageOrder controls the sidebar ordering and grouping. The slug must match
// a markdown file at site/md/<slug>.md (or site/md/index.md for the empty slug).
var pageOrder = []page{
	{Slug: "", Title: "Overview", Group: ""},

	{Slug: "http", Title: "HTTP / HTTPS", Group: "Monitors"},
	{Slug: "tcp", Title: "TCP Port", Group: "Monitors"},
	{Slug: "dns", Title: "DNS", Group: "Monitors"},
	{Slug: "icmp", Title: "ICMP / Ping", Group: "Monitors"},
	{Slug: "ssh", Title: "SSH", Group: "Monitors"},
	{Slug: "ssl", Title: "SSL Certificate", Group: "Monitors"},
	{Slug: "api", Title: "JSON API", Group: "Monitors"},
	{Slug: "push", Title: "Push (Heartbeat)", Group: "Monitors"},
	{Slug: "websocket", Title: "WebSocket", Group: "Monitors"},
	{Slug: "smtp", Title: "SMTP Server", Group: "Monitors"},
	{Slug: "udp", Title: "UDP Port", Group: "Monitors"},
	{Slug: "redis", Title: "Redis", Group: "Monitors"},
	{Slug: "postgres", Title: "PostgreSQL", Group: "Monitors"},
	{Slug: "mysql", Title: "MySQL", Group: "Monitors"},
	{Slug: "mongo", Title: "MongoDB", Group: "Monitors"},

	{Slug: "https", Title: "HTTPS (TLS health)", Group: "Advanced monitors"},
	{Slug: "composite", Title: "Composite", Group: "Advanced monitors"},
	{Slug: "transaction", Title: "Transaction", Group: "Advanced monitors"},
	{Slug: "dns_http", Title: "DNS + HTTP", Group: "Advanced monitors"},
	{Slug: "grpc", Title: "gRPC Health", Group: "Advanced monitors"},
	{Slug: "prometheus", Title: "Prometheus Scrape", Group: "Advanced monitors"},
	{Slug: "database_query", Title: "Database Query", Group: "Advanced monitors"},
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "build-docs:", err)
		os.Exit(1)
	}
}

func run() error {
	repoRoot, err := findRepoRoot()
	if err != nil {
		return err
	}
	mdDir := filepath.Join(repoRoot, "site", "md")
	outDir := filepath.Join(repoRoot, "site", "docs")

	// Validate every page in pageOrder has a markdown source.
	for _, p := range pageOrder {
		src := mdSourcePath(mdDir, p.Slug)
		if _, err := os.Stat(src); err != nil {
			return fmt.Errorf("missing markdown source for %q: %w", p.Slug, err)
		}
	}

	// Warn about orphan markdown files that aren't in pageOrder.
	known := make(map[string]struct{}, len(pageOrder))
	for _, p := range pageOrder {
		known[p.Slug] = struct{}{}
	}
	entries, err := os.ReadDir(mdDir)
	if err != nil {
		return fmt.Errorf("reading %s: %w", mdDir, err)
	}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".md" {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".md")
		slug := name
		if name == "index" {
			slug = ""
		}
		if _, ok := known[slug]; !ok {
			fmt.Fprintf(os.Stderr, "build-docs: WARN orphan markdown not in pageOrder: %s\n", e.Name())
		}
	}

	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
		goldmark.WithRendererOptions(gmhtml.WithUnsafe()),
	)

	written := 0
	for _, p := range pageOrder {
		src := mdSourcePath(mdDir, p.Slug)
		body, err := os.ReadFile(src)
		if err != nil {
			return fmt.Errorf("reading %s: %w", src, err)
		}
		var buf bytes.Buffer
		if err := md.Convert(body, &buf); err != nil {
			return fmt.Errorf("rendering %s: %w", src, err)
		}
		page := renderTemplate(p, buf.String())

		dst := outputPath(outDir, p.Slug)
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return fmt.Errorf("mkdir %s: %w", dst, err)
		}
		if err := os.WriteFile(dst, []byte(page), 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", dst, err)
		}
		written++
	}

	fmt.Printf("build-docs: wrote %d pages under %s\n", written, outDir)
	return nil
}

// findRepoRoot walks up from CWD looking for go.mod with `module github.com/updu/updu`.
func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		gomod := filepath.Join(dir, "go.mod")
		data, err := os.ReadFile(gomod)
		if err == nil && bytes.Contains(data, []byte("module github.com/updu/updu\n")) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("could not locate updu repo root (no go.mod with module github.com/updu/updu found in any ancestor)")
		}
		dir = parent
	}
}

func mdSourcePath(mdDir, slug string) string {
	if slug == "" {
		return filepath.Join(mdDir, "index.md")
	}
	return filepath.Join(mdDir, slug+".md")
}

func outputPath(outDir, slug string) string {
	if slug == "" {
		return filepath.Join(outDir, "index.html")
	}
	return filepath.Join(outDir, slug, "index.html")
}

// pageTitle is what we use in the <title> tag.
func pageTitle(p page) string {
	if p.Slug == "" {
		return "Overview"
	}
	return p.Title
}

// assetPrefix is the relative path back to site/ root from the page directory.
func assetPrefix(slug string) string {
	if slug == "" {
		return ".."
	}
	return "../.."
}

// renderTemplate emits the full HTML page for one doc.
func renderTemplate(p page, body string) string {
	prefix := assetPrefix(p.Slug)
	var b strings.Builder

	fmt.Fprintf(&b, `<!DOCTYPE html>
<html lang="en">

<head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>%s — updu Docs</title>
    <link rel="icon" type="image/png" href="%s/favicon.png" />
    <link rel="preconnect" href="https://fonts.googleapis.com" />
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin />
    <link
        href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800&family=JetBrains+Mono:wght@400;700&display=swap"
        rel="stylesheet" />
    <link rel="stylesheet" href="%s/marketing.css" />
    <link rel="stylesheet" href="%s/docs/style.css" />
</head>

<body>
    <nav class="nav" id="nav"
        style="border-bottom: 1px solid var(--border); background: var(--bg-deep); position: sticky; top: 0;">
        <div class="nav-inner">
            <a href="/" class="nav-logo">
                <img src="%s/logo-dark.svg" alt="updu - uptime monitoring" />
            </a>
            <ul class="nav-links">
                <li><a href="/">Home</a></li>
                <li>
                    <a href="https://github.com/nwpeckham88/updu" class="nav-cta" target="_blank" rel="noopener">
                        GitHub
                    </a>
                </li>
            </ul>
        </div>
    </nav>

    <div class="docs-container">
        <aside class="docs-sidebar">
`, htmlEscape(pageTitle(p)), prefix, prefix, prefix, prefix)

	b.WriteString(renderSidebar(p.Slug))

	b.WriteString(`        </aside>
        <main class="docs-content">
`)
	b.WriteString(body)
	b.WriteString(`        </main>
    </div>
</body>

</html>
`)
	return b.String()
}

func renderSidebar(activeSlug string) string {
	// Group pages preserving pageOrder.
	var groups []string
	byGroup := map[string][]page{}
	for _, p := range pageOrder {
		if _, ok := byGroup[p.Group]; !ok {
			groups = append(groups, p.Group)
		}
		byGroup[p.Group] = append(byGroup[p.Group], p)
	}

	// Section headings: top group is "Documentation", remainder uses the group label.
	headings := map[string]string{
		"":                  "Documentation",
		"Monitors":          "Monitors",
		"Advanced monitors": "Advanced monitors",
	}

	var b strings.Builder
	for _, g := range groups {
		heading, ok := headings[g]
		if !ok {
			heading = g
		}
		fmt.Fprintf(&b, "            <h3>%s</h3>\n", htmlEscape(heading))
		b.WriteString("            <ul>\n")
		// stable order within group already comes from pageOrder, but make sure
		// we never accidentally reorder if pageOrder changes.
		items := append([]page(nil), byGroup[g]...)
		sort.SliceStable(items, func(i, j int) bool {
			return indexOf(items[i].Slug) < indexOf(items[j].Slug)
		})
		for _, p := range items {
			href := sidebarHref(p.Slug)
			cls := ""
			if p.Slug == activeSlug {
				cls = "active"
			}
			fmt.Fprintf(&b, "                <li><a href=\"%s\" class=\"%s\">%s</a></li>\n",
				href, cls, htmlEscape(p.Title))
		}
		b.WriteString("            </ul>\n")
	}
	return b.String()
}

func sidebarHref(slug string) string {
	if slug == "" {
		return "/docs/index.html"
	}
	return "/docs/" + slug + "/index.html"
}

func indexOf(slug string) int {
	for i, p := range pageOrder {
		if p.Slug == slug {
			return i
		}
	}
	return -1
}

func htmlEscape(s string) string {
	return strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
	).Replace(s)
}
