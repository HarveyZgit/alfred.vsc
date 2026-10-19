package workflow

import (
	"path/filepath"
	"strings"
	"testing"
)

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

func TestBrowseSpaceSeparatedWords(t *testing.T) {
	c := fixture(t)
	c.Roots = []string{filepath.Join(c.Home, "projects")}
	mkdir(t, filepath.Join(c.Roots[0], "proxy-backend"))
	mkdir(t, filepath.Join(c.Roots[0], "中文项目"))
	for _, tt := range []struct {
		query string
		count int
	}{
		{"/back end", 1}, {"/END back", 1}, {"/bcknd", 0},
		{"/back missing", 0}, {"/中 项", 1}, {"/中项", 0},
	} {
		feedback, err := browse(c, tt.query, emptyPreferences())
		if err != nil {
			t.Fatal(err)
		}
		if got := len(targets(feedback)); got != tt.count {
			t.Fatalf("browse(%q) returned %d projects, want %d", tt.query, got, tt.count)
		}
	}
}
