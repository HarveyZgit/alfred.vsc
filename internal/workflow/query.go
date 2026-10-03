package workflow

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

type Icon struct {
	Path string `json:"path"`
}
type Modifier struct {
	Arg       string            `json:"arg,omitempty"`
	Subtitle  string            `json:"subtitle"`
	Variables map[string]string `json:"variables,omitempty"`
	Valid     bool              `json:"valid"`
}
type Item struct {
	UID          string              `json:"uid,omitempty"`
	Title        string              `json:"title"`
	Subtitle     string              `json:"subtitle"`
	Arg          string              `json:"arg,omitempty"`
	Valid        bool                `json:"valid"`
	Autocomplete string              `json:"autocomplete,omitempty"`
	Icon         *Icon               `json:"icon,omitempty"`
	Variables    map[string]string   `json:"variables,omitempty"`
	Mods         map[string]Modifier `json:"mods,omitempty"`
	Text         map[string]string   `json:"text,omitempty"`
}
type Feedback struct {
	Items    []Item   `json:"items"`
	Rerun    float64  `json:"rerun,omitempty"`
	Warnings []string `json:"_warnings,omitempty"`
}

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
		return a.position < b.position
	})
	out := make([]Project, 0, min(limit, len(matches)))
	for _, m := range matches[:min(limit, len(matches))] {
		out = append(out, projects[m.position])
	}
	return out
}
func requestJSON(target, editor string) string {
	b, _ := json.Marshal(OpenRequest{Target: target, Editor: editor})
	return string(b)
}
func render(c Config, projects []Project, prefs Preferences) []Item {
	statuses := visibleBranches(projects)
	items := []Item{}
	for _, p := range projects {
		editor := c.Editor
		if prefs.Editors[p.ID] != "" {
			editor = prefs.Editors[p.ID]
		}
		label := p.Path
		icon := "assets/folder.png"
		valid := true
		if p.Kind == "remote" {
			label = remoteLabel(p) + " · " + p.Path
			icon = "assets/remote.png"
		} else {
			status := statuses[p.ID]
			if strings.HasPrefix(label, c.Home+string(filepath.Separator)) {
				label = "~" + strings.TrimPrefix(label, c.Home)
			}
			if status.missing {
				label = "目录不可用 · " + label
				valid = false
			} else if status.timedOut {
				label = "分支读取超时 · " + label
			} else if status.branch != "" {
				label = "⎇ " + status.branch + " · " + label
			}
		}
		title := p.Name
		if prefs.Pinned[p.ID] {
			title = "★ " + title
		}
		items = append(items, Item{UID: p.ID, Title: title, Subtitle: label + " · " + editor, Arg: requestJSON(p.URI, editor), Valid: valid, Icon: &Icon{icon}, Variables: map[string]string{"vsc_target": p.URI}, Text: map[string]string{"copy": p.URI, "largetype": label}, Mods: map[string]Modifier{
			"cmd": {Subtitle: "选择打开此目录的 IDE", Valid: valid}, "ctrl": {Subtitle: "隐藏此项目（不删除文件）", Valid: true}, "alt": {Subtitle: "固定 / 取消固定此项目", Valid: true}, "shift": {Subtitle: "在新窗口打开", Arg: func() string {
				b, _ := json.Marshal(OpenRequest{Target: p.URI, Editor: editor, NewWindow: true})
				return string(b)
			}(), Valid: valid},
		}})
	}
	return items
}
func Query(c Config, query string) (Feedback, error) {
	if e := migrate(c); e != nil {
		return Feedback{}, e
	}
	prefs, e := loadPreferences(c)
	if e != nil {
		return Feedback{}, e
	}
	if strings.HasPrefix(strings.TrimSpace(query), "/") {
		return browse(c, strings.TrimSpace(query), prefs)
	}
	history, warnings := loadHistory(c)
	index, stale := loadIndex(c)
	pending := false
	if stale {
		pending = refreshIndex(c)
	}
	warnings = append(warnings, index.Warnings...)
	projects := rank(merge(history, index.Projects), prefs, query, c.Limit)
	feedback := Feedback{Items: render(c, projects, prefs), Warnings: warnings}
	if pending {
		feedback.Rerun = 0.5
	}
	if len(feedback.Items) == 0 {
		message := "尝试项目名称、路径或主机名"
		title := "没有匹配的目录"
		if len(history) == 0 && len(index.Projects) == 0 {
			title = "尚未发现项目"
			message = "在工作流配置中设置项目根目录，或先用 VS Code 打开目录"
		}
		if pending {
			title = "正在发现 Git 仓库…"
		}
		feedback.Items = append(feedback.Items, Item{Title: title, Subtitle: message, Valid: false})
	}
	return feedback, nil
}
func browse(c Config, query string, prefs Preferences) (Feedback, error) {
	// Root selection is explicit when several roots exist, so same-name folders
	// never get merged into one autocomplete navigation state.
	rest := strings.TrimPrefix(query, "/")
	roots := c.Roots
	prefix := "/"
	root := ""
	if len(roots) == 0 {
		return Feedback{Items: []Item{{Title: "请配置项目根目录", Subtitle: "VSC_DIRECTORIES", Valid: false}}}, nil
	}
	if len(roots) > 1 {
		parts := strings.SplitN(rest, "/", 2)
		if len(parts) == 1 {
			items := []Item{}
			for i, r := range roots {
				key := fmt.Sprintf("%d", i+1)
				if rest == "" || strings.Contains(strings.ToLower(r), strings.ToLower(rest)) || rest == key {
					items = append(items, Item{Title: filepath.Base(r), Subtitle: r, Valid: false, Autocomplete: "/" + key + "/"})
				}
			}
			return Feedback{Items: items}, nil
		}
		for i, r := range roots {
			if parts[0] == fmt.Sprintf("%d", i+1) {
				root = r
				prefix = "/" + parts[0] + "/"
				rest = parts[1]
				break
			}
		}
		if root == "" {
			return Feedback{Items: []Item{{Title: "请选择项目根目录", Valid: false, Autocomplete: "/"}}}, nil
		}
	} else {
		root = roots[0]
	}
	parent, search := filepath.Split(rest)
	candidate := filepath.Join(root, parent)
	realRoot, e := filepath.EvalSymlinks(root)
	if e != nil {
		return Feedback{}, e
	}
	real, e := filepath.EvalSymlinks(candidate)
	if e != nil {
		return Feedback{}, e
	}
	rel, e := filepath.Rel(realRoot, real)
	if e != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return Feedback{}, fmt.Errorf("browse path must remain inside the configured root")
	}
	entries, e := os.ReadDir(real)
	if e != nil {
		return Feedback{}, e
	}
	projects := []Project{}
	completions := map[string]string{}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") || skipDirs[entry.Name()] {
			continue
		}
		name := filepath.Join(real, entry.Name())
		info, e := os.Stat(name)
		if e != nil || !info.IsDir() {
			continue
		}
		if _, ok := fuzzy(search, entry.Name()); !ok {
			continue
		}
		p, e := localProject(name)
		if e == nil {
			projects = append(projects, p)
			completions[p.ID] = prefix + filepath.ToSlash(parent) + entry.Name() + "/"
		}
	}
	items := render(c, projects[:min(len(projects), c.Limit)], prefs)
	for i := range items {
		items[i].Autocomplete = completions[items[i].UID]
	}
	if parent != "" {
		up := filepath.ToSlash(filepath.Dir(strings.TrimSuffix(parent, string(filepath.Separator))))
		if up == "." {
			up = ""
		}
		if up != "" {
			up += "/"
		}
		items = append(items, Item{Title: "返回上级", Subtitle: real, Valid: false, Autocomplete: prefix + up})
	}
	if len(items) == 0 {
		items = append(items, Item{Title: "没有匹配的子目录", Valid: false, Autocomplete: prefix})
	}
	return Feedback{Items: items}, nil
}
