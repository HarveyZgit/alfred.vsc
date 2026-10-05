package workflow

import (
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

func fuzzy(query, key string) (int, bool) {
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
	q := []rune(query)
	k := []rune(key)
	index, total, previous := 0, 0, -2
	for i, ch := range k {
		if index < len(q) && ch == q[index] {
			if previous == i-1 {
				total += 10
			} else {
				total++
			}
			if i == 0 || strings.ContainsRune("-_/ ", k[i-1]) {
				total += 5
			}
			index++
			previous = i
		}
	}
	return total + 100 - min(len(k), 100), index == len(q)
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
	tokens := strings.FieldsFunc(strings.ToLower(query), unicode.IsSpace)
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
	for i, p := range projects {
		if prefs.Hidden[p.ID] || kind != "" && p.Kind != kind {
			continue
		}
		total := 0
		ok := true
		key := p.Name + " " + p.Path + " " + p.Authority + " " + p.Label
		for _, token := range tokens {
			value, found := fuzzy(token, key)
			if !found {
				ok = false
				break
			}
			nameScore, nameMatch := fuzzy(token, p.Name)
			if nameMatch {
				value = max(value, nameScore+1000)
			}
			total += value
		}
		if ok {
			matches = append(matches, match{total, i, prefs.Pinned[p.ID]})
		}
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
