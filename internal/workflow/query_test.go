package workflow

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func BenchmarkQueryTenThousand(b *testing.B) {
	base := b.TempDir()
	c := Config{Home: base, DataDir: filepath.Join(base, "data"), CacheDir: filepath.Join(base, "cache"), VSCodeDir: filepath.Join(base, "Code"), Editor: "vscode", Limit: 50, Depth: 8, RefreshSeconds: 300}
	c.Database = filepath.Join(base, "state.vscdb")
	db, err := sql.Open("sqlite", c.Database)
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()
	db.Exec("CREATE TABLE ItemTable (key TEXT PRIMARY KEY,value TEXT)")
	entries := []map[string]string{}
	for i := 0; i < 10000; i++ {
		name := filepath.Join(base, fmt.Sprintf("project-%05d", i))
		if i < 50 {
			os.MkdirAll(filepath.Join(name, ".git"), 0700)
			os.WriteFile(filepath.Join(name, ".git/HEAD"), []byte("ref: refs/heads/main\n"), 0600)
		}
		p, _ := localProject(name)
		entries = append(entries, map[string]string{"folderUri": p.URI})
	}
	payload, _ := json.Marshal(map[string]any{"entries": entries})
	db.Exec("INSERT INTO ItemTable VALUES ('recently.opened',?)", string(payload))
	if _, err = Query(c, "project"); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err = Query(c, "project"); err != nil {
			b.Fatal(err)
		}
	}
}
