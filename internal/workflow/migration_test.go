package workflow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func legacyFixture(t *testing.T, c Config, records any, trash any) string {
	t.Helper()
	b, e := json.Marshal(map[string]any{"records": records, "trash": trash, "unknown_legacy_field": "retained"})
	if e != nil {
		t.Fatal(e)
	}
	name := filepath.Join(c.Home, "old workflow/.records.cache.json")
	write(t, name, string(b))
	return name
}

func TestMigrationPreviewScopeAndRawBackup(t *testing.T) {
	c := fixture(t)
	root := filepath.Join(c.Home, "root")
	c.Roots = []string{root}
	a := repo(t, filepath.Join(root, "a #100% 中文"), "main")
	outside := repo(t, filepath.Join(c.Home, "outside"), "main")
	records := []any{map[string]any{"path": "file://" + a.Path, "type": "folder"}, map[string]any{"path": outside.URI, "type": "folder"}, map[string]any{"path": "vscode-remote://ssh-remote+host/file.txt", "type": "remote"}}
	source := legacyFixture(t, c, records, map[string]any{"hidden": map[string]string{"path": "file://" + a.Path}, "outside": map[string]string{"path": outside.URI}})
	config := filepath.Join(filepath.Dir(source), ".user.config.json")
	write(t, config, `{"legacy_custom":"keep"}`)
	before, _ := os.ReadFile(source)
	r, e := ImportLegacy(c, filepath.Dir(source), false)
	if e != nil || r.OrderImported != 1 || r.HiddenImported != 1 || len(r.Unmigrated) != 3 {
		t.Fatalf("preview: %+v %v", r, e)
	}
	if _, e = os.Stat(c.DataDir); !os.IsNotExist(e) {
		t.Fatal("preview wrote destination")
	}
	for _, item := range r.Unmigrated {
		if item.Resolution == "" {
			t.Fatal("missing recovery instructions")
		}
	}
	r, e = ImportLegacy(c, source, true)
	if e != nil {
		t.Fatal(e)
	}
	p, _ := loadPreferences(c)
	if !p.Hidden[a.ID] || p.Hidden[outside.ID] || !reflect.DeepEqual(p.LegacyOrder, []string{a.ID}) {
		t.Fatalf("wrong imported scope: %+v", p)
	}
	copied, _ := os.ReadFile(filepath.Join(r.Backup, "legacy-records.json"))
	current, _ := os.ReadFile(source)
	if string(copied) != string(before) || string(current) != string(before) {
		t.Fatal("source altered or incomplete backup")
	}
	copied, _ = os.ReadFile(filepath.Join(r.Backup, "legacy-user-config.json"))
	if string(copied) != `{"legacy_custom":"keep"}` {
		t.Fatal("user config not retained")
	}
	if _, e = os.Stat(filepath.Join(r.Backup, "preferences-before.json")); e != nil {
		t.Fatal(e)
	}
}

func TestMigrationRetryAfterAddingSourcePreservesUserEdits(t *testing.T) {
	c := fixture(t)
	root := filepath.Join(c.Home, "root")
	c.Roots = []string{root}
	a := repo(t, filepath.Join(root, "a"), "main")
	b := repo(t, filepath.Join(c.Home, "other/b"), "main")
	source := legacyFixture(t, c, []any{map[string]string{"path": a.URI}, map[string]string{"path": b.URI}}, map[string]any{"a": map[string]string{"path": a.URI}, "b": map[string]string{"path": b.URI}})
	r, e := ImportLegacy(c, source, true)
	if e != nil {
		t.Fatal(e)
	}
	changePreferences(c, func(p *Preferences) error {
		delete(p.Hidden, a.ID)
		p.Pinned[a.ID] = true
		p.Editors[a.ID] = "zed"
		return nil
	})
	again, e := ImportLegacy(c, source, true)
	if e != nil || !again.AlreadyApplied {
		t.Fatalf("not idempotent: %v %v", again, e)
	}
	c.Roots = append(c.Roots, filepath.Dir(b.Path))
	again, e = ImportLegacy(c, source, true)
	if e != nil || again.AlreadyApplied || again.Backup != r.Backup || len(again.Unmigrated) != 0 {
		t.Fatalf("retry: %+v %v", again, e)
	}
	p, _ := loadPreferences(c)
	if p.Hidden[a.ID] || !p.Hidden[b.ID] || !p.Pinned[a.ID] || p.Editors[a.ID] != "zed" {
		t.Fatalf("user changes lost: %+v", p)
	}
	if !reflect.DeepEqual(p.LegacyOrder, []string{a.ID, b.ID}) {
		t.Fatal(p.LegacyOrder)
	}
}

func TestMigrationHistoryOrderYieldsToVSCodeChange(t *testing.T) {
	c := fixture(t)
	a := repo(t, filepath.Join(c.Home, "a"), "main")
	b := repo(t, filepath.Join(c.Home, "b"), "main")
	db := database(t, c)
	setHistory(t, db, "recently.opened", []any{map[string]string{"folderUri": a.URI}, map[string]string{"folderUri": b.URI}})
	source := legacyFixture(t, c, []any{map[string]string{"path": b.URI}, map[string]string{"path": a.URI}}, map[string]any{})
	if _, e := ImportLegacy(c, source, true); e != nil {
		t.Fatal(e)
	}
	f, e := Query(c, "")
	if e != nil || !reflect.DeepEqual(targets(f), []string{b.URI, a.URI}) {
		t.Fatalf("old usage order lost: %v %v", targets(f), e)
	}
	remote := "vscode-remote://ssh-remote+h/project"
	setHistory(t, db, "recently.opened", []any{map[string]string{"folderUri": remote}, map[string]string{"folderUri": a.URI}, map[string]string{"folderUri": b.URI}})
	f, e = Query(c, "")
	if e != nil || !reflect.DeepEqual(targets(f), []string{remote, a.URI, b.URI}) {
		t.Fatalf("history did not take over: %v %v", targets(f), e)
	}
}

func TestMigrationRemoteFilesNotImportedAndMenuURIs(t *testing.T) {
	c := fixture(t)
	db := database(t, c)
	remote := "vscode-remote://ssh-remote+host/dir%20space%23x"
	local := repo(t, filepath.Join(c.Home, "menu space"), "main")
	setHistory(t, db, "recently.opened", []any{map[string]string{"folderUri": remote}, map[string]string{"folderUri": local.URI}})
	source := legacyFixture(t, c, []any{
		map[string]any{"path": "vscode-remote://ssh-remote+host/dir space#x", "extra": map[string]string{"from": "getRecordsFromVscodeDB"}},
		map[string]any{"path": local.URI, "extra": map[string]string{"from": "getRecordsFromVscodeMenu"}},
		map[string]string{"path": "vscode-remote://ssh-remote+host/never-confirmed", "type": "remote"},
	}, map[string]any{"remote": map[string]string{"path": "vscode-remote://ssh-remote+host/dir space#x"}})
	r, e := ImportLegacy(c, source, true)
	if e != nil || r.HiddenImported != 1 || r.OrderImported != 2 || len(r.Unmigrated) != 1 {
		t.Fatalf("remote/menu: %+v %v", r, e)
	}
	if r.Items[0].Target != remote {
		t.Fatal(r.Items[0])
	}
}

func TestMigrationRejectsMalformedSourceAndCorruptDestination(t *testing.T) {
	c := fixture(t)
	source := filepath.Join(c.Home, "old.json")
	for _, raw := range []string{"{broken", `{}`, `{"records": "wrong"}`} {
		write(t, source, raw)
		if _, e := ImportLegacy(c, source, true); e == nil {
			t.Fatal("invalid source accepted")
		}
		if _, e := os.Stat(c.DataDir); !os.IsNotExist(e) {
			t.Fatal("invalid source created destination")
		}
	}
	write(t, source, `{"records":[],"trash":{}}`)
	write(t, filepath.Join(c.DataDir, "preferences.json"), "{broken")
	if _, e := ImportLegacy(c, source, true); e == nil {
		t.Fatal("overwrote corrupt destination")
	}
	b, _ := os.ReadFile(filepath.Join(c.DataDir, "preferences.json"))
	if string(b) != "{broken" {
		t.Fatal("corrupt preferences overwritten")
	}
}

func TestMigrationDestinationConfig(t *testing.T) {
	c := fixture(t)
	root := filepath.Join(c.Home, "configured-root")
	p := repo(t, filepath.Join(root, "p"), "main")
	b, _ := json.Marshal(map[string]any{"roots": []string{root}})
	write(t, filepath.Join(c.DataDir, "config.json"), string(b))
	t.Setenv("VSC_DATA_DIR", filepath.Join(c.Home, "wrong-default"))
	t.Setenv("VSC_DIRECTORIES", "")
	// Remove the overriding empty environment variable to exercise the destination config.
	os.Unsetenv("VSC_DIRECTORIES")
	loaded, e := loadConfigAt(c.DataDir)
	if e != nil || !reflect.DeepEqual(loaded.Roots, []string{root}) {
		t.Fatalf("destination config: %+v %v", loaded, e)
	}
	source := legacyFixture(t, c, []any{map[string]string{"path": p.URI}}, map[string]any{})
	r, e := ImportLegacy(loaded, source, false)
	if e != nil || r.OrderImported != 1 {
		t.Fatalf("wrong scope: %+v %v", r, e)
	}
	if !strings.HasPrefix(r.Backup, c.DataDir) {
		t.Fatal(r.Backup)
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
