package analyzer

import "go.yorun.ai/skelc/model"

func (p *Analysis) checkConfigMemberTypes(dataList []*model.Data) {
	for _, dataType := range dataList {
		if dataType.Kind != model.DataKindConfig {
			continue
		}
		for _, member := range dataType.Members {
			checkConfigMemberType(p.reporter, dataType, member)
		}
	}
}

func checkConfigMemberType(reporter *diagnosticReporter, dataType *model.Data, member *model.DataMember) bool {
	if member.Type.Kind == model.TypeKindData {
		reporter.reportf("%s config %s member %s cannot reference data %s", member.Pos, dataType.Name, member.Name, member.Type.Data.Name)
		return false
	}

	switch member.Type.Kind {
	case model.TypeKindScalar:
		return reporter.check(member.Type.Scalar != model.ScalarBinary,
			"%s config %s member %s cannot use binary type", member.Pos, dataType.Name, member.Name)
	case model.TypeKindEnum:
		return true
	case model.TypeKindList:
		valid := reporter.check(member.Type.List.Value.Kind == model.TypeKindScalar || member.Type.List.Value.Kind == model.TypeKindEnum,
			"%s config %s member %s list value type must be scalar or enum", member.Pos, dataType.Name, member.Name)
		return reporter.checkNot(member.Type.List.Value.Kind == model.TypeKindScalar && member.Type.List.Value.Scalar == model.ScalarBinary,
			"%s config %s member %s list value cannot use binary type", member.Pos, dataType.Name, member.Name) && valid
	case model.TypeKindMap:
		valid := reporter.check(member.Type.Map.Key.Kind == model.TypeKindScalar || member.Type.Map.Key.Kind == model.TypeKindEnum,
			"%s config %s member %s map key type must be scalar or enum", member.Pos, dataType.Name, member.Name)
		valid = reporter.checkNot(member.Type.Map.Key.Kind == model.TypeKindScalar && member.Type.Map.Key.Scalar == model.ScalarBinary,
			"%s config %s member %s map key cannot use binary type", member.Pos, dataType.Name, member.Name) && valid
		valueScalar := member.Type.Map.Value.Kind == model.TypeKindScalar
		valid = reporter.check(valueScalar, "%s config %s member %s map value type must be scalar", member.Pos, dataType.Name, member.Name) && valid
		if valueScalar {
			valid = reporter.check(member.Type.Map.Value.Scalar != model.ScalarBinary,
				"%s config %s member %s map value cannot use binary type", member.Pos, dataType.Name, member.Name) && valid
		}
		return valid
	default:
		reporter.reportf("%s config %s member %s has unsupported type kind %v",
			member.Pos, dataType.Name, member.Name, member.Type.Kind)
		return false
	}
}
