package workflow

import (
	"errors"
	"os"
	"syscall"
	"time"
)

func queryPath(c Config, input string, prefs Preferences) Feedback {
	directory := expand(input, c.Home)
	feedback := Feedback{SkipKnowledge: true}
	invalid := func(title string) Feedback {
		feedback.Items = []Item{{Title: title, Subtitle: directory, Valid: false}}
		return feedback
	}
	type result struct {
		project Project
		err     error
		notDir  bool
	}
	// Path input never resolves relative to a configured root. Bound even symlink
	// resolution so an offline mount cannot indefinitely stall a typing query.
	ready := make(chan result, 1)
	go func() {
		info, err := os.Stat(directory)
		if err != nil {
			ready <- result{err: err}
			return
		}
		if !info.IsDir() {
			ready <- result{notDir: true}
			return
		}
		project, err := localProject(directory)
		ready <- result{project: project, err: err}
	}()
	timer := time.NewTimer(30 * time.Millisecond)
	defer timer.Stop()
	var target Project
	select {
	case found := <-ready:
		switch {
		case found.notDir || errors.Is(found.err, syscall.ENOTDIR):
			return invalid("路径不是目录")
		case os.IsNotExist(found.err):
			return invalid("目录不存在")
		case os.IsPermission(found.err):
			return invalid("无权访问目录")
		case found.err != nil:
			return invalid("无法读取输入的目录")
		}
		target = found.project
	case <-timer.C:
		return invalid("目录检查超时，请重试")
	}

	history, warnings := loadHistory(c)
	index, _ := loadIndex(c)
	feedback.Warnings = append(warnings, index.Warnings...)
	for _, project := range merge(history, index.Projects) {
		if project.ID == target.ID && !prefs.Hidden[project.ID] {
			feedback.Items = render(c, []Project{project}, prefs)
			return feedback
		}
	}

	// An explicit path can be opened without importing it or changing hidden
	// records. Unlisted targets use the configured default, not stale overrides.
	feedback.Items = render(c, []Project{target}, emptyPreferences())
	item := &feedback.Items[0]
	if item.Valid {
		item.Title = "用 " + appNames[c.Editor] + " 打开输入的目录？"
		item.Subtitle = "按 Enter 打开 · " + item.Subtitle
		item.Mods["alt"] = Modifier{Subtitle: "临时打开的目录无需固定", Valid: false}
		item.Mods["ctrl"] = Modifier{Subtitle: "临时打开的目录无需隐藏", Valid: false}
	}
	return feedback
}
