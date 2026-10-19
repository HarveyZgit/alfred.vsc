package workflow

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestQueryExactNameAndAlfredOrdering(t *testing.T) {
	c := fixture(t)
	db := database(t, c)
	coding := repo(t, filepath.Join(c.Home, "coding"), "main")
	backend := repo(t, filepath.Join(c.Home, "backend"), "fresh-branch")
	other := repo(t, filepath.Join(c.Home, "proxy-backend"), "main")
	setHistory(t, db, "recently.opened", []map[string]string{
		{"folderUri": coding.URI}, {"folderUri": other.URI}, {"folderUri": backend.URI},
	})
	for _, query := range []string{"backend", "", "not-a-project"} {
		feedback, err := Query(c, query)
		if err != nil {
			t.Fatal(err)
		}
		data, err := json.Marshal(feedback)
		if err != nil {
			t.Fatal(err)
		}
		var decoded map[string]any
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatal(err)
		}
		if decoded["skipknowledge"] != true {
			t.Fatalf("query %q allows Alfred to reorder results: %s", query, data)
		}
		if query == "backend" {
			got := targets(feedback)
			if len(got) != 1 || got[0] != backend.URI {
				t.Fatalf("exact search returned %v", got)
			}
			if !strings.Contains(feedback.Items[0].Subtitle, "fresh-branch") {
				t.Fatalf("missing current branch: %s", data)
			}
		}
	}
}

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
