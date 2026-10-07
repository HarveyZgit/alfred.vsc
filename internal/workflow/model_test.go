package workflow

import (
	"path/filepath"
	"strings"
	"testing"
)

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
