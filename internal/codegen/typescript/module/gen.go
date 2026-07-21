package module

import (
	_ "embed"
	"sort"
	"strings"

	"go.yorun.ai/skelc/internal/codegen"
	"go.yorun.ai/skelc/internal/util/checkutil"
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

func Generate(option Option) {
	payload := buildPackageJSONPayload(option)
	codegen.NewRenderer(option.Out).Render(packageJSONFilename, packageJSONTemplate, payload)
}

func buildPackageJSONPayload(option Option) *PackageJSONPayload {
	return &PackageJSONPayload{
		PackageName:      option.PackageName,
		PeerDependencies: packageJSONDependencies(option),
	}
}

func packageJSONDependencies(option Option) []PackageJSONDependency {
	dependencies := map[string]string{"@yorun-ai/vrpc": defaultTSImportVersion}
	for _, path := range option.Imports {
		dependency := parseTSImportDependency(path)
		dependency.fillDefaultVersion()
		dependencies[dependency.Package] = dependency.Version
	}
	for domainName, importPath := range option.ResolvedImports {
		if option.Imports[domainName] == "" {
			dependencies[importPath] = defaultTSImportVersion
		}
	}
	return sortedPackageJSONDependencies(dependencies)
}

func ImportPath(path string) string {
	return parseTSImportDependency(path).Package
}

func (d *PackageJSONDependency) fillDefaultVersion() {
	if d.Version == "" {
		d.Version = defaultTSImportVersion
	}
}

func parseTSImportDependency(path string) PackageJSONDependency {
	index := strings.LastIndex(path, "@")
	if index <= 0 {
		return PackageJSONDependency{Package: path}
	}
	pkg := path[:index]
	version := path[index+1:]
	checkutil.Check(version != "", "invalid ts import %q: missing version", path)
	return PackageJSONDependency{Package: pkg, Version: version}
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
