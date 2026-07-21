package source

import (
	"fmt"
	"strings"
)

func buildPackageName(moduleScope string, domainName string, pubOnly bool) string {
	parts := strings.Split(domainName, ".")
	if pubOnly {
		parts[len(parts)-1] = parts[len(parts)-1] + "pub"
	}
	name := strings.Join(parts, "-")
	if strings.Contains(strings.TrimPrefix(moduleScope, "@"), "/") {
		return fmt.Sprintf("%s-%s", moduleScope, name)
	}
	return fmt.Sprintf("%s/%s", moduleScope, name)
}

func importPackageAlias(domainName string, pubOnly bool) string {
	parts := strings.Split(domainName, ".")
	if pubOnly {
		return parts[len(parts)-1] + "pub"
	}
	return parts[len(parts)-1]
}
