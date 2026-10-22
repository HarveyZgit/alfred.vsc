package workflow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestUnlistedAbsolutePathUsesDefaultEditorWithoutImport(t *testing.T) {
	c := fixture(t)
	c.Editor = "cursor"
	c.Roots = []string{filepath.Join(c.Home, "nonexistent-configured-root")}
	target := repo(t, filepath.Join(c.Home, "outside", "项目 space; $(whoami) # % '"), "fresh")
	// An unrelated record and stale per-project preference must not displace the
	// requested directory or change the default for an unlisted target.
	db := database(t, c)
	other := repo(t, filepath.Join(c.Home, "recorded"), "main")
	setHistory(t, db, "recently.opened", []map[string]string{{"folderUri": other.URI}})
	if err := changePreferences(c, func(p *Preferences) error {
		p.Editors[target.ID] = "zed"
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	prefsBefore, err := os.ReadFile(filepath.Join(c.DataDir, "preferences.json"))
	if err != nil {
		t.Fatal(err)
	}
	capture := filepath.Join(c.Home, "opened-args")
	t.Setenv("VSC_TEST_ARGS", capture)
	fakeEditor(t, &c, "cursor", `printf '%s\n' "$@" > "$VSC_TEST_ARGS"`)
	for _, input := range []string{target.Path, target.Path + "/", "~/outside/" + filepath.Base(target.Path)} {
		feedback, err := Query(c, input)
		if err != nil || len(feedback.Items) != 1 || !feedback.Items[0].Valid || !feedback.SkipKnowledge {
			t.Fatalf("%q: %+v %v", input, feedback, err)
		}
		item := feedback.Items[0]
		if !strings.Contains(item.Title, "Cursor") || !strings.Contains(item.Title, "打开输入的目录") || !strings.Contains(item.Subtitle, "fresh") {
			t.Fatalf("missing prompt or fresh branch: %+v", item)
		}
		var request OpenRequest
		if err := json.Unmarshal([]byte(item.Arg), &request); err != nil || request.Target != target.URI || request.Editor != "cursor" {
			t.Fatalf("wrong open request: %+v %v", request, err)
		}
		if _, err := os.Stat(capture); !os.IsNotExist(err) {
			t.Fatal("query opened IDE before user confirmation")
		}
		plan, err := PlanOpen(c, request)
		if err != nil {
			t.Fatal(err)
		}
		if err := ExecuteOpen(plan); err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(capture)
		if err != nil || string(got) != "--\n"+target.Path+"\n" {
			t.Fatalf("wrong IDE argv: %q %v", got, err)
		}
		if err := os.Remove(capture); err != nil {
			t.Fatal(err)
		}
	}
	prefsAfter, err := os.ReadFile(filepath.Join(c.DataDir, "preferences.json"))
	if err != nil || string(prefsBefore) != string(prefsAfter) {
		t.Fatal("temporary open changed preferences", err)
	}
	history, _ := loadHistory(c)
	index, _ := loadIndex(c)
	if len(history) != 1 || history[0].ID != other.ID || len(index.Projects) != 0 {
		t.Fatal("temporary open imported the directory")
	}
}

func TestAbsolutePathKnownHiddenAndInvalidTargets(t *testing.T) {
	c := fixture(t)
	c.Roots = []string{filepath.Join(c.Home, "missing-root-a"), filepath.Join(c.Home, "missing-root-b")}
	db := database(t, c)
	target := repo(t, filepath.Join(c.Home, "known"), "main")
	child := repo(t, filepath.Join(target.Path, "nested"), "main")
	setHistory(t, db, "recently.opened", []map[string]string{{"folderUri": child.URI}, {"folderUri": target.URI}})
	if err := changePreferences(c, func(p *Preferences) error {
		p.Editors[target.ID] = "zed"
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	f, err := Query(c, target.Path+"/")
	if err != nil || !reflect.DeepEqual(targets(f), []string{target.URI}) || f.Items[0].Title != target.Name {
		t.Fatalf("known exact path: %+v %v", f, err)
	}
	var request OpenRequest
	if err := json.Unmarshal([]byte(f.Items[0].Arg), &request); err != nil || request.Editor != "zed" {
		t.Fatal("known project lost its configured IDE", err)
	}
	if err := changePreferences(c, func(p *Preferences) error {
		p.Hidden[target.ID] = true
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	f, err = Query(c, target.Path)
	if err != nil || !f.Items[0].Valid || !strings.Contains(f.Items[0].Title, "打开输入的目录") {
		t.Fatalf("explicit hidden path: %+v %v", f, err)
	}
	prefs, _ := loadPreferences(c)
	if !prefs.Hidden[target.ID] {
		t.Fatal("explicit path unexpectedly unhid a record")
	}
	file := filepath.Join(c.Home, "file")
	write(t, file, "not a directory")
	for _, test := range []struct{ path, title string }{
		{filepath.Join(c.Home, "no-such-directory"), "目录不存在"},
		{file, "路径不是目录"},
		{filepath.Join(file, "child"), "路径不是目录"},
	} {
		f, err := Query(c, test.path)
		if err != nil || len(f.Items) != 1 || f.Items[0].Valid || f.Items[0].Title != test.title || f.Items[0].Arg != "" {
			t.Fatalf("invalid path %q: %+v %v", test.path, f, err)
		}
	}
}

func TestBrowseAutocompleteUsesExplicitRelativeNamespace(t *testing.T) {
	c := fixture(t)
	c.Roots = []string{filepath.Join(c.Home, "one"), filepath.Join(c.Home, "two")}
	for _, root := range c.Roots {
		mkdir(t, filepath.Join(root, "child/grandchild"))
	}
	f, err := Query(c, "/")
	if err != nil || len(f.Items) != 2 {
		t.Fatalf("root picker: %+v %v", f, err)
	}
	for _, root := range f.Items {
		if !strings.HasPrefix(root.Autocomplete, "./") {
			t.Fatalf("ambiguous root completion: %+v", root)
		}
		children, err := Query(c, root.Autocomplete)
		if err != nil || len(targets(children)) != 1 {
			t.Fatalf("root navigation: %+v %v", children, err)
		}
		grandchildren, err := Query(c, children.Items[0].Autocomplete)
		if err != nil || len(targets(grandchildren)) != 1 || !strings.HasPrefix(grandchildren.Items[0].Autocomplete, root.Autocomplete) {
			t.Fatalf("child navigation: %+v %v", grandchildren, err)
		}
	}
}

func TestDirectPathWithoutRootsSymlinkAndIndexedRecord(t *testing.T) {
	c := fixture(t)
	target := repo(t, filepath.Join(c.Home, "actual"), "main")
	alias := filepath.Join(c.Home, "alias")
	if err := os.Symlink(target.Path, alias); err != nil {
		t.Fatal(err)
	}
	home, err := localProject(c.Home)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct{ input, uri string }{{target.Path, target.URI}, {alias, target.URI}, {"~", home.URI}, {"~/", home.URI}} {
		f, err := Query(c, test.input)
		if err != nil || !reflect.DeepEqual(targets(f), []string{test.uri}) || !f.Items[0].Valid {
			t.Fatalf("no-root path %q: %+v %v", test.input, f, err)
		}
	}
	c.Roots = []string{target.Path}
	if _, err := BuildIndex(c, false); err != nil {
		t.Fatal(err)
	}
	f, err := Query(c, alias)
	if err != nil || !reflect.DeepEqual(targets(f), []string{target.URI}) || f.Items[0].Title != target.Name {
		t.Fatalf("indexed path: %+v %v", f, err)
	}
}
