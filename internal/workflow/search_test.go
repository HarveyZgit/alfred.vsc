package workflow

import (
	"reflect"
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

func TestExactNameRestrictsSearch(t *testing.T) {
	projects := []Project{
		{ID: "coding", Name: "coding", Path: "/Users/example/Work/Code/sample-workspace/coding", Kind: "folder"},
		{ID: "suffix", Name: "proxy-backend", Path: "/work/proxy-backend", Kind: "folder"},
		{ID: "nested", Name: "ai-kit", Path: "/work/proxy-backend/packages/ai-kit", Kind: "folder"},
		{ID: "exact", Name: "backend", Path: "/work/backend", Kind: "folder"},
		{ID: "remote", Name: "Backend", Path: "/srv/Backend", Kind: "remote", Authority: "ssh-remote+devbox"},
	}
	for _, tt := range []struct {
		name, query string
		hidden      []string
		limit       int
		want        []string
	}{
		{"exact", "backend", nil, 50, []string{"exact", "remote"}},
		{"case and whitespace", "  BACKEND  ", nil, 50, []string{"exact", "remote"}},
		{"local scope", "d backend", nil, 50, []string{"exact"}},
		{"remote scope", "r backend", nil, 50, []string{"remote"}},
		{"limit after exact selection", "backend", nil, 1, []string{"exact"}},
		{"hidden exact", "backend", []string{"exact"}, 50, []string{"remote"}},
		{"fuzzy fallback", "bcknd", nil, 1, []string{"exact"}},
		{"path fallback", "proxy-backend/packages", nil, 50, []string{"nested"}},
		{"authority fallback", "r devbox", nil, 50, []string{"remote"}},
		{"hidden exacts permit fallback", "backend", []string{"exact", "remote"}, 1, []string{"suffix"}},
		{"empty retains pin order", "", nil, 1, []string{"coding"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			prefs := emptyPreferences()
			prefs.Pinned["coding"] = true
			prefs.LegacyOrder = []string{"coding", "suffix", "nested", "exact", "remote"}
			for _, id := range tt.hidden {
				prefs.Hidden[id] = true
			}
			got := []string{}
			for _, p := range rank(projects, prefs, tt.query, tt.limit) {
				got = append(got, p.ID)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("rank(%q) = %v, want %v", tt.query, got, tt.want)
			}
		})
	}
}
