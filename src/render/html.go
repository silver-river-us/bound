package render

import (
	"fmt"
	"html"
	"strconv"
	"strings"

	"bound/src/model"
)

// HTMLOptions controls how MermaidHTMLWithOptions loads Mermaid and how the
// review is presented when printed. An empty value preserves the historical
// CDN-backed output of MermaidHTML.
type HTMLOptions struct {
	// MermaidURL is the URL of an ESM Mermaid distribution. It may point to a
	// local file or an application-served asset instead of the public CDN.
	MermaidURL string
	// InlineMermaid is trusted JavaScript for a locally embedded Mermaid
	// distribution. The script must expose the Mermaid API as
	// globalThis.mermaid (as UMD builds do).
	InlineMermaid string
	// PrintFriendly adds print CSS and hides interactive-only controls when the
	// page is printed. It does not change the screen presentation.
	PrintFriendly bool
}

const defaultMermaidURL = "https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.esm.min.mjs"

// DefaultMermaidURL is the default CDN asset used by HTML reviews.
const DefaultMermaidURL = defaultMermaidURL

// MermaidHTML renders a standalone architecture review page. Its default
// output remains CDN-backed for compatibility with existing generated reviews.
func MermaidHTML(a *model.Architecture) string {
	return MermaidHTMLWithOptions(a, HTMLOptions{})
}

// MermaidHTMLWithOptions renders a standalone architecture review page with
// optional local/embedded Mermaid assets and print-friendly styling.
func MermaidHTMLWithOptions(a *model.Architecture, options HTMLOptions) string {
	type diagram struct {
		Title string
		Body  string
		ID    string
	}
	diagrams := []diagram{
		{Title: "Context relationships", Body: contextDiagram(a), ID: "context-relationships"},
		{Title: "Component view", Body: componentDiagram(a), ID: "component-view"},
		{Title: "Interaction view", Body: interactionDiagram(a), ID: "interaction-view"},
		{Title: "Contracts and modules", Body: contractDiagram(a), ID: "contracts-and-modules"},
		{Title: "Source ownership", Body: sourceDiagram(a), ID: "source-ownership"},
	}

	var b strings.Builder
	b.WriteString(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Architecture: `)
	b.WriteString(html.EscapeString(a.Name))
	b.WriteString(`</title>
  <style>
    :root { color-scheme: dark; --bg: #0f172a; --panel: #111827; --line: #334155; --text: #e5e7eb; --muted: #94a3b8; --accent: #818cf8; }
    * { box-sizing: border-box; }
    body { margin: 0; background: var(--bg); color: var(--text); font: 15px/1.55 -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; }
    main { max-width: 1600px; margin: 0 auto; padding: 40px 28px 72px; }
    header { border-bottom: 1px solid var(--line); margin-bottom: 28px; padding-bottom: 24px; }
    h1 { margin: 0 0 8px; font-size: clamp(28px, 4vw, 46px); letter-spacing: -0.03em; }
    h2 { margin: 0; font-size: 20px; }
    p { color: var(--muted); margin: 8px 0 0; }
    code { color: #c4b5fd; }
    nav { display: flex; flex-wrap: wrap; gap: 8px 14px; margin-top: 20px; }
    nav a { color: var(--accent); }
    .summary { display: grid; grid-template-columns: repeat(auto-fit, minmax(120px, 1fr)); gap: 10px; margin: 22px 0 28px; }
    .stat { background: var(--panel); border: 1px solid var(--line); border-radius: 8px; padding: 12px 14px; }
    .stat strong { display: block; font-size: 22px; }
    .stat span { color: var(--muted); font-size: 13px; }
    .grid { display: grid; gap: 24px; }
    .card { background: var(--panel); border: 1px solid var(--line); border-radius: 12px; overflow: hidden; scroll-margin-top: 18px; }
    .card-header { align-items: center; display: flex; justify-content: space-between; gap: 16px; padding: 16px 18px; }
    .domain-filter { align-items: center; display: flex; flex-wrap: wrap; gap: 8px; }
    select { background: #1e293b; border: 1px solid #475569; border-radius: 6px; color: var(--text); padding: 5px 8px; }
    .diagram { background: #0b1120; border-top: 1px solid var(--line); min-height: 240px; overflow: auto; padding: 24px; }
    .diagram svg { height: auto; max-width: none; min-width: 720px; }
    .controls { display: flex; gap: 6px; }
    button { background: #1e293b; border: 1px solid #475569; border-radius: 6px; color: var(--text); cursor: pointer; padding: 5px 10px; }
    button:hover { background: #334155; }
    .fullscreen { background: var(--bg); inset: 0; overflow: auto; padding: 24px; position: fixed; z-index: 10; }
    .fullscreen .diagram { min-height: calc(100vh - 110px); }
    .fullscreen .diagram svg { min-width: 100%; }
    .domain-view:not(.active) { display: none; }
    .error { color: #fecaca; font-family: monospace; white-space: pre-wrap; }

    @media (max-width: 700px) { main { padding: 24px 14px 48px; } .card-header { align-items: flex-start; flex-direction: column; } }
  </style>
`)
	if options.PrintFriendly {
		b.WriteString(`<style media="print">
      :root { color-scheme: light; }
      body { background: white; color: black; }
      main { max-width: none; padding: 0; }
      header { margin-bottom: 14px; }
      nav, .controls, .domain-filter, .summary { display: none !important; }
      .card { background: white; border: 0; break-inside: avoid; margin-bottom: 18px; }
      .diagram { background: white; border: 1px solid #bbb; }
      .diagram svg { min-width: 0; max-width: 100%; }
      .domain-view:not(.active) { display: none; }
    </style>
`)
	}
	if options.InlineMermaid != "" {
		b.WriteString(`  <script type="module">`)
		b.WriteString(options.InlineMermaid)
		b.WriteString(`
    const mermaid = globalThis.mermaid;
`)
	} else {
		mermaidURL := options.MermaidURL
		if mermaidURL == "" {
			mermaidURL = defaultMermaidURL
		}
		fmt.Fprintf(&b, "  <script type=\"module\">\n    import mermaid from %s;\n", jsString(mermaidURL))
	}
	b.WriteString(`
        mermaid.initialize({ startOnLoad: false, theme: "dark", securityLevel: "strict" });
    const zoom = new WeakMap();
    const render = async () => {
      const nodes = [...document.querySelectorAll(".mermaid-source")];
      for (let index = 0; index < nodes.length; index += 1) {
        const source = nodes[index];
        const host = source.parentElement;
        try {
          const { svg } = await mermaid.render("bound-diagram-" + index, source.textContent);
          host.innerHTML = svg;
          zoom.set(host, 1);
        } catch (error) {
          host.innerHTML = "<div class=\"error\">" + String(error) + "</div>";
        }
      }
    };
    const activeDiagram = (card) => card.querySelector(".domain-view.active .diagram") || card.querySelector(".diagram");
    const scale = (host, factor) => {
      const next = Math.max(0.35, Math.min(2.5, (zoom.get(host) || 1) * factor));
      zoom.set(host, next);
      const svg = host.querySelector("svg");
      if (svg) svg.style.transform = "scale(" + next + ")";
      if (svg) svg.style.transformOrigin = "top left";
    };
    document.addEventListener("change", (event) => {
      const select = event.target.closest("select[data-domain-filter]");
      if (!select) return;
      const card = select.closest(".card");
      card.querySelectorAll(".domain-view").forEach((view) => view.classList.toggle("active", view.dataset.domainScope === select.value));
    });
    document.addEventListener("click", (event) => {
      const button = event.target.closest("button[data-action]");
      if (!button) return;
      const card = button.closest(".card");
      const host = activeDiagram(card);
      if (button.dataset.action === "in") scale(host, 1.2);
      if (button.dataset.action === "out") scale(host, 1 / 1.2);
      if (button.dataset.action === "reset") {
        zoom.set(host, 1);
        const svg = host.querySelector("svg");
        if (svg) svg.style.transform = "scale(1)";
      }
      if (button.dataset.action === "fullscreen") card.classList.toggle("fullscreen");
    });
    render();
  </script>
`)
	b.WriteString(`</head>
<body>
  <main>
    <header>
      <h1>Architecture: `)
	b.WriteString(html.EscapeString(a.Name))
	b.WriteString(`</h1>
      <p>Implementation: <code>`)
	b.WriteString(html.EscapeString(a.Implementation.Language))
	b.WriteString(`</code> at <code>`)
	b.WriteString(html.EscapeString(a.Implementation.Locator))
	b.WriteString(`</code></p>
      <p>Generated by Bound. Diagrams are interactive: zoom, reset, and fullscreen controls are available on each view.</p>
      <nav aria-label="Review sections">`)
	for _, item := range diagrams {
		fmt.Fprintf(&b, `<a href="#%s">%s</a>`, item.ID, html.EscapeString(item.Title))
	}
	b.WriteString(`<a href="#domain-model">Domain model</a></nav>
    </header>
    <section class="summary" aria-label="Architecture summary">`)
	writeStat(&b, "Contexts", len(a.Contexts))
	writeStat(&b, "Interfaces", countInterfaces(a))
	writeStat(&b, "Modules", len(a.Modules))
	writeStat(&b, "Domain types", len(a.Objects)+countContextTypes(a))
	writeStat(&b, "Relationships", len(a.Relations))
	writeStat(&b, "Source files", len(a.Files))
	b.WriteString(`</section>
    <div class="grid">`)
	for _, item := range diagrams {
		fmt.Fprintf(&b, `      <section class="card" id="%s">
        <div class="card-header"><h2>%s</h2>%s</div>
        <div class="diagram"><pre class="mermaid-source">%s</pre></div>
      </section>
`, item.ID, html.EscapeString(item.Title), controls(), html.EscapeString(item.Body))
	}
	writeDomainCard(&b, a)
	b.WriteString(`    </div>
  </main>
</body>
</html>
`)
	return b.String()
}

func jsString(value string) string {
	return strconv.Quote(value)
}

func writeStat(b *strings.Builder, label string, value int) {
	fmt.Fprintf(b, `<div class="stat"><strong>%d</strong><span>%s</span></div>`, value, html.EscapeString(label))
}

func controls() string {
	return `<div class="controls"><button data-action="out" aria-label="Zoom out">−</button><button data-action="reset">Reset</button><button data-action="in" aria-label="Zoom in">+</button><button data-action="fullscreen">Fullscreen</button></div>`
}

func writeDomainCard(b *strings.Builder, a *model.Architecture) {
	fmt.Fprintf(b, `      <section class="card" id="domain-model">
        <div class="card-header"><h2>Domain model</h2><div class="domain-filter"><label for="domain-scope">Scope</label><select id="domain-scope" data-domain-filter>`)
	fmt.Fprintf(b, `<option value="all">All architecture types</option>`)
	for _, name := range sortedContextNames(a) {
		fmt.Fprintf(b, `<option value="%s">%s</option>`, html.EscapeString(mermaidID("domain", name)), html.EscapeString(name))
	}
	b.WriteString(`</select>`)
	b.WriteString(controls())
	b.WriteString(`</div></div>`)
	fmt.Fprintf(b, `<div class="domain-view active" data-domain-scope="all"><div class="diagram"><pre class="mermaid-source">%s</pre></div></div>`, html.EscapeString(domainDiagram(a)))
	for _, name := range sortedContextNames(a) {
		fmt.Fprintf(b, `<div class="domain-view" data-domain-scope="%s"><div class="diagram"><pre class="mermaid-source">%s</pre></div></div>`, html.EscapeString(mermaidID("domain", name)), html.EscapeString(domainDiagramForContext(a.Contexts[name])))
	}
	b.WriteString(`</section>
`)
}

func countInterfaces(a *model.Architecture) int {
	count := 0
	for _, context := range a.Contexts {
		count += len(context.Interfaces)
	}
	return count
}

func countContextTypes(a *model.Architecture) int {
	count := 0
	for _, context := range a.Contexts {
		for _, contract := range context.Interfaces {
			count += len(contract.Types)
		}
	}
	return count
}
