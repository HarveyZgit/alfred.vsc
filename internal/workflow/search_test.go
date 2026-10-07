package workflow

import (
	"testing"
)

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
