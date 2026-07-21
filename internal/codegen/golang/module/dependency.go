package module

import (
	"sort"
	"strings"

	"go.yorun.ai/skelc/internal/util/checkutil"
)

type _GoImportDependency struct {
	Module  string
	Version string
}

func goModDependencies(imports map[string]string, extraDependencies []string) []_GoImportDependency {
	dependencies := map[string]string{}
	for _, path := range imports {
		dependency := parseGoImportDependency(path)
		dependency.fillDefaultVersion()
		dependencies[dependency.Module] = dependency.Version
	}
	for _, path := range extraDependencies {
		dependency := parseGoImportDependency(path)
		dependency.fillDefaultVersion()
		dependencies[dependency.Module] = dependency.Version
	}
	return sortedGoImportDependencies(dependencies)
}

func (d *_GoImportDependency) fillDefaultVersion() {
	if d.Version == "" {
		d.Version = defaultGoImportVersion
	}
}

func parseGoImportDependency(path string) _GoImportDependency {
	index := strings.LastIndex(path, "@")
	if index < 0 {
		return _GoImportDependency{Module: path}
	}
	module := path[:index]
	version := path[index+1:]
	checkutil.Check(module != "", "invalid go import %q: missing module", path)
	checkutil.Check(version != "", "invalid go import %q: missing version", path)
	return _GoImportDependency{Module: module, Version: version}
}

func ImportPath(path string) string {
	return parseGoImportDependency(path).Module
}

func sortedGoImportDependencies(dependencies map[string]string) []_GoImportDependency {
	ordered := make([]_GoImportDependency, 0, len(dependencies))
	for module, version := range dependencies {
		ordered = append(ordered, _GoImportDependency{Module: module, Version: version})
	}
	sort.Slice(ordered, func(i, j int) bool {
		return ordered[i].Module < ordered[j].Module
	})
	return ordered
}
