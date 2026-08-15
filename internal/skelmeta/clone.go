package skelmeta

import "go.yorun.ai/skelc/internal/util/nameutil"

const (
	CloneMethodName   = "Clone"
	CloneByMethodName = "CloneBy"
)

// CloneMethodFieldNames returns the Skel field names that would collide with
// generated clone methods after Go name conversion.
func CloneMethodFieldNames() []string {
	return []string{
		nameutil.ToLowerCamel(CloneMethodName),
		nameutil.ToLowerCamel(CloneByMethodName),
	}
}
