package module

import (
	_ "embed"
	"fmt"
	"sort"
	"strings"

	"go.yorun.ai/skelc/internal/codegen/common"
)

const (
	packageJSONFilename    = "package.json"
	defaultTSImportVersion = "*"
)

//go:embed tpl/package.json.tpl
var packageJSONTemplate string

type Option struct {
	Out             string
	PackageName     string
	Imports         map[string]string
	ResolvedImports map[string]string
}

type PackageJSONPayload struct {
	PackageName      string
	PeerDependencies []PackageJSONDependency
}

type PackageJSONDependency struct {
	Package string
	Version string
}

func Generate(option Option) error {
	payload, err := buildPackageJSONPayload(option)
	if err != nil {
		return err
	}
	renderer := common.NewRenderer(option.Out)
	renderer.Render(packageJSONFilename, packageJSONTemplate, payload)
	return renderer.Err()
}

func buildPackageJSONPayload(option Option) (*PackageJSONPayload, error) {
	dependencies, err := packageJSONDependencies(option)
	if err != nil {
		return nil, err
	}
	return &PackageJSONPayload{PackageName: option.PackageName, PeerDependencies: dependencies}, nil
}

func packageJSONDependencies(option Option) ([]PackageJSONDependency, error) {
	dependencies := map[string]string{"@yorun-ai/vrpc": defaultTSImportVersion}
	for _, path := range option.Imports {
		dependency, err := parseTSImportDependency(path)
		if err != nil {
			return nil, err
		}
		dependency.fillDefaultVersion()
		dependencies[dependency.Package] = dependency.Version
	}
	for domainName, importPath := range option.ResolvedImports {
		if option.Imports[domainName] == "" {
			dependencies[importPath] = defaultTSImportVersion
		}
	}
	return sortedPackageJSONDependencies(dependencies), nil
}

func ImportPath(path string) (string, error) {
	dependency, err := parseTSImportDependency(path)
	return dependency.Package, err
}

func (d *PackageJSONDependency) fillDefaultVersion() {
	if d.Version == "" {
		d.Version = defaultTSImportVersion
	}
}

func parseTSImportDependency(path string) (PackageJSONDependency, error) {
	index := strings.LastIndex(path, "@")
	if index <= 0 {
		if path == "" {
			return PackageJSONDependency{}, fmt.Errorf("invalid TypeScript import %q: missing package", path)
		}
		return PackageJSONDependency{Package: path}, nil
	}
	pkg := path[:index]
	version := path[index+1:]
	if version == "" {
		return PackageJSONDependency{}, fmt.Errorf("invalid TypeScript import %q: missing version", path)
	}
	return PackageJSONDependency{Package: pkg, Version: version}, nil
}

func sortedPackageJSONDependencies(dependencies map[string]string) []PackageJSONDependency {
	ordered := make([]PackageJSONDependency, 0, len(dependencies))
	for dependency, version := range dependencies {
		ordered = append(ordered, PackageJSONDependency{Package: dependency, Version: version})
	}
	sort.Slice(ordered, func(i, j int) bool {
		return ordered[i].Package < ordered[j].Package
	})
	return ordered
}
