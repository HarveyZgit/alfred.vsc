package workflow

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
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
func readJSON(name string, v any) error {
	b, e := os.ReadFile(name)
	if e != nil {
		return e
	}
	if len(b) > 32<<20 {
		return fmt.Errorf("state exceeds 32 MiB")
	}
	return json.Unmarshal(b, v)
}
func atomicJSON(name string, v any) error {
	b, e := json.Marshal(v)
	if e != nil {
		return e
	}
	return atomicBytes(name, b)
}

// Cache encoding is internal and disposable; preferences stay human-readable JSON.
func readCache(name string, v any) error {
	b, e := os.ReadFile(name)
	if e != nil {
		return e
	}
	if len(b) > 32<<20 || len(b) < 4 || string(b[:4]) != "VSC1" {
		return fmt.Errorf("invalid cache format")
	}
	return gob.NewDecoder(bytes.NewReader(b[4:])).Decode(v)
}
func atomicCache(name string, v any) error {
	var b bytes.Buffer
	b.WriteString("VSC1")
	if e := gob.NewEncoder(&b).Encode(v); e != nil {
		return e
	}
	return atomicBytes(name, b.Bytes())
}
func atomicBytes(name string, b []byte) error {
	dir := filepath.Dir(name)
	if e := os.MkdirAll(dir, 0700); e != nil {
		return e
	}
	f, e := os.CreateTemp(dir, ".write-*")
	if e != nil {
		return e
	}
	defer os.Remove(f.Name())
	if _, e = f.Write(b); e != nil {
		f.Close()
		return e
	}
	if e = f.Sync(); e != nil {
		f.Close()
		return e
	}
	if e = f.Close(); e != nil {
		return e
	}
	return os.Rename(f.Name(), name)
}
func lockFile(name string, nonblocking bool) (*os.File, error) {
	if e := os.MkdirAll(filepath.Dir(name), 0700); e != nil {
		return nil, e
	}
	f, e := os.OpenFile(name, os.O_CREATE|os.O_RDWR, 0600)
	if e != nil {
		return nil, e
	}
	flags := syscall.LOCK_EX
	if nonblocking {
		flags |= syscall.LOCK_NB
	}
	if e = syscall.Flock(int(f.Fd()), flags); e != nil {
		f.Close()
		return nil, e
	}
	return f, nil
}
func unlock(f *os.File) { syscall.Flock(int(f.Fd()), syscall.LOCK_UN); f.Close() }
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
