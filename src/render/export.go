package render

import (
	"fmt"
	"html"
	"strings"

	"bound/src/model"
)

// ExportFormat identifies a review output format.
type ExportFormat string

const (
	FormatMarkdown ExportFormat = "markdown"
	FormatSVG      ExportFormat = "svg"
	FormatHTML     ExportFormat = "html"
)

// ExportOptions controls format selection and HTML rendering. The zero value
// selects Markdown, which is the least surprising text-oriented export.
type ExportOptions struct {
	Format ExportFormat
	HTML   HTMLOptions
}

// Export renders an architecture review in Markdown, SVG, or HTML. The
// Markdown and HTML helpers remain available for callers that prefer a typed
// API without a format switch.
func Export(a *model.Architecture, options ExportOptions) (string, error) {
	format := options.Format
	if format == "" {
		format = FormatMarkdown
	}
	switch format {
	case FormatMarkdown:
		return MermaidMarkdown(a), nil
	case FormatSVG:
		return MermaidSVG(a), nil
	case FormatHTML:
		return MermaidHTMLWithOptions(a, options.HTML), nil
	default:
		return "", fmt.Errorf("unsupported review export format %q", format)
	}
}

// MermaidSVG returns a valid, self-contained SVG review artifact. Mermaid
// source is rendered as selectable text and retained in metadata, making this
// export useful in CI, documents, and offline review even when no Mermaid
// runtime is available. Interactive diagram rendering remains the HTML
// export's responsibility.
func MermaidSVG(a *model.Architecture) string {
	sources := []struct {
		title  string
		source string
	}{
		{"Context relationships", contextDiagram(a)},
		{"Component view", componentDiagram(a)},
		{"Interaction view", interactionDiagram(a)},
		{"Contracts and modules", contractDiagram(a)},
		{"Domain model", domainDiagram(a)},
		{"Source ownership", sourceDiagram(a)},
	}

	const lineHeight = 18
	lines := []string{"Architecture: " + a.Name}
	for _, item := range sources {
		lines = append(lines, "", item.title)
		lines = append(lines, strings.Split(item.source, "\n")...)
	}
	if len(lines) > 1 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	height := 24 + len(lines)*lineHeight
	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" role="img" aria-labelledby="title" viewBox="0 0 1200 %d" width="1200" height="%d">`, height, height)
	fmt.Fprintf(&b, `<title id="title">%s</title><metadata>%s</metadata>`, xmlEscape("Architecture: "+a.Name), xmlEscape(strings.Join(lines, "\n")))
	b.WriteString(`<rect width="100%" height="100%" fill="white"/><g fill="black" font-family="ui-monospace, SFMono-Regular, Menlo, monospace" font-size="14">`)
	for index, line := range lines {
		fmt.Fprintf(&b, `<text x="16" y="%d">%s</text>`, 24+(index+1)*lineHeight, html.EscapeString(line))
	}
	b.WriteString(`</g></svg>`)
	return b.String()
}

func xmlEscape(value string) string {
	return html.EscapeString(value)
}
