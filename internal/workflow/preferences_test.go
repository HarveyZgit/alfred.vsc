package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestPreferenceConcurrentWriters(t *testing.T) {
	c := fixture(t)
	var wg sync.WaitGroup
	errs := make(chan error, 30)
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errs <- changePreferences(c, func(p *Preferences) error { p.Hidden[fmt.Sprint(i)] = true; return nil })
		}(i)
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		if e != nil {
			t.Fatal(e)
		}
	}
	p, e := loadPreferences(c)
	if e != nil || len(p.Hidden) != 30 {
		t.Fatalf("lost updates: %d %v", len(p.Hidden), e)
	}
}

func TestCorruptPreferencesAreNotOverwritten(t *testing.T) {
	c := fixture(t)
	name := filepath.Join(c.DataDir, "preferences.json")
	write(t, name, "{broken")
	if e := changePreferences(c, func(p *Preferences) error { return nil }); e == nil {
		t.Fatal("expected error")
	}
	b, _ := os.ReadFile(name)
	if string(b) != "{broken" {
		t.Fatal("corrupt preferences overwritten")
	}
}
