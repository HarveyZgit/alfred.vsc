package workflow

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
)

func panelMigrations(c Config) (any, error) {
	files, err := filepath.Glob(filepath.Join(c.DataDir, "migrations", "*", "report.json"))
	if err != nil {
		return nil, err
	}
	reports := []any{}
	for _, file := range files {
		var report MigrationReport
		latest := filepath.Join(filepath.Dir(file), "latest-report.json")
		historical := false
		if err := readJSON(latest, &report); os.IsNotExist(err) {
			historical = true
			if err = readJSON(file, &report); err != nil {
				return nil, err
			}
		} else if err != nil {
			return nil, err
		}
		reports = append(reports, map[string]any{"report": report, "historical": historical})
	}
	return reports, nil
}

func (p *panel) panelMigrate(c Config, w http.ResponseWriter, r *http.Request) (any, error) {
	var err error
	var req struct {
		Source string `json:"source"`
		Apply  bool   `json:"apply"`
	}
	if err = panelBody(w, r, &req); err != nil {
		return nil, err
	}
	if req.Source == "" {
		return nil, fmt.Errorf("请填写旧版数据目录或缓存文件路径")
	}
	if !p.busy.TryLock() {
		return nil, fmt.Errorf("正在扫描或迁移，请稍后重试")
	}
	defer p.busy.Unlock()
	return ImportLegacy(c, expand(req.Source, c.Home), req.Apply)
}
