package workflow

import (
	"path/filepath"
	"reflect"
	"testing"
)

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
