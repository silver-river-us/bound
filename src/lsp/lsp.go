// Package lsp implements the Language Server Protocol features for Bound
// architecture files.
package lsp

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"strings"

	"bound/src/model"
	"bound/src/parser"
)

type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

type Diagnostic struct {
	Range    Range  `json:"range"`
	Severity int    `json:"severity,omitempty"`
	Source   string `json:"source,omitempty"`
	Message  string `json:"message"`
}

type TextDocument struct {
	URI  string `json:"uri"`
	Text string `json:"text"`
}

type Server struct {
	documents map[string]string
}

// NewServer creates a Bound language server.
func NewServer() *Server { return &Server{documents: map[string]string{}} }

// Analyze parses and validates a Bound document without running an
// implementation backend. It is useful to editors because source trees may be
// incomplete while a specification is being designed.
func Analyze(text string) []Diagnostic {
	architecture, err := parser.Parse(strings.NewReader(text))
	if err != nil {
		return []Diagnostic{diagnosticForError(err)}
	}
	// Buffer parsing intentionally does not resolve relative imports. Defer
	// semantic validation for imported documents to `bound compile`, otherwise
	// an open editor buffer would report false unknown-contract errors.
	for _, context := range architecture.Contexts {
		if len(context.Imports) > 0 {
			return nil
		}
	}
	if err := architecture.Validate(); err != nil {
		return []Diagnostic{diagnosticForError(err)}
	}
	return nil
}

// Run serves LSP requests over the standard input/output streams.
func (s *Server) Run(input io.Reader, output io.Writer) error {
	reader := bufio.NewReader(input)
	for {
		body, err := readMessage(reader)
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		var request request
		if err := json.Unmarshal(body, &request); err != nil {
			return err
		}
		response, notification := s.handle(request)
		if notification {
			if uri, ok := diagnosticURI(request); ok {
				params := map[string]interface{}{"uri": uri, "diagnostics": s.diagnostics(uri)}
				if err := writeMessage(output, map[string]interface{}{"jsonrpc": "2.0", "method": "textDocument/publishDiagnostics", "params": params}); err != nil {
					return err
				}
			}
			if request.Method == "exit" {
				return nil
			}
			continue
		}
		if err := writeMessage(output, response); err != nil {
			return err
		}
		if request.Method == "exit" {
			return nil
		}
	}
}

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type response struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   interface{} `json:"error,omitempty"`
}

func (s *Server) handle(request request) (response, bool) {
	if len(request.ID) == 0 || string(request.ID) == "null" {
		s.handleNotification(request)
		return response{}, true
	}
	result := interface{}(nil)
	switch request.Method {
	case "initialize":
		result = map[string]interface{}{
			"capabilities": map[string]interface{}{
				"textDocumentSync":   1,
				"completionProvider": map[string]interface{}{"triggerCharacters": []string{":", " "}},
			},
			"serverInfo": map[string]string{"name": "bound-lsp", "version": "0.1.0"},
		}
	case "textDocument/completion":
		result = CompletionItems()
	case "shutdown":
		result = nil
	default:
		result = nil
	}
	var id interface{}
	_ = json.Unmarshal(request.ID, &id)
	return response{JSONRPC: "2.0", ID: id, Result: result}, false
}

func diagnosticURI(request request) (string, bool) {
	if request.Method != "textDocument/didOpen" && request.Method != "textDocument/didChange" {
		return "", false
	}
	var params struct {
		TextDocument struct {
			URI string `json:"uri"`
		} `json:"textDocument"`
	}
	if json.Unmarshal(request.Params, &params) != nil || params.TextDocument.URI == "" {
		return "", false
	}
	return params.TextDocument.URI, true
}

func (s *Server) diagnostics(uri string) []Diagnostic {
	return Analyze(s.documents[uri])
}

func (s *Server) handleNotification(request request) {
	switch request.Method {
	case "textDocument/didOpen":
		var params struct {
			TextDocument TextDocument `json:"textDocument"`
		}
		if json.Unmarshal(request.Params, &params) == nil {
			s.documents[params.TextDocument.URI] = params.TextDocument.Text
		}
	case "textDocument/didChange":
		var params struct {
			TextDocument struct {
				URI string `json:"uri"`
			} `json:"textDocument"`
			ContentChanges []struct {
				Text string `json:"text"`
			} `json:"contentChanges"`
		}
		if json.Unmarshal(request.Params, &params) == nil && len(params.ContentChanges) > 0 {
			s.documents[params.TextDocument.URI] = params.ContentChanges[len(params.ContentChanges)-1].Text
		}
	case "textDocument/didClose":
		var params struct {
			TextDocument struct {
				URI string `json:"uri"`
			} `json:"textDocument"`
		}
		if json.Unmarshal(request.Params, &params) == nil {
			delete(s.documents, params.TextDocument.URI)
		}
	}
}

func diagnosticForError(err error) Diagnostic {
	diagnostic := Diagnostic{Severity: 1, Source: "bound", Message: err.Error()}
	var parserError *parser.Error
	var modelError *model.Error
	if errors.As(err, &parserError) {
		diagnostic.Message = parserError.Message
		diagnostic.Range = pointRange(parserError.Line, parserError.Column)
	}
	if errors.As(err, &modelError) {
		diagnostic.Message = modelError.Message
		if modelError.Suggestion != "" {
			diagnostic.Message += " (suggestion: " + modelError.Suggestion + ")"
		}
		diagnostic.Range = pointRange(modelError.Span.Line, modelError.Span.Column)
	}
	return diagnostic
}

func pointRange(line, column int) Range {
	if line < 1 {
		line = 1
	}
	if column < 1 {
		column = 1
	}
	position := Position{Line: line - 1, Character: column - 1}
	return Range{Start: position, End: Position{Line: position.Line, Character: position.Character + 1}}
}

// CompletionItems returns language keyword completions for a Bound document.
func CompletionItems() []map[string]interface{} {
	keywords := []string{"architecture", "implementation", "context", "interface", "entity", "value", "behavior", "state", "module", "implements", "uses", "exposes", "relationship", "entrypoint", "import", "quality", "rules", "end"}
	items := make([]map[string]interface{}, 0, len(keywords))
	for _, keyword := range keywords {
		items = append(items, map[string]interface{}{"label": keyword, "kind": 14})
	}
	return items
}

func readMessage(reader *bufio.Reader) ([]byte, error) {
	length := 0
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 && strings.EqualFold(strings.TrimSpace(parts[0]), "content-length") {
			if _, err := fmt.Sscanf(strings.TrimSpace(parts[1]), "%d", &length); err != nil {
				return nil, err
			}
		}
	}
	if length <= 0 {
		return nil, fmt.Errorf("missing Content-Length")
	}
	body := make([]byte, length)
	_, err := io.ReadFull(reader, body)
	return body, err
}

func writeMessage(output io.Writer, value interface{}) error {
	body, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(output, "Content-Length: %d\r\n\r\n", len(body))
	if err == nil {
		_, err = output.Write(body)
	}
	return err
}

// URIPath converts a file URI to a local path for clients that need it.
func URIPath(uri string) string {
	parsed, err := url.Parse(uri)
	if err != nil || parsed.Scheme != "file" {
		return uri
	}
	path, err := url.PathUnescape(parsed.Path)
	if err != nil {
		return filepath.Clean(parsed.Path)
	}
	return filepath.Clean(path)
}
