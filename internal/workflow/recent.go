package workflow

import "strings"

// Bound preference decoding and sorting costs while retaining ample selection history.
const recentProjectLimit = 1000

func recentProjects(ids []string) []string {
	if len(ids) == 0 {
		return nil
	}
	out := make([]string, 0, min(len(ids), recentProjectLimit))
	seen := make(map[string]bool, cap(out))
	for _, id := range ids {
		if len(id) != 32 || strings.Trim(id, "0123456789abcdef") != "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
		if len(out) == recentProjectLimit {
			break
		}
	}
	return out
}

func rememberProject(prefs *Preferences, id string) {
	prefs.Recent = recentProjects(append([]string{id}, prefs.Recent...))
}
