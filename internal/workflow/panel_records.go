package workflow

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

type panelRecord struct {
	Project
	Hidden bool   `json:"hidden"`
	Pinned bool   `json:"pinned"`
	Editor string `json:"editor"`
	Status string `json:"status"`
	Branch string `json:"branch,omitempty"`
}

func panelRecords(c Config, r *http.Request) (any, error) {
	prefs, err := loadPreferences(c)
	if err != nil {
		return nil, err
	}
	history, warnings := loadHistory(c)
	idx, stale := loadIndex(c)
	projects := merge(history, idx.Projects)
	query := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	filter := r.URL.Query().Get("filter")
	filtered := make([]Project, 0, len(projects))
	hidden, pinned := 0, 0
	for _, item := range projects {
		if prefs.Hidden[item.ID] {
			hidden++
		}
		if prefs.Pinned[item.ID] {
			pinned++
		}
		if filter == "hidden" && !prefs.Hidden[item.ID] || filter == "pinned" && !prefs.Pinned[item.ID] || filter == "remote" && item.Kind != "remote" || filter == "local" && item.Kind != "folder" {
			continue
		}
		if query != "" && !strings.Contains(strings.ToLower(item.Name+" "+item.Path+" "+item.Authority), query) {
			continue
		}
		filtered = append(filtered, item)
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 0 {
		page = 0
	}
	pages := max(1, (len(filtered)+49)/50)
	if page >= pages {
		page = pages - 1
	}
	start := page * 50
	end := min(start+50, len(filtered))
	statuses := visibleBranches(filtered[start:end])
	rows := make([]panelRecord, 0, end-start)
	for _, item := range filtered[start:end] {
		row := panelRecord{Project: item, Hidden: prefs.Hidden[item.ID], Pinned: prefs.Pinned[item.ID], Editor: prefs.Editors[item.ID], Status: "可用"}
		if item.Kind == "remote" {
			row.Status = "远程 · 未探测"
		} else {
			s := statuses[item.ID]
			row.Branch = s.branch
			if s.timedOut {
				row.Status = "读取超时"
			} else if s.missing {
				row.Status = s.reason
			}
		}
		rows = append(rows, row)
	}
	return map[string]any{"items": rows, "total": len(filtered), "all": len(projects), "hidden": hidden, "pinned": pinned, "page": page, "pages": pages, "warnings": warnings, "index_stale": stale, "data_dir": c.DataDir}, nil
}

func panelChangeRecord(c Config, w http.ResponseWriter, r *http.Request) (any, error) {
	var err error
	var req struct {
		ID      string `json:"id"`
		Action  string `json:"action"`
		Enabled bool   `json:"enabled"`
		Editor  string `json:"editor"`
	}
	if err = panelBody(w, r, &req); err != nil {
		return nil, err
	}
	history, _ := loadHistory(c)
	idx, _ := loadIndex(c)
	found := false
	for _, project := range merge(history, idx.Projects) {
		if project.ID == req.ID {
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("记录已不在当前来源中，请刷新列表")
	}
	err = changePreferences(c, func(prefs *Preferences) error {
		switch req.Action {
		case "hidden":
			if req.Enabled {
				prefs.Hidden[req.ID] = true
			} else {
				delete(prefs.Hidden, req.ID)
			}
		case "pinned":
			if req.Enabled {
				prefs.Pinned[req.ID] = true
			} else {
				delete(prefs.Pinned, req.ID)
			}
		case "editor":
			if req.Editor == "" {
				delete(prefs.Editors, req.ID)
			} else if validEditor(req.Editor) {
				prefs.Editors[req.ID] = req.Editor
			} else {
				return fmt.Errorf("无效 IDE")
			}
		default:
			return fmt.Errorf("无效操作")
		}
		return nil
	})
	return map[string]bool{"ok": err == nil}, err
}
