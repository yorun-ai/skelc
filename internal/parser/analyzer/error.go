package analyzer

import "fmt"

type MissingImportError struct {
	Position string
	Domain   string
}

func (e *MissingImportError) Error() string {
	return fmt.Sprintf("%s skel import %s not found; pass --skel-import %s=PATH", e.Position, e.Domain, e.Domain)
}
