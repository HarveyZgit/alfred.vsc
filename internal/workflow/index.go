package workflow

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var skipDirs = map[string]bool{"node_modules": true, "vendor": true, "build": true, "dist": true, "target": true, "venv": true, "__pycache__": true, "Pods": true, "miniprogram_npm": true}

type Index struct {
	Version  int       `json:"version"`
	Key      string    `json:"key"`
	Updated  int64     `json:"updated"`
	Projects []Project `json:"projects"`
	Warnings []string  `json:"warnings,omitempty"`
}

func indexKey(c Config) string {
	b, _ := json.Marshal(struct {
		Roots []string
		Depth int
	}{c.Roots, c.Depth})
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func scan(c Config) ([]Project, []string) {
	projects := []Project{}
	warnings := []string{}
	seen := map[string]bool{}
	deadline := time.Now().Add(20 * time.Second)
	visited := 0
	for _, root := range c.Roots {
		real, e := filepath.EvalSymlinks(root)
		if e != nil {
			warnings = append(warnings, "Cannot scan "+root+": "+e.Error())
			continue
		}
		info, e := os.Stat(real)
		if e != nil || !info.IsDir() {
			warnings = append(warnings, "Not a directory: "+root)
			continue
		}
		e = filepath.WalkDir(real, func(name string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				warnings = append(warnings, "Cannot read "+name)
				if d != nil && d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if !d.IsDir() {
				return nil
			}
			visited++
			if visited > 100000 || time.Now().After(deadline) {
				return fmt.Errorf("scan budget exceeded; reduce scan roots/depth")
			}
			rel, _ := filepath.Rel(real, name)
			depth := 0
			if rel != "." {
				depth = len(strings.Split(rel, string(filepath.Separator)))
				if strings.HasPrefix(d.Name(), ".") || skipDirs[d.Name()] || depth > c.Depth {
					return filepath.SkipDir
				}
			}
			if _, err := os.Stat(filepath.Join(name, ".git")); err == nil {
				p, err := localProject(name)
				if err == nil && !seen[p.ID] {
					seen[p.ID] = true
					projects = append(projects, p)
				}
			}
			return nil
		})
		if e != nil {
			warnings = append(warnings, e.Error())
			break
		}
	}
	return projects, warnings
}

func BuildIndex(c Config, nonblocking bool) (Index, error) {
	lock, e := lockFile(filepath.Join(c.CacheDir, "index.lock"), nonblocking)
	if e != nil {
		return Index{}, e
	}
	defer unlock(lock)
	projects, warnings := scan(c)
	if len(warnings) > 0 {
		// A transient inaccessible root must not erase a previously discovered project.
		previous, _ := loadIndex(c)
		projects = merge(projects, previous.Projects)
	}
	index := Index{Version: 1, Key: indexKey(c), Updated: time.Now().Unix(), Projects: projects, Warnings: warnings}
	e = atomicCache(filepath.Join(c.CacheDir, "index.cache"), index)
	return index, e
}

func loadIndex(c Config) (Index, bool) {
	var index Index
	e := readCache(filepath.Join(c.CacheDir, "index.cache"), &index)
	valid := e == nil && index.Version == 1 && index.Key == indexKey(c)
	if !valid {
		index = Index{Projects: []Project{}}
	}
	return index, !valid || time.Since(time.Unix(index.Updated, 0)) > c.refresh()
}

func refreshIndex(c Config) bool {
	if !c.Background || len(c.Roots) == 0 {
		return false
	}
	// A separate launch lock prevents a keypress burst from spawning many scanners.
	f, e := lockFile(filepath.Join(c.CacheDir, "launch.lock"), true)
	if e != nil {
		return false
	}
	defer unlock(f)
	marker := filepath.Join(c.CacheDir, "index-launch")
	if info, e := os.Stat(marker); e == nil && time.Since(info.ModTime()) < 5*time.Second {
		return true
	}
	exe, e := os.Executable()
	if e != nil {
		return false
	}
	log, e := os.OpenFile(filepath.Join(c.CacheDir, "index.log"), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if e != nil {
		return false
	}
	defer log.Close()
	cmd := exec.Command(exe, "index", "--background")
	cmd.Stderr = log
	cmd.Stdout = log
	if e = cmd.Start(); e != nil {
		return false
	}
	_ = cmd.Process.Release()
	_ = os.WriteFile(marker, []byte("started"), 0600)
	return true
}
