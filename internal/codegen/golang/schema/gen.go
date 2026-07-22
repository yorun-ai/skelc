package schema

import (
	"embed"
	"go/format"
	"io/fs"
	"strings"
	"text/template"

	"go.yorun.ai/skelc/internal/codegen/common"
	"go.yorun.ai/skelc/internal/codegen/golang/view"
	"go.yorun.ai/skelc/internal/util/checkutil"
	"go.yorun.ai/skelc/model"
)

//go:embed tpl
var templateFS embed.FS

const schemaGoFilename = "schema.go"

var schemaGoTemplate = loadTemplates()

type _Gen struct {
	Domain *model.Domain

	view            *view.Domain
	mode            view.Mode
	pkgName         string
	compilerVersion string
	Renderer        *common.Renderer
}

type Option struct {
	Domain          *model.Domain
	View            *view.Domain
	Mode            view.Mode
	PackageName     string
	CompilerVersion string
	Out             string
}

func Generate(option Option) error {
	gen := newGen(option)
	gen.gen()
	return gen.Renderer.Err()
}

func newGen(option Option) *_Gen {
	return &_Gen{
		Domain:          option.Domain,
		view:            option.View,
		mode:            option.Mode,
		pkgName:         option.PackageName,
		compilerVersion: option.CompilerVersion,
		Renderer:        common.NewRenderer(option.Out),
	}
}

func (g *_Gen) gen() {
	payload := g.buildSchemaGoPayload()
	content := common.RenderTemplateWithFuncs(schemaGoTemplate, payload, g.schemaGoTemplateFuncs())
	formatted, err := format.Source([]byte(content))
	checkutil.CheckNilError(err, "format generated %s failed", schemaGoFilename)
	g.Renderer.Write(schemaGoFilename, string(formatted))
}

func (g *_Gen) isSplitPub() bool {
	return g.mode == view.ModePub
}

func (g *_Gen) isSplitRegular() bool {
	return g.mode == view.ModeRegular
}

func loadTemplates() string {
	names, err := fs.Glob(templateFS, "tpl/*.go.tpl")
	checkutil.CheckNilError(err, "list go schema templates failed")
	var templates strings.Builder
	for _, name := range names {
		content, readErr := templateFS.ReadFile(name)
		checkutil.CheckNilError(readErr, "read go schema template %s failed", name)
		templates.Write(content)
		templates.WriteByte('\n')
	}
	return templates.String()
}

type SchemaGoPayload struct {
	PackageName string
	Schema      *_DomainSchema
}

func (g *_Gen) buildSchemaGoPayload() *SchemaGoPayload {
	return &SchemaGoPayload{
		PackageName: g.pkgName,
		Schema:      g.buildDomainSchema(),
	}
}

func (g *_Gen) schemaGoTemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"authLiteral":              renderAuthModeLiteral,
		"permissionRequireLiteral": renderPermRequireModeLiteral,
		"quote":                    quote,
		"scalarLiteral":            renderScalarLiteral,
		"viaLiteral":               renderActorViaLiteral,
	}
}
