package workflow

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type legacyRecord struct {
	Path string `json:"path"`
	Type string `json:"type"`
	Icon struct {
		Path string `json:"path"`
	} `json:"icon"`
	Extra struct {
		From string `json:"from"`
	} `json:"extra"`
}

func migrationRead(name string) ([]byte, error) {
	f, e := os.Open(name)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	b, e := io.ReadAll(io.LimitReader(f, (32<<20)+1))
	if e == nil && len(b) > 32<<20 {
		e = fmt.Errorf("migration input exceeds 32 MiB: %s", name)
	}
	return b, e
}

func legacySource(source string) (string, error) {
	source, e := filepath.Abs(source)
	if e != nil {
		return "", e
	}
	info, e := os.Stat(source)
	if e != nil {
		return "", e
	}
	if !info.IsDir() {
		return source, nil
	}
	for _, name := range []string{filepath.Join(source, ".records.cache.json"), filepath.Join(source, "src/.records.cache.json")} {
		if _, e := os.Stat(name); e == nil {
			return name, nil
		} else if !os.IsNotExist(e) {
			return "", e
		}
	}
	return "", fmt.Errorf("旧目录中未找到 .records.cache.json: %s", source)
}

// V1/V2 wrote decoded local paths, so '%' and '#' must be treated literally
// before any URL parser gets a chance to interpret them as escapes or fragments.
func legacyProject(r legacyRecord) (Project, error) {
	raw := r.Path
	if strings.HasPrefix(raw, "file:///") && r.Extra.From != "getRecordsFromVscodeMenu" {
		return localProject(strings.TrimPrefix(raw, "file://"))
	}
	if filepath.IsAbs(raw) {
		return localProject(raw)
	}
	if strings.HasPrefix(raw, "vscode-remote://") && r.Extra.From == "getRecordsFromVscodeDB" {
		rest := strings.TrimPrefix(raw, "vscode-remote://")
		at := strings.Index(rest, "/")
		if at < 1 {
			return Project{}, fmt.Errorf("invalid legacy remote URI")
		}
		raw = (&url.URL{Scheme: "vscode-remote", Host: rest[:at], Path: rest[at:]}).String()
	}
	return project(raw, "")
}

func historyOrderKey(projects []Project) string {
	var ordered strings.Builder
	ordered.Grow(len(projects) * 33)
	for _, p := range projects {
		ordered.WriteString(p.ID)
		ordered.WriteByte('\n')
	}
	return fmt.Sprintf("%x", sha256.Sum256([]byte(ordered.String())))
}

func migrationHistory(c Config) ([]Project, []string) {
	warnings := []string{}
	// Preview is genuinely read-only: do not initialize history caches or prefs.
	for _, name := range historyPaths(c) {
		b, e := readHistoryDB(name)
		if e == nil && b != nil {
			if p, e := parseHistory(b); e == nil {
				return p, warnings
			}
		}
	}
	var old historySnapshot
	if readCache(filepath.Join(c.CacheDir, "history.cache"), &old) == nil {
		for _, name := range historyPaths(c) {
			if name == old.Source {
				return old.Projects, append(warnings, "VS Code 当前历史不可读：以同源最后快照判断迁移范围；请确认后再执行。")
			}
		}
	}
	return nil, append(warnings, "VS Code 历史不可读或未找到：目前只能确认配置根目录中的 Git 仓库；请检查来源后重跑迁移。")
}

func legacyDirectory(r legacyRecord, p Project, known map[string]bool) bool {
	if r.Type == "file" || strings.HasSuffix(r.Icon.Path, "file.png") || strings.HasSuffix(strings.ToLower(p.Path), ".code-workspace") {
		return false
	}
	if p.Kind == "remote" {
		return known[p.ID] || r.Type == "folder"
	} // "remote" alone cannot distinguish a remote file.
	if info, e := os.Stat(p.Path); e == nil {
		return info.IsDir()
	}
	return known[p.ID] || r.Type == "folder" || strings.HasSuffix(r.Icon.Path, "folder.png")
}
