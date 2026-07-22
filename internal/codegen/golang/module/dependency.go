package module

import (
	"fmt"
	"sort"
	"strings"
)

type _GoImportDependency struct {
	Module  string
	Version string
}

func goModDependencies(imports map[string]string, extraDependencies []string) ([]_GoImportDependency, error) {
	dependencies := map[string]string{}
	for _, path := range imports {
		dependency, err := parseGoImportDependency(path)
		if err != nil {
			return nil, err
		}
		dependency.fillDefaultVersion()
		dependencies[dependency.Module] = dependency.Version
	}
	for _, path := range extraDependencies {
		dependency, err := parseGoImportDependency(path)
		if err != nil {
			return nil, err
		}
		dependency.fillDefaultVersion()
		dependencies[dependency.Module] = dependency.Version
	}
	return sortedGoImportDependencies(dependencies), nil
}

func (d *_GoImportDependency) fillDefaultVersion() {
	if d.Version == "" {
		d.Version = defaultGoImportVersion
	}
}

func parseGoImportDependency(path string) (_GoImportDependency, error) {
	index := strings.LastIndex(path, "@")
	if index < 0 {
		if path == "" {
			return _GoImportDependency{}, fmt.Errorf("invalid Go import %q: missing module", path)
		}
		return _GoImportDependency{Module: path}, nil
	}
	module := path[:index]
	version := path[index+1:]
	if module == "" {
		return _GoImportDependency{}, fmt.Errorf("invalid Go import %q: missing module", path)
	}
	if version == "" {
		return _GoImportDependency{}, fmt.Errorf("invalid Go import %q: missing version", path)
	}
	return _GoImportDependency{Module: module, Version: version}, nil
}

func ImportPath(path string) (string, error) {
	dependency, err := parseGoImportDependency(path)
	return dependency.Module, err
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
