package analyzer

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"go.yorun.ai/skelc/model"
)

func (p *Analysis) skelName(name string) string {
	return fmt.Sprintf("%s.%s", p.name, name)
}

func (p *Analysis) loadImports(domainByName map[string]*Analysis) {
	for _, grammarImport := range p.content.Imports {
		domainName := grammarImport.Domain.String()
		alias := defaultImportAlias(domainName)
		if grammarImport.Alias != nil {
			alias = grammarImport.Alias.Value
		}
		importedDomain := domainByName[domainName]
		if importedDomain == nil {
			p.reporter.report(&MissingImportError{Position: position(grammarImport.Pos), Domain: domainName})
			continue
		}
		if previous, exists := p.importsMap[alias]; exists {
			p.reporter.check(previous.Model.Name == domainName,
				"%s duplicated import alias %s found, already used by %s",
				grammarImport.Pos, alias, previous.Model.Name)
			continue
		}
		importModel := &model.Import{
			Pos:           position(grammarImport.Pos),
			Domain:        importedDomain.Model(),
			Name:          domainName,
			Alias:         alias,
			ExplicitAlias: grammarImport.Alias != nil,
		}
		p.importsMap[alias] = &domainImport{Domain: importedDomain, Model: importModel}
		p.imports = append(p.imports, importModel)
	}
}

func defaultImportAlias(domainName string) string {
	parts := strings.Split(domainName, ".")
	return parts[len(parts)-1]
}

func (p *Analysis) checkDuplicated(name string, namePos model.Position) bool {
	message := `%s duplicated identifier "%s" found, also present at %s`
	valid := true
	if previous := p.enumsMap[name]; previous != nil {
		p.reporter.reportf(message, namePos, name, previous.Pos)
		valid = false
	}
	if previous := p.dataMap[name]; previous != nil {
		p.reporter.reportf(message, namePos, name, previous.Pos)
		valid = false
	}
	if previous := p.actorsMap[name]; previous != nil {
		p.reporter.reportf(message, namePos, name, previous.Pos)
		valid = false
	}
	if previous := p.servicesMap[name]; previous != nil {
		p.reporter.reportf(message, namePos, name, previous.Pos)
		valid = false
	}
	if previous := p.websMap[name]; previous != nil {
		p.reporter.reportf(message, namePos, name, previous.Pos)
		valid = false
	}
	if previous := p.tasksMap[name]; previous != nil {
		p.reporter.reportf(message, namePos, name, previous.Pos)
		valid = false
	}
	return valid
}

func (p *Analysis) checkDuplicatedResource(name string, namePos model.Position) bool {
	if previous := p.resourcesMap[name]; previous != nil {
		p.reporter.reportf(`%s duplicated resource "%s" found, also present at %s`, namePos, name, previous.Pos)
		return false
	}
	return true
}

func (p *Analysis) checkActorGeneratedNames() {
	generated := map[string]model.Position{}
	for _, name := range slices.Sorted(maps.Keys(p.actorsMap)) {
		actor := p.actorsMap[name]
		if actor.AuthEnabled {
			p.checkGeneratedIdentifier(actor.AuthCredential.Name, actor.AuthCredential.Pos, generated)
			p.checkGeneratedIdentifier(actor.AuthInfo.Name, actor.AuthInfo.Pos, generated)
			p.checkGeneratedIdentifier(actor.AuthService.Name, actor.Pos, generated)
		}
		if actor.PermService != nil {
			p.checkGeneratedIdentifier(actor.PermService.Name, actor.Pos, generated)
		}
	}
	for _, name := range slices.Sorted(maps.Keys(p.resourcesMap)) {
		resource := p.resourcesMap[name]
		if resource.CheckService != nil {
			p.checkGeneratedIdentifier(resource.CheckService.Name, resource.Pos, generated)
		}
	}
}

func (p *Analysis) checkGeneratedIdentifier(name string, namePos model.Position, generated map[string]model.Position) {
	valid := p.checkDuplicated(name, namePos)
	if previous, duplicated := generated[name]; duplicated {
		p.reporter.reportf(`%s duplicated identifier "%s" found, also present at %s`, namePos, name, previous)
		valid = false
	}
	if valid {
		generated[name] = namePos
	}
}
