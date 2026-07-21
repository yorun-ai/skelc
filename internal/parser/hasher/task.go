package hasher

import (
	"fmt"
	"strings"

	"go.yorun.ai/skelc/model"
)

func (s *_hashState) triggerHash(trigger *model.TaskTrigger) string {
	return hashValue(_TriggerHashValue{
		Name:             trigger.Name,
		SkelName:         trigger.SkelName,
		Description:      trigger.Description,
		Example:          trigger.Example,
		InputDescription: trigger.InputDescription,
		Arguments:        s.buildArgumentHashValues(trigger.Arguments),
	})
}

func (s *_hashState) taskHash(task *model.Task) string {
	return s.memoHash("task", task.SkelName, func() string {
		for _, trigger := range task.Triggers {
			trigger.Hash = s.triggerHash(trigger)
		}
		return hashValue(_TaskHashValue{
			Name:        task.Name,
			SkelName:    task.SkelName,
			Description: task.Description,
			Triggers: buildNamedValues(task.Triggers,
				func(trigger *model.TaskTrigger) string { return trigger.SkelName },
				func(trigger *model.TaskTrigger) string { return trigger.Hash }),
		})
	})
}

func (s *_hashState) memoHash(kind string, skelName string, build func() string) string {
	key := fmt.Sprintf("%s:%s", kind, skelName)
	if hash, ok := s.cache[key]; ok {
		return hash
	}
	if s.status[key] {
		return cycleHash(kind, skelName)
	}
	s.status[key] = true
	hash := build()
	s.status[key] = false
	s.cache[key] = hash
	return hash
}

func (s *_hashState) buildActorAudienceHashValues(audiences []*model.ActorAudience) []*_ActorRefHashValue {
	values := make([]*_ActorRefHashValue, 0, len(audiences))
	for _, audience := range audiences {
		name, skelName := s.actorRefNames(audience.Actor)
		hash := ""
		if actor := s.actorBySkel[skelName]; actor != nil {
			hash = s.actorHash(actor)
		}
		values = append(values, &_ActorRefHashValue{
			Name:     name,
			SkelName: skelName,
			Hash:     hash,
			Via:      audience.Via,
		})
	}
	return values
}

func (s *_hashState) actorRefNames(actorName string) (string, string) {
	if alias, baseName, ok := strings.Cut(actorName, "."); ok {
		for _, import_ := range s.domain.Imports() {
			if import_.Alias == alias {
				return baseName, import_.Name + "." + baseName
			}
		}
	}
	return actorName, s.domain.Name() + "." + actorName
}
