package workflow

import (
	"database/sql"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestHistoryUnionTypesAndFreshness(t *testing.T) {
	c := fixture(t)
	root := filepath.Join(c.Home, "projects")
	c.Roots = []string{root}
	a := repo(t, filepath.Join(root, "alpha"), "main")
	plain := filepath.Join(c.Home, "plain")
	mkdir(t, plain)
	p, _ := localProject(plain)
	_ = repo(t, filepath.Join(root, "node_modules/ignored"), "main")
	mkdir(t, filepath.Join(root, "not-a-repo"))
	db := database(t, c)
	remote := "vscode-remote://ssh-remote+devbox/srv/project"
	setHistory(t, db, "history.recentlyOpenedPathsList", []any{map[string]string{"folderUri": a.URI}, map[string]string{"folderUri": p.URI}, map[string]string{"folderUri": remote, "label": "Server"}, map[string]string{"fileUri": a.URI + "/file.txt"}, map[string]string{"fileUri": remote + "/file.rs"}, map[string]any{"workspace": map[string]string{"configPath": a.URI + "/a.code-workspace"}}})
	if _, e := BuildIndex(c, false); e != nil {
		t.Fatal(e)
	}
	f, e := Query(c, "")
	if e != nil {
		t.Fatal(e)
	}
	if got := targets(f); !reflect.DeepEqual(got, []string{a.URI, p.URI, remote}) {
		t.Fatalf("union: %v", got)
	}
	setHistory(t, db, "history.recentlyOpenedPathsList", []any{map[string]string{"folderUri": remote}})
	f, e = Query(c, "")
	if e != nil {
		t.Fatal(e)
	}
	if got := targets(f); !reflect.DeepEqual(got, []string{remote, a.URI}) {
		t.Fatalf("history did not refresh: %v", got)
	}
}

func TestNewHistoryKeyAndEmptyHistory(t *testing.T) {
	c := fixture(t)
	db := database(t, c)
	p := repo(t, filepath.Join(c.Home, "p"), "main")
	setHistory(t, db, "history.recentlyOpenedPathsList", []any{map[string]string{"folderUri": p.URI}})
	setHistory(t, db, "recently.opened", []any{})
	projects, w := loadHistory(c)
	if len(projects) != 0 || len(w) != 0 {
		t.Fatalf("empty current history must win: %v %v", projects, w)
	}
}

func TestHistoryWALReadOnly(t *testing.T) {
	c := fixture(t)
	db := database(t, c)
	if _, e := db.Exec("PRAGMA journal_mode=WAL"); e != nil {
		t.Fatal(e)
	}
	p := repo(t, filepath.Join(c.Home, "p"), "main")
	setHistory(t, db, "recently.opened", []any{map[string]string{"folderUri": p.URI}})
	tx, e := db.Begin()
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback()
	_, e = tx.Exec("UPDATE ItemTable SET value='{}'")
	if e != nil {
		t.Fatal(e)
	}
	projects, w := loadHistory(c)
	if len(projects) != 1 || len(w) != 0 {
		t.Fatalf("WAL committed view: %v %v", projects, w)
	}
	tx.Rollback()
	var value string
	db.QueryRow("SELECT value FROM ItemTable").Scan(&value)
	if !strings.Contains(value, p.URI) {
		t.Fatal("source altered")
	}
}

func TestBrokenHistoryKeepsLastSnapshot(t *testing.T) {
	c := fixture(t)
	db := database(t, c)
	p := repo(t, filepath.Join(c.Home, "p"), "main")
	setHistory(t, db, "recently.opened", []any{map[string]string{"folderUri": p.URI}})
	loadHistory(c)
	db.Exec("UPDATE ItemTable SET value='broken-json'")
	projects, w := loadHistory(c)
	if len(projects) != 1 || len(w) == 0 {
		t.Fatalf("fallback: %v %v", projects, w)
	}
}

func TestHiddenHistoryNeverResurrects(t *testing.T) {
	c := fixture(t)
	db := database(t, c)
	p := repo(t, filepath.Join(c.Home, "p"), "main")
	setHistory(t, db, "recently.opened", []any{map[string]string{"folderUri": p.URI}})
	if e := changePreferences(c, func(prefs *Preferences) error { prefs.Hidden[p.ID] = true; return nil }); e != nil {
		t.Fatal(e)
	}
	for i := 0; i < 2; i++ {
		f, e := Query(c, "")
		if e != nil || len(targets(f)) != 0 {
			t.Fatalf("hidden item reappeared: %+v %v", f, e)
		}
	}
	changePreferences(c, func(prefs *Preferences) error { delete(prefs.Hidden, p.ID); return nil })
	f, _ := Query(c, "")
	if len(targets(f)) != 1 {
		t.Fatal("restore failed")
	}
}

func TestHistoryLockedIsBounded(t *testing.T) {
	c := fixture(t)
	db := database(t, c)
	p := repo(t, filepath.Join(c.Home, "p"), "main")
	setHistory(t, db, "recently.opened", []any{map[string]string{"folderUri": p.URI}})
	loadHistory(c)
	if _, e := db.Exec("BEGIN EXCLUSIVE"); e != nil {
		t.Fatal(e)
	}
	defer db.Exec("ROLLBACK")
	start := time.Now()
	projects, w := loadHistory(c)
	if time.Since(start) > time.Second || len(projects) != 1 || len(w) == 0 {
		t.Fatalf("lock fallback: %d %v", len(projects), w)
	}
}

func TestHistoryMissingDatabaseFallbackAndSourceChange(t *testing.T) {
	c := fixture(t)
	db := database(t, c)
	p := repo(t, filepath.Join(c.Home, "p"), "main")
	setHistory(t, db, "recently.opened", []any{map[string]string{"folderUri": p.URI}})
	loadHistory(c)
	db.Close()
	name := filepath.Join(c.VSCodeDir, "User/globalStorage/state.vscdb")
	if e := os.Rename(name, name+".away"); e != nil {
		t.Fatal(e)
	}
	projects, w := loadHistory(c)
	if len(projects) != 1 || len(w) == 0 {
		t.Fatal("missing DB should retain recoverable snapshot")
	}
	c.Database = filepath.Join(c.Home, "another-db")
	projects, _ = loadHistory(c)
	if len(projects) != 0 {
		t.Fatal("snapshot must not leak across configured sources")
	}
}

func TestSharedStorageWithoutHistoryFallsBack(t *testing.T) {
	c := fixture(t)
	db := database(t, c)
	p := repo(t, filepath.Join(c.Home, "p"), "main")
	setHistory(t, db, "recently.opened", []any{map[string]string{"folderUri": p.URI}})
	shared := filepath.Join(c.Home, ".vscode-shared/sharedStorage/state.vscdb")
	mkdir(t, filepath.Dir(shared))
	empty, e := sql.Open("sqlite", shared)
	if e != nil {
		t.Fatal(e)
	}
	defer empty.Close()
	if _, e = empty.Exec("CREATE TABLE ItemTable (key TEXT PRIMARY KEY,value TEXT)"); e != nil {
		t.Fatal(e)
	}
	projects, w := loadHistory(c)
	if len(projects) != 1 || len(w) != 0 {
		t.Fatalf("fallback: %v %v", projects, w)
	}
}
