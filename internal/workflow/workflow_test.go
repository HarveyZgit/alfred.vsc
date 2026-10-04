package workflow

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
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
func TestPreferenceConcurrentWriters(t *testing.T) {
	c := fixture(t)
	var wg sync.WaitGroup
	errs := make(chan error, 30)
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errs <- changePreferences(c, func(p *Preferences) error { p.Hidden[fmt.Sprint(i)] = true; return nil })
		}(i)
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		if e != nil {
			t.Fatal(e)
		}
	}
	p, e := loadPreferences(c)
	if e != nil || len(p.Hidden) != 30 {
		t.Fatalf("lost updates: %d %v", len(p.Hidden), e)
	}
}
func TestCorruptPreferencesAreNotOverwritten(t *testing.T) {
	c := fixture(t)
	name := filepath.Join(c.DataDir, "preferences.json")
	write(t, name, "{broken")
	if e := changePreferences(c, func(p *Preferences) error { return nil }); e == nil {
		t.Fatal("expected error")
	}
	b, _ := os.ReadFile(name)
	if string(b) != "{broken" {
		t.Fatal("corrupt preferences overwritten")
	}
}
func TestLegacyTrashMigration(t *testing.T) {
	c := fixture(t)
	p := repo(t, filepath.Join(c.Home, "p"), "main")
	b, _ := json.Marshal(map[string]any{"trash": map[string]any{"old-md5": map[string]string{"path": p.URI}}})
	write(t, filepath.Join(c.DataDir, ".records.cache.json"), string(b))
	c.Roots = []string{c.Home}
	if _, e := ImportLegacy(c, filepath.Join(c.DataDir, ".records.cache.json"), true); e != nil {
		t.Fatal(e)
	}
	prefs, _ := loadPreferences(c)
	if !prefs.Hidden[p.ID] || !prefs.Migrated {
		t.Fatal("legacy hidden record lost")
	}
}
func TestRealTimeGitVariants(t *testing.T) {
	c := fixture(t)
	normal := repo(t, filepath.Join(c.Home, "normal"), "main")
	worktree := filepath.Join(c.Home, "worktree")
	write(t, filepath.Join(worktree, ".git"), "gitdir: ../meta/tree\n")
	write(t, filepath.Join(c.Home, "meta/tree/HEAD"), "ref: refs/heads/work\n")
	mkdir(t, filepath.Join(normal.Path, "subdir"))
	for _, tt := range []struct{ path, want string }{{normal.Path, "main"}, {worktree, "work"}, {filepath.Join(normal.Path, "subdir"), "main"}} {
		if got := gitBranch(tt.path); got != tt.want {
			t.Fatalf("%s: %s", tt.path, got)
		}
	}
	write(t, filepath.Join(normal.Path, ".git/HEAD"), "ref: refs/heads/feature/new\n")
	items := render(c, []Project{normal}, emptyPreferences())
	if !strings.Contains(items[0].Subtitle, "feature/new") {
		t.Fatal("stale branch")
	}
	write(t, filepath.Join(normal.Path, ".git/HEAD"), strings.Repeat("a", 40))
	if got := gitBranch(normal.Path); got != "aaaaaaa" {
		t.Fatal(got)
	}
}
func TestRemoteClassificationAndEncoding(t *testing.T) {
	c := fixture(t)
	local := repo(t, filepath.Join(c.Home, "remote 项目 #100%"), "main")
	if local.Kind != "folder" {
		t.Fatal("local classified remote")
	}
	round, e := project(local.URI, "")
	if e != nil || round.Path != local.Path {
		t.Fatalf("URI roundtrip %v %v", round, e)
	}
	r, e := project("vscode-remote://ssh-remote+server/path/%E4%B8%AD%E6%96%87%20dir", "")
	if e != nil || r.Kind != "remote" {
		t.Fatal(e)
	}
	items := render(c, []Project{r}, emptyPreferences())
	if strings.Contains(items[0].Subtitle, "⎇") || !strings.Contains(items[0].Subtitle, "server") {
		t.Fatal(items)
	}
	if _, e = project("https://example.com/path", ""); e == nil {
		t.Fatal("unsupported URI")
	}
}
func TestScanWorktreeDepthAndExclusions(t *testing.T) {
	c := fixture(t)
	root := filepath.Join(c.Home, "root")
	c.Roots = []string{root}
	normal := repo(t, filepath.Join(root, "normal"), "main")
	repo(t, filepath.Join(root, ".hidden"), "main")
	repo(t, filepath.Join(root, "build/ignored"), "main")
	repo(t, filepath.Join(root, "normal/nested"), "main")
	write(t, filepath.Join(root, "worktree/.git"), "gitdir: ../metadata\n")
	os.Symlink(root, filepath.Join(root, "cycle"))
	c.Depth = 1
	projects, w := scan(c)
	if len(w) != 0 || len(projects) != 2 {
		t.Fatalf("scan: %+v %v", projects, w)
	}
	if projects[0].URI != normal.URI {
		t.Fatal(projects)
	}
}
func TestUnicodeRankingAndKindFilter(t *testing.T) {
	a := Project{ID: "a", Name: "中文项目", Path: "/研发/中文项目", Kind: "folder"}
	b := Project{ID: "b", Name: "中文远程", Kind: "remote", Authority: "ssh-remote+devbox"}
	prefs := emptyPreferences()
	for _, tt := range []struct{ query, id string }{{"中文项目", "a"}, {"d 中项", "a"}, {"r devbox", "b"}} {
		got := rank([]Project{a, b}, prefs, tt.query, 50)
		if len(got) == 0 || got[0].ID != tt.id {
			t.Fatalf("%s: %v", tt.query, got)
		}
	}
}
func TestBrowseRootIdentityAndTraversal(t *testing.T) {
	c := fixture(t)
	c.Roots = []string{filepath.Join(c.Home, "one"), filepath.Join(c.Home, "two")}
	for _, r := range c.Roots {
		mkdir(t, filepath.Join(r, "same/child"))
	}
	f, e := browse(c, "/", emptyPreferences())
	if e != nil || len(f.Items) != 2 || f.Items[0].Autocomplete == f.Items[1].Autocomplete {
		t.Fatalf("roots: %v %v", f, e)
	}
	f, e = browse(c, "/1/same/", emptyPreferences())
	if e != nil || len(targets(f)) != 1 || !strings.Contains(targets(f)[0], "/one/") {
		t.Fatalf("drill: %v %v", f, e)
	}
	if _, e = browse(c, "/1/../../", emptyPreferences()); e == nil {
		t.Fatal("traversal allowed")
	}
}
func fakeEditor(t *testing.T, c *Config, name, body string) string {
	t.Helper()
	p := filepath.Join(c.Home, "Editor With Spaces", name)
	write(t, p, "#!/bin/sh\n"+body+"\n")
	os.Chmod(p, 0700)
	c.Editors[name] = p
	return p
}
func TestOpenPlansAndFailure(t *testing.T) {
	c := fixture(t)
	p := repo(t, filepath.Join(c.Home, "项目 space; literal"), "main")
	exe := fakeEditor(t, &c, "vscode", "exit 7")
	plan, e := PlanOpen(c, OpenRequest{Target: p.URI, Editor: "vscode", NewWindow: true})
	if e != nil || plan.Executable != exe || !reflect.DeepEqual(plan.Args, []string{"--new-window", "--", p.Path}) {
		t.Fatalf("argv: %+v %v", plan, e)
	}
	if ExecuteOpen(plan) == nil {
		t.Fatal("failed open reported success")
	}
	prefs, _ := loadPreferences(c)
	if len(prefs.Editors) != 0 {
		t.Fatal("failed launch changed preferences")
	}
	remote := "vscode-remote://ssh-remote+server/path/a%20b"
	plan, e = PlanOpen(c, OpenRequest{Target: remote, Editor: "vscode"})
	if e != nil || !reflect.DeepEqual(plan.Args, []string{"--folder-uri", remote}) {
		t.Fatalf("remote: %+v %v", plan, e)
	}
	fakeEditor(t, &c, "zed", "exit 0")
	plan, e = PlanOpen(c, OpenRequest{Target: remote, Editor: "zed", NewWindow: true})
	if e != nil || !reflect.DeepEqual(plan.Args, []string{"--new", "ssh://server/path/a%20b"}) {
		t.Fatalf("Zed: %+v %v", plan, e)
	}
	if _, e = PlanOpen(c, OpenRequest{Target: "vscode-remote://dev-container+opaque/work", Editor: "zed"}); e == nil {
		t.Fatal("container converted to SSH")
	}
}
func TestCLIErrorContract(t *testing.T) {
	c := fixture(t)
	t.Setenv("VSC_DATA_DIR", c.DataDir)
	t.Setenv("VSC_CACHE_DIR", c.CacheDir)
	t.Setenv("VSC_DEV_MODE", "")
	t.Setenv("VSC_DIRECTORIES", "")
	t.Setenv("VSC_IDE_PATH", c.VSCodeDir)
	t.Setenv("VSC_NO_BACKGROUND", "1")
	var out, errout bytes.Buffer
	if code := Run([]string{"open", "--target", "https://invalid"}, &out, &errout); code == 0 {
		t.Fatal("invalid open succeeded")
	}
	out.Reset()
	errout.Reset()
	if code := Run([]string{"query", "nothing"}, &out, &errout); code != 0 || !json.Valid(out.Bytes()) {
		t.Fatalf("query must produce JSON: %d %s", code, out.String())
	}
}
func TestConfigurationValidation(t *testing.T) {
	c := fixture(t)
	t.Setenv("VSC_DATA_DIR", c.DataDir)
	t.Setenv("VSC_DEV_MODE", "")
	t.Setenv("VSC_DIRECTORIES", `["~/Code","/tmp/a,b"]`)
	t.Setenv("VSC_LIMIT", "0")
	if _, e := LoadConfig(); e == nil {
		t.Fatal("invalid limit accepted")
	}
	t.Setenv("VSC_LIMIT", "50")
	got, e := LoadConfig()
	if e != nil || len(got.Roots) != 2 || got.Roots[1] != "/tmp/a,b" {
		t.Fatalf("roots: %+v %v", got, e)
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
func TestIndexIncompleteScanPreservesProjects(t *testing.T) {
	c := fixture(t)
	root := filepath.Join(c.Home, "root")
	c.Roots = []string{root}
	repo(t, filepath.Join(root, "p"), "main")
	if _, e := BuildIndex(c, false); e != nil {
		t.Fatal(e)
	}
	os.Rename(root, root+".away")
	index, e := BuildIndex(c, false)
	if e != nil || len(index.Projects) != 1 || len(index.Warnings) == 0 {
		t.Fatalf("incomplete scan lost snapshot: %+v %v", index, e)
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
