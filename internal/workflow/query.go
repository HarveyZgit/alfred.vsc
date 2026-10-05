package workflow

import (
	"strings"
)

func Query(c Config, query string) (Feedback, error) {
	prefs, e := loadPreferences(c)
	if e != nil {
		return Feedback{}, e
	}
	if strings.HasPrefix(strings.TrimSpace(query), "/") {
		return browse(c, strings.TrimSpace(query), prefs)
	}
	history, warnings := loadHistory(c)
	if len(prefs.LegacyOrder) > 0 && historyOrderKey(history) != prefs.LegacyHistory {
		prefs.LegacyOrder = nil
	}
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
