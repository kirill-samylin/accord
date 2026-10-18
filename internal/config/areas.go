package config

import (
	"fmt"
	"strings"
)

// CanonicalArea validates relative POSIX areas without consulting the filesystem.
func CanonicalArea(value string) (string, error) {
	value = strings.TrimSuffix(value, "/")
	if value == "" || strings.ContainsAny(value, "\\\x00:") || strings.HasPrefix(value, "/") {
		return "", fmt.Errorf("area must be a relative POSIX path")
	}
	for _, part := range strings.Split(value, "/") {
		if part == "" || part == "." || part == ".." {
			return "", fmt.Errorf("area contains an empty or traversal segment")
		}
	}
	return value, nil
}
func validateAreas(c *Config, r *Result) {
	occupied := map[string]bool{}
	for i := range c.Areas {
		area := &c.Areas[i]
		p := "areas"
		name, err := CanonicalArea(area.Name)
		if err != nil {
			add(r, p, "CONFIG_INVALID", err.Error())
			continue
		}
		area.Name = name
		all := append([]string{name}, area.Aliases...)
		for j, item := range all {
			canonical, err := CanonicalArea(item)
			if err != nil {
				add(r, p, "CONFIG_INVALID", err.Error())
				continue
			}
			if occupied[canonical] {
				add(r, p, "CONFIG_INVALID", "duplicate or colliding area name/alias")
			}
			occupied[canonical] = true
			if j > 0 {
				area.Aliases[j-1] = canonical
			}
		}
		if area.SingleEditorRole != "" {
			if _, ok := c.Roles[area.SingleEditorRole]; !ok {
				add(r, p+".single_editor_role", "CONFIG_INVALID", "unknown role")
			}
		}
	}
}

// ResolveArea resolves exact aliases first and then declared canonical children.
func (s *Snapshot) ResolveArea(value string) (string, error) {
	path, err := CanonicalArea(value)
	if err != nil {
		return "", err
	}
	c := s.Decode()
	for _, a := range c.Areas {
		for _, alias := range a.Aliases {
			if path == alias {
				return a.Name, nil
			}
		}
	}
	for _, a := range c.Areas {
		if path == a.Name || strings.HasPrefix(path, a.Name+"/") {
			return path, nil
		}
	}
	return "", fmt.Errorf("area is not declared")
}

// AreasIntersect uses path segments; textual prefix similarity is insufficient.
func AreasIntersect(a, b string) bool {
	return a == b || strings.HasPrefix(a, b+"/") || strings.HasPrefix(b, a+"/")
}
