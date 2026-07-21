package analyzer

import (
	"fmt"

	"go.yorun.ai/skelc/internal/util/checkutil"
	"go.yorun.ai/skelc/model"
)

func (p *Analysis) checkConfigMemberTypes(dataList []*model.Data) {
	for _, dataType := range dataList {
		if dataType.Kind != model.DataKindConfig {
			continue
		}
		for _, member := range dataType.Members {
			checkConfigMemberType(dataType, member)
		}
	}
}

func checkConfigMemberType(dataType *model.Data, member *model.DataMember) {
	checkutil.CheckFunc(member.Type.Kind != model.TypeKindData, func() string {
		return fmt.Sprintf("%s config %s member %s cannot reference data %s",
			member.Pos, dataType.Name, member.Name, member.Type.Data.Name)
	})

	switch member.Type.Kind {
	case model.TypeKindScalar:
		checkutil.Check(member.Type.Scalar != model.ScalarBinary,
			"%s config %s member %s cannot use binary type", member.Pos, dataType.Name, member.Name)
		return
	case model.TypeKindEnum:
		return
	case model.TypeKindList:
		checkutil.Check(member.Type.List.Value.Kind == model.TypeKindScalar || member.Type.List.Value.Kind == model.TypeKindEnum,
			"%s config %s member %s list value type must be scalar or enum", member.Pos, dataType.Name, member.Name)
		checkutil.CheckNot(member.Type.List.Value.Kind == model.TypeKindScalar && member.Type.List.Value.Scalar == model.ScalarBinary,
			"%s config %s member %s list value cannot use binary type", member.Pos, dataType.Name, member.Name)
		return
	case model.TypeKindMap:
		checkutil.Check(member.Type.Map.Key.Kind == model.TypeKindScalar || member.Type.Map.Key.Kind == model.TypeKindEnum,
			"%s config %s member %s map key type must be scalar or enum", member.Pos, dataType.Name, member.Name)
		checkutil.CheckNot(member.Type.Map.Key.Kind == model.TypeKindScalar && member.Type.Map.Key.Scalar == model.ScalarBinary,
			"%s config %s member %s map key cannot use binary type", member.Pos, dataType.Name, member.Name)
		checkutil.Check(member.Type.Map.Value.Kind == model.TypeKindScalar,
			"%s config %s member %s map value type must be scalar", member.Pos, dataType.Name, member.Name)
		checkutil.Check(member.Type.Map.Value.Scalar != model.ScalarBinary,
			"%s config %s member %s map value cannot use binary type", member.Pos, dataType.Name, member.Name)
		return
	default:
		checkutil.Panicf("%s config %s member %s has unsupported type kind %v",
			member.Pos, dataType.Name, member.Name, member.Type.Kind)
	}
}
