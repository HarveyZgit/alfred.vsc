package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

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
