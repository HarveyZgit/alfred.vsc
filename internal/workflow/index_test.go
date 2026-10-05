package workflow

import (
	"os"
	"path/filepath"
	"testing"
)

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
