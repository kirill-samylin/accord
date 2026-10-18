package interchange

import "strings"

// Shared by parse/export and typed references so identity delimiters cannot
// appear in identifiers, including references resolved by a source mapping.
func validIdentity(kind, id string) bool {
	return kind != "" && id != "" && len(id) <= 256 && !strings.ContainsRune(id, '\x00')
}

func validRef(ref Ref, allowed ...string) bool {
	if !validIdentity(ref.Kind, ref.ExternalID) {
		return false
	}
	for _, kind := range allowed {
		if ref.Kind == kind {
			return true
		}
	}
	return false
}
