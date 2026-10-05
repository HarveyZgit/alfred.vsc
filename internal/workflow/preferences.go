package workflow

import (
	"fmt"
	"os"
	"path/filepath"
)

type Preferences struct {
	Version       int               `json:"version"`
	Hidden        map[string]bool   `json:"hidden"`
	Pinned        map[string]bool   `json:"pinned"`
	Editors       map[string]string `json:"editors"`
	Migrated      bool              `json:"legacy_migrated"`
	LegacyImports map[string]bool   `json:"legacy_imports,omitempty"`
	LegacyOrder   []string          `json:"legacy_order,omitempty"`
	LegacyHistory string            `json:"legacy_history,omitempty"`
}

func emptyPreferences() Preferences {
	return Preferences{Version: 1, Hidden: map[string]bool{}, Pinned: map[string]bool{}, Editors: map[string]string{}}
}

func loadPreferences(c Config) (Preferences, error) {
	p := emptyPreferences()
	e := readJSON(filepath.Join(c.DataDir, "preferences.json"), &p)
	if os.IsNotExist(e) {
		return p, nil
	}
	if e != nil {
		return p, e
	}
	if p.Version != 1 {
		return p, fmt.Errorf("unsupported preferences version %d", p.Version)
	}
	if p.Hidden == nil {
		p.Hidden = map[string]bool{}
	}
	if p.Pinned == nil {
		p.Pinned = map[string]bool{}
	}
	if p.Editors == nil {
		p.Editors = map[string]string{}
	}
	return p, nil
}

func changePreferences(c Config, fn func(*Preferences) error) error {
	lock, e := lockFile(filepath.Join(c.DataDir, "preferences.lock"), false)
	if e != nil {
		return e
	}
	defer unlock(lock)
	p, e := loadPreferences(c)
	if e != nil {
		return e
	}
	if e = fn(&p); e != nil {
		return e
	}
	return atomicJSON(filepath.Join(c.DataDir, "preferences.json"), p)
}
