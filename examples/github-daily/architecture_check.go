package main

import (
	"fmt"

	"github.com/silver-river-us/bound/internal/analyze"
	"github.com/silver-river-us/bound/internal/parser"
)

func checkArchitecture(path, sourceRoot string) error {
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
