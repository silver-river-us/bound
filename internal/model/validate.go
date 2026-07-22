package model

import "fmt"

func (a *Architecture) Validate() error {
	if a.Name == "" {
		return fmt.Errorf("architecture name is required")
	}
	for name, context := range a.Contexts {
		if context.Implementation.Language == "" || context.Implementation.Locator == "" {
			return fmt.Errorf("context %s must declare an implementation", name)
		}
	}
	for _, relation := range a.Relations {
		if _, ok := a.Contexts[relation.From]; !ok {
			return fmt.Errorf("relationship references unknown context %s", relation.From)
		}
		to, ok := a.Contexts[relation.To]
		if !ok {
			return fmt.Errorf("relationship references unknown context %s", relation.To)
		}
		if relation.From == relation.To {
			return fmt.Errorf("context %s cannot depend on itself", relation.From)
		}
		if relation.Via != "" && !to.Exposes[relation.Via] {
			return fmt.Errorf("%s does not expose %s", relation.To, relation.Via)
		}
	}
	return nil
}
