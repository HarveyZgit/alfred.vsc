package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"syscall"
	"testing"
)

func TestUnavailableProjectsOnlyConfirmedFailures(t *testing.T) {
	base := t.TempDir()
	file := filepath.Join(base, "file")
	write(t, file, "keep this file")
	errors := map[string]error{"missing": os.ErrNotExist, "denied": os.ErrPermission, "io": syscall.EIO, "parent-file": syscall.ENOTDIR}
	projects := []Project{{ID: "dir", Path: base, Kind: "folder"}, {ID: "file", Path: file, Kind: "folder"}, {ID: "remote", Path: "must-not-probe", Kind: "remote"}}
	for path := range errors {
		projects = append(projects, Project{ID: path, Path: path, Kind: "folder"})
	}
	ids, skipped := unavailableProjects(context.Background(), projects, func(path string) (os.FileInfo, error) {
		if path == "must-not-probe" {
			t.Error("probed remote path")
		}
		if err, ok := errors[path]; ok {
			return nil, &os.PathError{Op: "stat", Path: path, Err: err}
		}
		return os.Stat(path)
	})
	slices.Sort(ids)
	if !reflect.DeepEqual(ids, []string{"file", "missing", "parent-file"}) || skipped != 2 {
		t.Fatalf("candidates=%v skipped=%d", ids, skipped)
	}
}

func TestUnavailableProjectsDoesNotWaitForSlowProbe(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	release, finished := make(chan struct{}), make(chan struct{})
	ids, skipped := unavailableProjects(ctx, []Project{{ID: "slow", Kind: "folder"}}, func(string) (os.FileInfo, error) {
		cancel()
		<-release
		defer close(finished)
		return nil, os.ErrNotExist
	})
	close(release)
	<-finished
	if len(ids) != 0 || skipped != 1 {
		t.Fatalf("unfinished probe was classified as missing: %v %d", ids, skipped)
	}
}

func TestPanelCleanupAllSourcesAndRestore(t *testing.T) {
	c, p := panelFixture(t)
	c.Roots = []string{filepath.Join(c.Home, "root")}
	t.Setenv("VSC_DIRECTORIES", c.Roots[0])
	if err := atomicJSON(filepath.Join(c.DataDir, "config.json"), c); err != nil {
		t.Fatal(err)
	}
	indexed := repo(t, filepath.Join(c.Roots[0], "stale"), "main")
	if _, err := BuildIndex(c, false); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(indexed.Path); err != nil {
		t.Fatal(err)
	}
	db := database(t, c)
	live := repo(t, filepath.Join(c.Home, "live"), "main")
	remote, _ := project("vscode-remote://ssh-remote+host/not-on-this-machine", "")
	file, _ := localProject(filepath.Join(c.Home, "now-a-file"))
	write(t, file.Path, "keep me")
	already, _ := localProject(filepath.Join(c.Home, "already-hidden"))
	entries := []map[string]string{{"folderUri": live.URI}, {"folderUri": remote.URI}, {"folderUri": file.URI}, {"folderUri": already.URI}}
	missing := []Project{}
	for i := 0; i < 60; i++ {
		item, _ := localProject(filepath.Join(c.Home, fmt.Sprintf("missing-%02d", i)))
		missing = append(missing, item)
		entries = append(entries, map[string]string{"folderUri": item.URI})
	}
	setHistory(t, db, "recently.opened", entries)
	if err := changePreferences(c, func(prefs *Preferences) error {
		prefs.Hidden[already.ID] = true
		prefs.Pinned[file.ID] = true
		prefs.Editors[file.ID] = "zed"
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	var before string
	if err := db.QueryRow("SELECT value FROM ItemTable WHERE key='recently.opened'").Scan(&before); err != nil {
		t.Fatal(err)
	}
	for i, want := range []int{62, 0} {
		// Cleaning is independent of pagination and the current UI search/filter.
		w := panelRequest(t, p, "POST", "/api/records/cleanup?q=live&filter=remote&page=99", map[string]any{})
		var report cleanupResult
		if err := json.Unmarshal(w.Body.Bytes(), &report); err != nil || w.Code != 200 || report.Hidden != want || report.Skipped != 0 {
			t.Fatalf("cleanup %d: %d %s", i, w.Code, w.Body.String())
		}
	}
	prefs, err := loadPreferences(c)
	if err != nil || len(prefs.Hidden) != 63 || prefs.Hidden[live.ID] || prefs.Hidden[remote.ID] || !prefs.Hidden[indexed.ID] || !prefs.Pinned[file.ID] || prefs.Editors[file.ID] != "zed" {
		t.Fatalf("unexpected preferences: %+v %v", prefs, err)
	}
	var after string
	if err := db.QueryRow("SELECT value FROM ItemTable WHERE key='recently.opened'").Scan(&after); err != nil || after != before {
		t.Fatal("VS Code history changed", err)
	}
	if data, err := os.ReadFile(file.Path); err != nil || string(data) != "keep me" {
		t.Fatal("cleanup deleted a file", err)
	}
	mkdir(t, missing[0].Path)
	w := panelRequest(t, p, "POST", "/api/record", map[string]any{"id": missing[0].ID, "action": "hidden", "enabled": false})
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	feedback, err := Query(c, missing[0].Name)
	if err != nil || !reflect.DeepEqual(targets(feedback), []string{missing[0].URI}) {
		t.Fatalf("restore failed: %+v %v", feedback, err)
	}
}

func TestPanelCleanupRejectsInvalidRequestsAndBusy(t *testing.T) {
	_, p := panelFixture(t)
	for _, test := range []struct {
		method string
		body   any
	}{{http.MethodGet, nil}, {http.MethodPost, map[string]any{"path": "/outside"}}} {
		if w := panelRequest(t, p, test.method, "/api/records/cleanup", test.body); w.Code != 400 {
			t.Fatalf("invalid cleanup accepted: %d", w.Code)
		}
	}
	p.busy.Lock()
	defer p.busy.Unlock()
	if w := panelRequest(t, p, "POST", "/api/records/cleanup", map[string]any{}); w.Code != 400 {
		t.Fatal("cleanup overlapped another scan")
	}
}
