package framework

import (
	"fmt"

	"bound/src/lib/compiler"
)

// CheckArchitecture parses, validates, and checks an architecture against its
// implementation source tree.
func CheckArchitecture(path, sourceRoot string) error {
	_, err := compiler.Compile(path, compiler.Options{SourceRoot: sourceRoot})
	if err != nil {
		return fmt.Errorf("compile architecture: %w", err)
	}
	return nil
}
