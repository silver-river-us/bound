package model

import "fmt"

func (a *Architecture) Validate() error {
	if a.Name == "" {
		return fmt.Errorf("architecture name is required")
	}
	for name, object := range a.Objects {
		if object.Name == "" || object.Name != name {
			return fmt.Errorf("architecture has an invalid object")
		}
		for attributeName, attribute := range object.Attributes {
			if attribute.Name == "" || attribute.Name != attributeName || attribute.Type == "" {
				return fmt.Errorf("object %s has an invalid attribute", name)
			}
		}
	}
	for name, context := range a.Contexts {
		if context.Implementation.Language == "" || context.Implementation.Locator == "" {
			return fmt.Errorf("context %s must declare an implementation", name)
		}
		for interfaceName, contract := range context.Interfaces {
			if interfaceName != contract.Name || contract.Name == "" {
				return fmt.Errorf("context %s has an invalid interface", name)
			}
			for operationName, operation := range contract.Operations {
				if operationName != operation.Name || operation.Name == "" {
					return fmt.Errorf("interface %s.%s has an invalid operation", name, interfaceName)
				}
			}
		}
		for exposed := range context.Exposes {
			if _, ok := context.Interfaces[exposed]; !ok {
				return fmt.Errorf("context %s exposes undefined interface %s", name, exposed)
			}
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
		if relation.Via != "" {
			if !to.Exposes[relation.Via] {
				return fmt.Errorf("%s does not expose %s", relation.To, relation.Via)
			}
			if _, ok := to.Interfaces[relation.Via]; !ok {
				return fmt.Errorf("%s exposes undefined interface %s", relation.To, relation.Via)
			}
		}
	}
	return nil
}

func (a *Architecture) Allows(from, to string) bool {
	for _, relation := range a.Relations {
		if relation.From == from && relation.To == to {
			return true
		}
	}
	return false
}
