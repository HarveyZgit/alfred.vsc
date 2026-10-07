package workflow

import (
	"path/filepath"
	"strings"
	"testing"
)

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
