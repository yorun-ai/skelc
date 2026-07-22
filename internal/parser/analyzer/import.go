package analyzer

import (
	"fmt"
	"strings"

	"go.yorun.ai/skelc/internal/util/checkutil"
	"go.yorun.ai/skelc/model"
)

func (p *Analysis) skelName(name string) string {
	return fmt.Sprintf("%s.%s", p.name, name)
}

func (p *Analysis) loadImports(domainByName map[string]*Analysis) {
	if len(p.content.Imports) == 0 {
		return
	}
	for _, grammarImport := range p.content.Imports {
		domainName := grammarImport.Domain.String()
		alias := defaultImportAlias(domainName)
		if grammarImport.Alias != nil {
			alias = grammarImport.Alias.Value
		}
		importedDomain := domainByName[domainName]
		if importedDomain == nil {
			panic(&MissingImportError{Position: position(grammarImport.Pos), Domain: domainName})
		}
		if prev, exists := p.importsMap[alias]; exists {
			checkutil.Check(prev.Model.Name == domainName,
				"%s duplicated import alias %s found, already used by %s",
				grammarImport.Pos, alias, prev.Model.Name,
			)
			continue
		}
		importModel := &model.Import{
			Pos:           position(grammarImport.Pos),
			Domain:        importedDomain.Model(),
			Name:          domainName,
			Alias:         alias,
			ExplicitAlias: grammarImport.Alias != nil,
		}
		import_ := &domainImport{Domain: importedDomain, Model: importModel}
		p.importsMap[alias] = import_
		p.imports = append(p.imports, importModel)
	}
}

func defaultImportAlias(domainName string) string {
	parts := strings.Split(domainName, ".")
	return parts[len(parts)-1]
}

func (p *Analysis) checkDuplicated(name string, namePos model.Position) {
	errMsg := `%s duplicated identifier "%s" found, also present at %s`
	previousEnum, exists := p.enumsMap[name]
	checkutil.CheckFuncAt(namePos, !exists, func() string {
		return fmt.Sprintf(errMsg, namePos, name, previousEnum.Pos)
	})
	previousData, exists := p.dataMap[name]
	checkutil.CheckFuncAt(namePos, !exists, func() string {
		return fmt.Sprintf(errMsg, namePos, name, previousData.Pos)
	})
	previousActor, exists := p.actorsMap[name]
	checkutil.CheckFuncAt(namePos, !exists, func() string {
		return fmt.Sprintf(errMsg, namePos, name, previousActor.Pos)
	})
	previousService, exists := p.servicesMap[name]
	checkutil.CheckFuncAt(namePos, !exists, func() string {
		return fmt.Sprintf(errMsg, namePos, name, previousService.Pos)
	})
	previousWeb, exists := p.websMap[name]
	checkutil.CheckFuncAt(namePos, !exists, func() string {
		return fmt.Sprintf(errMsg, namePos, name, previousWeb.Pos)
	})
	previousTask, exists := p.tasksMap[name]
	checkutil.CheckFuncAt(namePos, !exists, func() string {
		return fmt.Sprintf(errMsg, namePos, name, previousTask.Pos)
	})
}

func (p *Analysis) checkDuplicatedResource(name string, namePos model.Position) {
	previous, exists := p.resourcesMap[name]
	checkutil.CheckFuncAt(namePos, !exists, func() string {
		return fmt.Sprintf(`%s duplicated resource "%s" found, also present at %s`, namePos, name, previous.Pos)
	})
}

func (p *Analysis) checkActorGeneratedNames() {
	generated := map[string]model.Position{}
	for _, actor := range p.actorsMap {
		if actor.AuthEnabled {
			p.checkGeneratedIdentifier(actor.AuthCredential.Name, actor.AuthCredential.Pos, generated)
			p.checkGeneratedIdentifier(actor.AuthInfo.Name, actor.AuthInfo.Pos, generated)
			p.checkGeneratedIdentifier(actor.AuthService.Name, actor.Pos, generated)
		}
		if actor.PermService != nil {
			p.checkGeneratedIdentifier(actor.PermService.Name, actor.Pos, generated)
		}
	}
	for _, resource := range p.resourcesMap {
		if resource.CheckService != nil {
			p.checkGeneratedIdentifier(resource.CheckService.Name, resource.Pos, generated)
		}
	}
}

func (p *Analysis) checkGeneratedIdentifier(name string, namePos model.Position, generated map[string]model.Position) {
	p.checkDuplicated(name, namePos)
	duplicatedPosition, duplicated := generated[name]
	checkutil.CheckFuncAt(namePos, !duplicated, func() string {
		return fmt.Sprintf(`%s duplicated identifier "%s" found, also present at %s`, namePos, name, duplicatedPosition)
	})
	generated[name] = namePos
}
