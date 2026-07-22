package render

import (
	"fmt"
	"github.com/silver-river-us/bound/internal/model"
	"sort"
	"strings"
)

func Structurizr(a *model.Architecture) string {
	var b strings.Builder
	fmt.Fprintf(&b, "workspace %q {\n  model {\n", a.Name)
	names := make([]string, 0, len(a.Contexts))
	for name := range a.Contexts {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		context := a.Contexts[name]
		description := context.Implementation.Language + ": " + context.Implementation.Locator
		fmt.Fprintf(&b, "    %s = softwareSystem %q {\n      description %q\n    }\n", name, name, description)
	}
	for _, relation := range a.Relations {
		label := "depends on"
		if relation.Via != "" {
			label = "via " + relation.Via
		}
		fmt.Fprintf(&b, "    %s -> %s %q\n", relation.From, relation.To, label)
	}
	b.WriteString("  }\n  views {\n    systemLandscape {\n      include *\n      autolayout lr\n    }\n  }\n}\n")
	return b.String()
}
