package schema

func Entries(document *Document) []*Entry {
	entries := make([]*Entry, 0, len(document.Declarations))
	for _, declaration := range document.Declarations {
		entries = append(entries, &Entry{
			Pub: declaration.Pub, Name: declaration.Name, Kind: declaration.Kind, SkelName: declaration.SkelName,
		})
	}
	return entries
}

func Find(document *Document, kind, skelName string) *Declaration {
	for _, declaration := range document.Declarations {
		if declaration.Kind == DeclarationType(kind) && declaration.SkelName == skelName {
			return declaration
		}
	}
	return nil
}
