// The published demo index: docs/demos/index.html.
//
// Every demo is the same page and the same .wasm with a different query
// string, so there is nothing for a crawler to find but the launcher — and the
// launcher is a terminal drawn into a canvas, which reads as an empty
// document. Forty-five demos exist and none of them is named anywhere a search
// engine can see.
//
// This is ONE page, deliberately. A page per demo would be forty-five
// documents each carrying a single sentence, which is the shape of a doorway
// page and is fairly treated as one; the descriptions are one line each and
// pretending otherwise would not make them longer. Listed together, with what
// the thing actually is around them, they are a reference.
package main

import (
	"bytes"
	"fmt"
	htmpl "html/template"
	"os"
	"path/filepath"

	"github.com/0magnet/tuiwasm/demos"
)

type demoEntry struct {
	Name, Desc, Shape, URL string
}

type demoIndex struct {
	Screen, Text     []demoEntry
	Count            int
	Title, Desc      string
	Canonical, Image string
	CSS              htmpl.CSS
}

func buildIndex() demoIndex {
	ix := demoIndex{
		Canonical: site + "demos/",
		Image:     site + "tuiwasm-gallery.png",
	}
	for _, d := range demos.All() {
		e := demoEntry{Name: d.Name, Desc: d.Desc, Shape: "screen", URL: site + "?demo=" + d.Name}
		if d.Text != nil {
			e.Shape = "text"
			ix.Text = append(ix.Text, e)
			continue
		}
		ix.Screen = append(ix.Screen, e)
	}
	ix.Count = len(ix.Screen) + len(ix.Text)
	ix.Title = fmt.Sprintf("All %d demos — tuiwasm", ix.Count)
	ix.Desc = fmt.Sprintf("Every one of the %d terminal demos and animations tuiwasm runs in a browser tab — "+
		"Conway's Life, Langton's ant, boids, a Julia set, reaction-diffusion, matrix rain, "+
		"plasma, fire and the rest — each one a link that opens it running.", ix.Count)
	ix.CSS = htmpl.CSS(indexCSS)
	return ix
}

func writeSite(dir string, dry bool) error {
	var buf bytes.Buffer
	if err := indexTmpl.Execute(&buf, buildIndex()); err != nil {
		return err
	}
	path := filepath.Join(dir, "demos", "index.html")
	if dry {
		old, err := os.ReadFile(path) //nolint:gosec // a path this command composed from its own flag
		if err != nil || !bytes.Equal(old, buf.Bytes()) {
			return fmt.Errorf("out of date: %s", path)
		}
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	if old, err := os.ReadFile(path); err == nil && bytes.Equal(old, buf.Bytes()) { //nolint:gosec // as above
		fmt.Println("gendemos: demos/index.html unchanged")
		return nil
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		return err
	}
	fmt.Printf("gendemos: wrote %s (%d demos)\n", path, buildIndex().Count)
	return nil
}

const indexCSS = `
:root { color-scheme: light dark;
        --bg:#ffffff; --fg:#15171c; --muted:#5c6470; --line:#e4e7ec; --accent:#2f6f4f; --card:#f7f8fa; }
@media (prefers-color-scheme: dark) {
  :root { --bg:#0d0f14; --fg:#e6e9ee; --muted:#8b94a3; --line:#232833; --accent:#6fce9f; --card:#141821; }
}
* { box-sizing: border-box; }
body { margin:0; background:var(--bg); color:var(--fg);
       font:16px/1.65 ui-sans-serif, system-ui, -apple-system, "Segoe UI", sans-serif; }
.wrap { max-width: 880px; margin:0 auto; padding: 2rem 1.1rem 5rem; }
a { color:var(--accent); }
nav { font-size:.86rem; color:var(--muted); margin-bottom:2rem; }
nav a { color:var(--muted); text-decoration:none; }
h1 { font-size:1.9rem; line-height:1.2; margin:0 0 .6rem; letter-spacing:-.01em; }
h2 { font-size:1.1rem; margin:2.6rem 0 .2rem; padding-bottom:.45rem; border-bottom:1px solid var(--line); }
.lead { color:var(--muted); max-width:70ch; }
img.gallery { width:100%; height:auto; border:1px solid var(--line); border-radius:10px;
              margin:1.6rem 0; background:#000; }
ul.demos { list-style:none; padding:0; margin:.8rem 0 0; }
ul.demos li { padding:.55rem 0; border-bottom:1px solid var(--line); display:flex; gap:.9rem;
              align-items:baseline; flex-wrap:wrap; }
ul.demos a { font-family:ui-monospace,SFMono-Regular,Menlo,monospace; font-weight:600;
             min-width:9rem; text-decoration:none; }
ul.demos a:hover { text-decoration:underline; }
ul.demos span { color:var(--muted); }
`

var indexTmpl = htmpl.Must(htmpl.New("demos").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.Title}}</title>
<meta name="description" content="{{.Desc}}">
<meta name="author" content="0magnet">
<link rel="canonical" href="{{.Canonical}}">
<meta property="og:type" content="website">
<meta property="og:site_name" content="magnetosphere">
<meta property="og:title" content="{{.Title}}">
<meta property="og:description" content="{{.Desc}}">
<meta property="og:url" content="{{.Canonical}}">
<meta property="og:image" content="{{.Image}}">
<meta name="twitter:card" content="summary_large_image">
<meta name="twitter:title" content="{{.Title}}">
<meta name="twitter:description" content="{{.Desc}}">
<meta name="twitter:image" content="{{.Image}}">
<script type="application/ld+json">
{"@context":"https://schema.org","@type":"CollectionPage","name":{{.Title}},"url":{{.Canonical}},
"description":{{.Desc}},
"isPartOf":{"@type":"WebSite","name":"tuiwasm","url":"https://tuiwasm.magnetosphere.net/"}}
</script>
<style>{{.CSS}}</style>
</head>
<body>
<div class="wrap">
<nav><a href="../">tuiwasm</a> › demos</nav>
<h1>All {{.Count}} demos</h1>
<p class="lead">tuiwasm runs Go's terminal-UI libraries — tcell for the screen layer, and the Charm
stack above it: lipgloss for styling, bubbletea for the update loop, glamour for Markdown — inside a
browser tab, drawing into a real terminal emulator built on
<a href="https://github.com/0magnet/xterm-go">xterm-go</a>. Every demo below is the same
WebAssembly binary with a different query string; each link opens it running.</p>
<img class="gallery" src="../tuiwasm-gallery.png"
     alt="Every tuiwasm demo, one frame each, captured from the browser build" loading="lazy">
<h2>Screen demos <span class="lead">— they paint cells and read keys</span></h2>
<ul class="demos">
{{range .Screen}}<li><a href="{{.URL}}">{{.Name}}</a> <span>{{.Desc}}</span></li>
{{end}}</ul>
<h2>Text demos <span class="lead">— they write styled text</span></h2>
<ul class="demos">
{{range .Text}}<li><a href="{{.URL}}">{{.Name}}</a> <span>{{.Desc}}</span></li>
{{end}}</ul>
<p class="lead" style="margin-top:2.5rem">Source:
<a href="https://github.com/0magnet/tuiwasm">github.com/0magnet/tuiwasm</a>.
More Go and WebAssembly projects: <a href="https://magnetosphere.net/software">magnetosphere.net/software</a>.</p>
</div>
</body>
</html>
`))
