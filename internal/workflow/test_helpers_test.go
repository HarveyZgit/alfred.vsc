package workflow

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func fixture(t *testing.T) Config {
	t.Helper()
	base := t.TempDir()
	c := Config{Home: base, OS: "linux", DataDir: filepath.Join(base, "data"), CacheDir: filepath.Join(base, "cache"), VSCodeDir: filepath.Join(base, "Code"), Editor: "vscode", Editors: map[string]string{}, Limit: 50, Depth: 8, RefreshSeconds: 300}
	return c
}

func write(t *testing.T, name, content string) {
	t.Helper()
	if e := os.MkdirAll(filepath.Dir(name), 0700); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(name, []byte(content), 0600); e != nil {
		t.Fatal(e)
	}
}

func mkdir(t *testing.T, name string) {
	t.Helper()
	if e := os.MkdirAll(name, 0700); e != nil {
		t.Fatal(e)
	}
}

func repo(t *testing.T, name, branch string) Project {
	t.Helper()
	write(t, filepath.Join(name, ".git/HEAD"), "ref: refs/heads/"+branch+"\n")
	p, e := localProject(name)
	if e != nil {
		t.Fatal(e)
	}
	return p
}

func database(t *testing.T, c Config) *sql.DB {
	t.Helper()
	name := filepath.Join(c.VSCodeDir, "User/globalStorage/state.vscdb")
	mkdir(t, filepath.Dir(name))
	db, e := sql.Open("sqlite", name)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = db.Exec("CREATE TABLE ItemTable (key TEXT PRIMARY KEY,value TEXT)"); e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func setHistory(t *testing.T, db *sql.DB, key string, entries any) {
	t.Helper()
	b, e := json.Marshal(map[string]any{"entries": entries})
	if e != nil {
		t.Fatal(e)
	}
	_, e = db.Exec("INSERT OR REPLACE INTO ItemTable VALUES (?,?)", key, string(b))
	if e != nil {
		t.Fatal(e)
	}
}

func targets(f Feedback) []string {
	out := []string{}
	for _, i := range f.Items {
		if i.UID != "" {
			var r OpenRequest
			_ = json.Unmarshal([]byte(i.Arg), &r)
			out = append(out, r.Target)
		}
	}
	return out
}

func fakeEditor(t *testing.T, c *Config, name, body string) string {
	t.Helper()
	p := filepath.Join(c.Home, "Editor With Spaces", name)
	write(t, p, "#!/bin/sh\n"+body+"\n")
	os.Chmod(p, 0700)
	c.Editors[name] = p
	return p
}
