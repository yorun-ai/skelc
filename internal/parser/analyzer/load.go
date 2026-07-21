package analyzer

import "go.yorun.ai/skelc/internal/util/checkutil"

func (p *Analysis) load() {
	checkutil.CheckNot(p.content == nil || p.content.Domain == nil || p.content.Domain.Name == nil || p.content.Domain.Name.String() == "",
		"missing domain name")
	p.name = p.content.Domain.Name.String()
	p.description = p.content.Domain.Description

	for _, entry := range p.content.Entries {
		switch {
		case entry.Enum != nil:
			enum := parseEnum(entry.Enum)
			p.checkDuplicated(enum.Name, enum.Pos)
			enum.Domain = p.name
			enum.SkelName = p.skelName(enum.Name)
			p.enumsMap[enum.Name] = enum

		case entry.Data != nil:
			data := parseData(entry.Data)
			p.checkDuplicated(data.Name, data.Pos)
			data.Domain = p.name
			data.SkelName = p.skelName(data.Name)
			p.dataMap[data.Name] = data

		case entry.Config != nil:
			config := parseConfig(entry.Config)
			p.checkDuplicated(config.Name, config.Pos)
			config.Domain = p.name
			config.SkelName = p.skelName(config.Name)
			p.dataMap[config.Name] = config

		case entry.Actor != nil:
			actor := parseActor(entry.Actor)
			p.checkDuplicated(actor.Name, actor.Pos)
			actor.SkelName = p.skelName(actor.Name)
			p.actorsMap[actor.Name] = actor
			if actor.AuthEnabled {
				actor.AuthCredential.Domain = p.name
				actor.AuthCredential.SkelName = p.skelName(actor.AuthCredential.Name)
				actor.AuthInfo.Domain = p.name
				actor.AuthInfo.SkelName = p.skelName(actor.AuthInfo.Name)
			}
			actor.AuthService = buildActorAuthService(actor)
			actor.PermService = buildActorPermissionService(actor)

		case entry.Resource != nil:
			resource := parseResource(entry.Resource, entry.Pub)
			p.checkDuplicatedResource(resource.Name, resource.Pos)
			resource.SkelName = p.skelName(resource.Name)
			for _, action := range resource.Actions {
				action.PermissionCode = resource.SkelName + ":" + action.Name
			}
			resource.CheckService = buildResourceCheckService(p.name, resource)
			p.resourcesMap[resource.Name] = resource

		case entry.Event != nil:
			event := parseEvent(entry.Event)
			p.checkDuplicated(event.Name, event.Pos)
			event.Domain = p.name
			event.SkelName = p.skelName(event.Name)
			p.dataMap[event.Name] = event

		case entry.Service != nil:
			service := parseService(entry.Service)
			p.checkDuplicated(service.Name, service.Pos)
			service.SkelName = p.skelName(service.Name)
			p.servicesMap[service.Name] = service

		case entry.Web != nil:
			web := parseWeb(entry.Web, entry.Pub)
			p.checkDuplicated(web.Name, web.Pos)
			web.SkelName = p.skelName(web.Name)
			p.websMap[web.Name] = web

		case entry.Task != nil:
			checkutil.CheckNot(entry.Pub, "%s task %s does not support pub", entry.Task.Name.Pos, entry.Task.Name.Value)
			task := parseTask(entry.Task)
			p.checkDuplicated(task.Name, task.Pos)
			task.SkelName = p.skelName(task.Name)
			p.tasksMap[task.Name] = task
		}
	}
	p.checkActorGeneratedNames()
}
