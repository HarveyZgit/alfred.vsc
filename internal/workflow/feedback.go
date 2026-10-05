package workflow

import (
	"encoding/json"
	"path/filepath"
	"strings"
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
				label = status.reason + " · " + label
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
		items = append(items, Item{
			UID:       p.ID,
			Title:     title,
			Subtitle:  label + " · " + editor,
			Arg:       requestJSON(p.URI, editor),
			Valid:     valid,
			Icon:      &Icon{icon},
			Variables: map[string]string{"vsc_target": p.URI},
			Text:      map[string]string{"copy": p.URI, "largetype": label},
			Mods: map[string]Modifier{
				"cmd":  {Subtitle: "选择打开此目录的 IDE", Valid: valid},
				"ctrl": {Subtitle: "隐藏此项目（不删除文件）", Valid: true},
				"alt":  {Subtitle: "固定 / 取消固定此项目", Valid: true},
				"shift": {
					Subtitle: "在新窗口打开",
					Arg: func() string {
						b, _ := json.Marshal(OpenRequest{Target: p.URI, Editor: editor, NewWindow: true})
						return string(b)
					}(),
					Valid: valid,
				},
			},
		})
	}
	return items
}
