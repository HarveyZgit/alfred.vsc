package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
)

func TestRecentProjectsNormalizeAndBound(t *testing.T) {
	first := fmt.Sprintf("%032x", 1)
	second := fmt.Sprintf("%032x", 2)
	got := recentProjects([]string{"", "invalid", first, first, second})
	if !reflect.DeepEqual(got, []string{first, second}) {
		t.Fatalf("invalid IDs or duplicates retained: %v", got)
	}
	prefs := emptyPreferences()
	for i := 0; i < recentProjectLimit+10; i++ {
		rememberProject(&prefs, fmt.Sprintf("%032x", i))
	}
	if len(prefs.Recent) != recentProjectLimit || prefs.Recent[0] != fmt.Sprintf("%032x", recentProjectLimit+9) {
		t.Fatal("recent history is not bounded or newest first")
	}
	selected := prefs.Recent[17]
	rememberProject(&prefs, selected)
	if prefs.Recent[0] != selected || len(prefs.Recent) != recentProjectLimit || len(recentProjects(prefs.Recent)) != len(prefs.Recent) {
		t.Fatal("reselection must move, not duplicate, an ID")
	}
}

func TestRecentPreferencesConcurrentWrites(t *testing.T) {
	c := fixture(t)
	if err := changePreferences(c, func(p *Preferences) error {
		p.Pinned["keep"] = true
		p.Editors["keep"] = "zed"
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	errs := make(chan error, 30)
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errs <- changePreferences(c, func(p *Preferences) error {
				id := fmt.Sprintf("%032x", i)
				rememberProject(p, id)
				p.Hidden[id] = true
				return nil
			})
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	p, err := loadPreferences(c)
	if err != nil || len(p.Recent) != 30 || len(p.Hidden) != 30 || !p.Pinned["keep"] || p.Editors["keep"] != "zed" {
		t.Fatalf("concurrent update lost preferences: %+v %v", p, err)
	}
}

func TestMalformedRecentPreferencesAreNotOverwritten(t *testing.T) {
	c := fixture(t)
	name := filepath.Join(c.DataDir, "preferences.json")
	for _, value := range []string{`{"version":1,"recent_projects":false}`, `{"version":1,"recent_projects":[42]}`} {
		write(t, name, value)
		if err := changePreferences(c, func(p *Preferences) error { rememberProject(p, fmt.Sprintf("%032x", 1)); return nil }); err == nil {
			t.Fatal("malformed list accepted")
		}
		b, _ := os.ReadFile(name)
		if string(b) != value {
			t.Fatal("malformed state overwritten")
		}
	}
	write(t, name, `{"version":1,"recent_projects":["bad","00000000000000000000000000000001","00000000000000000000000000000001"]}`)
	p, err := loadPreferences(c)
	if err != nil || len(p.Recent) != 1 {
		t.Fatalf("invalid/duplicate IDs were not normalized: %+v %v", p, err)
	}
}

func TestRecentRankingScopeAndLimit(t *testing.T) {
	projects := []Project{
		{ID: "short", Name: "github", Path: "/projects/github", Kind: "folder"},
		{ID: "chosen", Name: "example-material-hub", Path: "/projects/example-material-hub", Kind: "folder"},
		{ID: "remote", Name: "github", Path: "/projects/github", Kind: "remote"},
		{ID: "other", Name: "other", Path: "/projects/other", Kind: "folder"},
	}
	for _, tt := range []struct {
		name, query    string
		pinned, hidden []string
		want           []string
	}{
		{"selected substring before shorter name", "hub", nil, nil, []string{"chosen", "short", "remote"}},
		{"global empty query", "", nil, nil, []string{"chosen", "short", "remote", "other"}},
		{"global different query", "projects", nil, nil, []string{"chosen", "other", "short", "remote"}},
		{"unmatched selection excluded", "other", nil, nil, []string{"other"}},
		{"exact name still narrows", "github", nil, nil, []string{"short", "remote"}},
		{"type filter retained", "r hub", nil, nil, []string{"remote"}},
		{"hidden selection excluded", "hub", nil, []string{"chosen"}, []string{"short", "remote"}},
		{"pinned priority", "hub", []string{"short"}, nil, []string{"short", "chosen", "remote"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			prefs := emptyPreferences()
			prefs.Recent = []string{"stale", "chosen"}
			for _, id := range tt.pinned {
				prefs.Pinned[id] = true
			}
			for _, id := range tt.hidden {
				prefs.Hidden[id] = true
			}
			got := []string{}
			for _, p := range rank(projects, prefs, tt.query, 50) {
				got = append(got, p.ID)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
	for i := 0; i < 60; i++ {
		projects = append(projects, Project{ID: fmt.Sprint(i), Name: fmt.Sprintf("hub-%d", i), Kind: "folder"})
	}
	prefs := emptyPreferences()
	if rank(projects, prefs, "hub", 1)[0].ID == "chosen" {
		t.Fatal("fixture must rank chosen project below the old limit")
	}
	prefs.Recent = []string{"chosen"}
	if rank(projects, prefs, "hub", 1)[0].ID != "chosen" {
		t.Fatal("MRU applied after limiting candidates")
	}
	prefs.Pinned["chosen"] = true
	prefs.Recent = nil
	if rank(projects, prefs, "hub", 1)[0].ID != "chosen" {
		t.Fatal("pin must outrank relevance within matching candidates")
	}
}

func BenchmarkRecentRanking(b *testing.B) {
	for _, n := range []int{1000, 10000} {
		projects := make([]Project, n)
		prefs := emptyPreferences()
		for i := range projects {
			id := fmt.Sprintf("%032x", i)
			projects[i] = Project{ID: id, Name: fmt.Sprintf("example-%d-hub", i), Kind: "folder"}
			if i < recentProjectLimit {
				prefs.Recent = append(prefs.Recent, id)
			}
		}
		b.Run(fmt.Sprintf("projects_%d", n), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				rank(projects, prefs, "hub", 50)
			}
		})
	}
}
