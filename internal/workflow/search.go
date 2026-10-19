package workflow

import (
	"sort"
	"strings"
	"unicode/utf8"
)

func substringScore(query, key string) (int, bool) {
	query = strings.ToLower(query)
	key = strings.ToLower(key)
	if query == "" {
		return 0, true
	}
	if strings.Contains(key, query) {
		base := 500
		if strings.HasPrefix(key, query) {
			base = 1000
		}
		return base + 100 - min(utf8.RuneCountInString(key), 100), true
	}
	return 0, false
}

func containsWords(query, key string) bool {
	key = strings.ToLower(key)
	for _, token := range strings.Fields(strings.ToLower(query)) {
		if !strings.Contains(key, token) {
			return false
		}
	}
	return true
}

func parseQuery(query string) (string, string) {
	query = strings.TrimSpace(query)
	fields := strings.Fields(query)
	if len(fields) > 0 {
		switch fields[0] {
		case "d", "dir", "folder", ":local":
			return "folder", strings.TrimSpace(strings.TrimPrefix(query, fields[0]))
		case "r", "remote", ":remote":
			return "remote", strings.TrimSpace(strings.TrimPrefix(query, fields[0]))
		}
	}
	return "", query
}

func rank(projects []Project, prefs Preferences, query string, limit int) []Project {
	kind, query := parseQuery(query)
	tokens := strings.Fields(strings.ToLower(query))
	type match struct {
		score    int
		position int
		pinned   bool
	}
	legacy := make(map[string]int, len(prefs.LegacyOrder))
	for i, id := range prefs.LegacyOrder {
		legacy[id] = i + 1
	}
	matches := make([]match, 0, len(projects))
	var exactMatches []match
	for i, p := range projects {
		if prefs.Hidden[p.ID] || kind != "" && p.Kind != kind {
			continue
		}
		// A complete name narrows the result set before ranking or limiting it.
		// Keep all eligible namesakes so local and remote copies remain selectable.
		if query != "" && strings.EqualFold(query, p.Name) {
			exactMatches = append(exactMatches, match{position: i, pinned: prefs.Pinned[p.ID]})
			continue
		}
		if len(exactMatches) > 0 {
			continue
		}
		total := 0
		ok := true
		key := p.Name + " " + p.Path + " " + p.Authority + " " + p.Label
		for _, token := range tokens {
			value, found := substringScore(token, key)
			if !found {
				ok = false
				break
			}
			nameScore, nameMatch := substringScore(token, p.Name)
			if nameMatch {
				value = max(value, nameScore+1000)
			}
			total += value
		}
		if ok {
			matches = append(matches, match{total, i, prefs.Pinned[p.ID]})
		}
	}
	if len(exactMatches) > 0 {
		matches = exactMatches
	}
	sort.SliceStable(matches, func(i, j int) bool {
		a, b := matches[i], matches[j]
		if a.score != b.score {
			return a.score > b.score
		}
		if a.pinned != b.pinned {
			return a.pinned
		}
		ar, br := legacy[projects[a.position].ID], legacy[projects[b.position].ID]
		if ar != 0 && br != 0 && ar != br {
			return ar < br
		}
		if (ar != 0) != (br != 0) {
			return ar != 0
		}
		return a.position < b.position
	})
	out := make([]Project, 0, min(limit, len(matches)))
	for _, m := range matches[:min(limit, len(matches))] {
		out = append(out, projects[m.position])
	}
	return out
}
