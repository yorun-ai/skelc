package common

import "go.yorun.ai/skelc/model"

// TypeVisitor observes one type during a pre-order walk. Returning an error
// stops the walk immediately.
type TypeVisitor func(*model.Type) error

// WalkType visits a type and its structural children: list elements, map keys
// and values, and generic type arguments. Shared type nodes are visited once.
func WalkType(type_ *model.Type, visit TypeVisitor) error {
	return walkType(type_, visit, false, map[*model.Type]bool{}, nil)
}

// WalkTypeGraph additionally follows members of referenced data declarations.
// It is intended for graph-wide questions such as wire-schema discovery and
// safely terminates on recursive data definitions.
func WalkTypeGraph(type_ *model.Type, visit TypeVisitor) error {
	return walkType(type_, visit, true, map[*model.Type]bool{}, map[*model.Data]bool{})
}

func walkType(type_ *model.Type, visit TypeVisitor, followData bool, seenTypes map[*model.Type]bool, seenData map[*model.Data]bool) error {
	if type_ == nil || seenTypes[type_] {
		return nil
	}
	seenTypes[type_] = true
	if err := visit(type_); err != nil {
		return err
	}
	switch type_.Kind {
	case model.TypeKindList:
		if type_.List != nil {
			return walkType(type_.List.Value, visit, followData, seenTypes, seenData)
		}
	case model.TypeKindMap:
		if type_.Map != nil {
			if err := walkType(type_.Map.Key, visit, followData, seenTypes, seenData); err != nil {
				return err
			}
			return walkType(type_.Map.Value, visit, followData, seenTypes, seenData)
		}
	case model.TypeKindData:
		for _, argument := range type_.TypeArguments {
			if err := walkType(argument, visit, followData, seenTypes, seenData); err != nil {
				return err
			}
		}
		if followData && type_.Data != nil && !seenData[type_.Data] {
			seenData[type_.Data] = true
			for _, member := range type_.Data.Members {
				if member != nil {
					if err := walkType(member.Type, visit, followData, seenTypes, seenData); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}
