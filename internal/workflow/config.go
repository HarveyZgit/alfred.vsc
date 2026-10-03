package workflow

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const Version = "3.0.0"

type Config struct {
	Roots          []string          `json:"roots"`
	VSCodeDir      string            `json:"vscode_dir"`
	Database       string            `json:"database,omitempty"`
	Editor         string            `json:"editor"`
	Editors        map[string]string `json:"editors"`
	Limit          int               `json:"limit"`
	Depth          int               `json:"scan_depth"`
	RefreshSeconds int               `json:"refresh_seconds"`
	DataDir        string            `json:"-"`
	CacheDir       string            `json:"-"`
	Home           string            `json:"-"`
	OS             string            `json:"-"`
	Background     bool              `json:"-"`
}

func LoadConfig() (Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Config{}, err
	}
	c := Config{Home: home, OS: runtime.GOOS, Editor: "vscode", Editors: map[string]string{}, Limit: 50, Depth: 8, RefreshSeconds: 300, Background: os.Getenv("VSC_NO_BACKGROUND") != "1"}
	if runtime.GOOS == "darwin" {
		c.VSCodeDir = filepath.Join(home, "Library/Application Support/Code")
	} else {
		c.VSCodeDir = filepath.Join(home, ".config/Code")
	}
	c.DataDir = os.Getenv("VSC_DATA_DIR")
	if c.DataDir == "" {
		c.DataDir = os.Getenv("alfred_workflow_data")
	}
	if c.DataDir == "" {
		base, e := os.UserConfigDir()
		if e != nil {
			return c, e
		}
		c.DataDir = filepath.Join(base, "alfred-vsc")
	}
	if os.Getenv("VSC_DEV_MODE") == "1" {
		c.DataDir = os.Getenv("VSC_DEV_CACHE_DIR")
		if c.DataDir == "" {
			c.DataDir = filepath.Join(os.TempDir(), "vsc-dev")
		}
	}
	c.DataDir = expand(c.DataDir, home)
	if b, e := os.ReadFile(filepath.Join(c.DataDir, "config.json")); e == nil {
		if e = json.Unmarshal(b, &c); e != nil {
			return c, fmt.Errorf("config.json: %w", e)
		}
	} else if !os.IsNotExist(e) {
		return c, e
	}
	if v, ok := os.LookupEnv("VSC_DIRECTORIES"); ok {
		c.Roots = nil
		if strings.HasPrefix(strings.TrimSpace(v), "[") {
			if err = json.Unmarshal([]byte(v), &c.Roots); err != nil {
				return c, fmt.Errorf("VSC_DIRECTORIES must be a JSON string array: %w", err)
			}
		} else {
			c.Roots = strings.FieldsFunc(v, func(r rune) bool { return r == ',' || r == '\n' })
		}
	}
	for name, dst := range map[string]*string{"VSC_IDE_PATH": &c.VSCodeDir, "VSC_DB_PATH": &c.Database, "VSC_DEFAULT_EDITOR": &c.Editor} {
		if v := os.Getenv(name); v != "" {
			*dst = v
		}
	}
	if c.Editors == nil {
		c.Editors = map[string]string{}
	}
	for _, editor := range []string{"vscode", "zed", "trae", "cursor"} {
		if v := os.Getenv("VSC_EDITOR_" + strings.ToUpper(editor)); v != "" {
			c.Editors[editor] = expand(v, home)
		}
	}
	for editor, path := range c.Editors {
		c.Editors[editor] = expand(path, home)
	}
	// Treat legacy overrides as an executable path, never as a shell command.
	if v := os.Getenv("VSC_OPEN_DEFAULT"); v != "" {
		c.Editors[c.Editor] = expand(v, home)
	}
	for name, dst := range map[string]*int{"VSC_LIMIT": &c.Limit, "VSC_SCAN_DEPTH": &c.Depth, "VSC_REFRESH_SECONDS": &c.RefreshSeconds} {
		if v := os.Getenv(name); v != "" {
			n, e := strconv.Atoi(v)
			if e != nil {
				return c, fmt.Errorf("%s: %w", name, e)
			}
			*dst = n
		}
	}
	if c.Limit < 1 || c.Limit > 200 {
		return c, fmt.Errorf("limit must be between 1 and 200")
	}
	if c.Depth < 0 || c.Depth > 32 {
		return c, fmt.Errorf("scan_depth must be between 0 and 32")
	}
	if c.RefreshSeconds < 1 {
		return c, fmt.Errorf("refresh_seconds must be positive")
	}
	if !validEditor(c.Editor) {
		return c, fmt.Errorf("unknown editor %q (use vscode, zed, trae or cursor)", c.Editor)
	}
	roots := []string{}
	seen := map[string]bool{}
	for _, root := range c.Roots {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		root = expand(root, home)
		abs, e := filepath.Abs(root)
		if e != nil {
			return c, e
		}
		if !seen[abs] {
			roots = append(roots, abs)
			seen[abs] = true
		}
	}
	c.Roots = roots
	c.VSCodeDir = expand(c.VSCodeDir, home)
	c.Database = expand(c.Database, home)
	c.CacheDir = os.Getenv("VSC_CACHE_DIR")
	if c.CacheDir == "" {
		c.CacheDir = filepath.Join(c.DataDir, "cache")
	}
	c.CacheDir = expand(c.CacheDir, home)
	return c, nil
}
func expand(s, home string) string {
	if s == "~" {
		return home
	}
	if strings.HasPrefix(s, "~/") {
		return filepath.Join(home, s[2:])
	}
	return s
}
func validEditor(s string) bool         { return s == "vscode" || s == "zed" || s == "trae" || s == "cursor" }
func (c Config) refresh() time.Duration { return time.Duration(c.RefreshSeconds) * time.Second }
