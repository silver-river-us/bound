package lsp_test

import (
	"fmt"
	"strings"
	"testing"

	"bound/src/lsp"
)

func TestAnalyzeReportsBoundDiagnostics(t *testing.T) {
	diagnostics := lsp.Analyze("architecture Example do\n  implementation future \".\"\nend\n")
	if len(diagnostics) != 1 {
		t.Fatalf("diagnostics = %#v, want one diagnostic", diagnostics)
	}
	if diagnostics[0].Source != "bound" || diagnostics[0].Range.Start.Line != 1 {
		t.Fatalf("diagnostic = %#v, want bound source at line 2", diagnostics[0])
	}
}

func TestCompletionItemsIncludeArchitectureKeywords(t *testing.T) {
	items := lsp.CompletionItems()
	found := false
	for _, item := range items {
		if item["label"] == "architecture" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("completion items = %#v, missing architecture", items)
	}
}

func TestServerRespondsToInitializeAndCompletion(t *testing.T) {
	input := strings.NewReader(message(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`) + message(`{"jsonrpc":"2.0","id":2,"method":"textDocument/completion","params":{}}`))
	var output strings.Builder
	if err := lsp.NewServer().Run(input, &output); err != nil {
		t.Fatalf("run server: %v", err)
	}
	if !strings.Contains(output.String(), `"serverInfo":{"name":"bound-lsp"`) || !strings.Contains(output.String(), `"label":"architecture"`) {
		t.Fatalf("output = %s", output.String())
	}
}

func TestServerPublishesDiagnosticsForOpenDocuments(t *testing.T) {
	params := `{"textDocument":{"uri":"file:///tmp/example.bo","text":"architecture Example do\n  implementation future \".\"\nend\n"}}`
	input := strings.NewReader(message(`{"jsonrpc":"2.0","method":"textDocument/didOpen","params":` + params + `}`))
	var output strings.Builder
	if err := lsp.NewServer().Run(input, &output); err != nil {
		t.Fatalf("run server: %v", err)
	}
	if !strings.Contains(output.String(), `"method":"textDocument/publishDiagnostics"`) || !strings.Contains(output.String(), "unsupported implementation language") {
		t.Fatalf("output = %s", output.String())
	}
}

func message(body string) string { return fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(body), body) }
