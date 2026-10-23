package worker

// Codex's typed effective profile adds null defaults. Only map entries with
// null values are absent settings; meaningful values and container types stay
// intact. Copying maps leaves the raw layer projection/digest unchanged.
func normalizeEffectiveProfile(value any) any {
	if source, ok := value.(map[string]any); ok {
		result := make(map[string]any, len(source))
		for key, nested := range source {
			if nested != nil {
				result[key] = normalizeEffectiveProfile(nested)
			}
		}
		return result
	}
	return value
}
