package framework

import (
	"fmt"

	"bound/internal/analyze"
	"bound/internal/parser"
)

// CheckArchitecture parses, validates, and checks an architecture against its
// implementation source tree.
func CheckArchitecture(path, sourceRoot string) error {
	architecture, err := parser.ParseFile(path)
	if err != nil {
		return fmt.Errorf("parse architecture: %w", err)
	}
	if err := architecture.Validate(); err != nil {
		return fmt.Errorf("validate architecture: %w", err)
	}
	if err := analyze.Go(sourceRoot, architecture); err != nil {
		return fmt.Errorf("check implementation: %w", err)
	}
	return nil
}
