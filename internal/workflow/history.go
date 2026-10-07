package workflow

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

func historyPaths(c Config) []string {
	if c.Database != "" {
		return []string{c.Database}
	}
	paths := []string{}
	for _, app := range []string{filepath.Join(c.Home, "Applications/Visual Studio Code.app"), "/Applications/Visual Studio Code.app"} {
		var product struct {
			Shared string `json:"sharedDataFolderName"`
		}
		if readJSON(filepath.Join(app, "Contents/Resources/app/product.json"), &product) == nil && product.Shared != "" && filepath.Base(product.Shared) == product.Shared {
			paths = append(paths, filepath.Join(c.Home, product.Shared, "sharedStorage/state.vscdb"))
		}
	}
	// VS Code may keep application-shared history separately from profile storage.
	paths = append(paths, filepath.Join(c.Home, ".vscode-shared/sharedStorage/state.vscdb"), filepath.Join(c.VSCodeDir, "User/globalStorage/state.vscdb"))
	return paths
}

func parseHistory(b []byte) ([]Project, error) {
	var data struct {
		Entries []json.RawMessage `json:"entries"`
	}
	if e := json.Unmarshal(b, &data); e != nil {
		return nil, e
	}
	out := []Project{}
	for _, raw := range data.Entries {
		var entry struct {
			Folder string `json:"folderUri"`
			Label  string `json:"label"`
		}
		if json.Unmarshal(raw, &entry) != nil || entry.Folder == "" {
			continue
		}
		p, e := project(entry.Folder, entry.Label)
		if e == nil {
			out = append(out, p)
		}
	}
	return merge(out), nil
}

func readHistoryDB(name string) ([]byte, error) {
	abs, e := filepath.Abs(name)
	if e != nil {
		return nil, e
	}
	u := url.URL{Scheme: "file", Path: abs}
	values := url.Values{"mode": {"ro"}, "_pragma": {"busy_timeout(30)", "query_only(1)"}}
	u.RawQuery = values.Encode()
	db, e := sql.Open("sqlite", u.String())
	if e != nil {
		return nil, e
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	var value string
	e = db.QueryRowContext(ctx, `SELECT value FROM ItemTable WHERE key IN ('recently.opened','history.recentlyOpenedPathsList') ORDER BY CASE key WHEN 'recently.opened' THEN 0 ELSE 1 END LIMIT 1`).Scan(&value)
	if e == sql.ErrNoRows {
		return nil, nil
	}
	if e != nil {
		return nil, e
	}
	return []byte(value), nil
}

type historySnapshot struct {
	Source   string    `json:"source"`
	Digest   string    `json:"digest"`
	Projects []Project `json:"projects"`
}

func loadHistory(c Config) ([]Project, []string) {
	warnings := []string{}
	cache := filepath.Join(c.CacheDir, "history.cache")
	var old historySnapshot
	oldErr := readCache(cache, &old)
	paths := historyPaths(c)
	for _, name := range paths {
		if _, e := os.Stat(name); e != nil {
			if !os.IsNotExist(e) {
				warnings = append(warnings, e.Error())
			}
			continue
		}
		payload, e := readHistoryDB(name)
		if e == nil && payload == nil {
			continue
		} // An absent key permits the legacy storage fallback.
		if e == nil {
			digest := fmt.Sprintf("%x", sha256.Sum256(payload))
			if oldErr == nil && old.Source == name && old.Digest == digest {
				return old.Projects, warnings
			}
			var projects []Project
			projects, e = parseHistory(payload)
			if e == nil {
				snapshot := historySnapshot{Source: name, Digest: digest, Projects: projects}
				if err := atomicCache(cache, snapshot); err != nil {
					warnings = append(warnings, "History snapshot: "+err.Error())
				}
				return projects, warnings
			}
		}
		warnings = append(warnings, fmt.Sprintf("VS Code history unavailable: %s", e))
	}
	if oldErr == nil {
		for _, source := range paths {
			if source == old.Source {
				return old.Projects, append(warnings, "Using last readable VS Code history")
			}
		}
	}
	return []Project{}, append(warnings, "VS Code history not found; directory discovery remains available")
}
