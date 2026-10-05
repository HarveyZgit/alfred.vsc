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
