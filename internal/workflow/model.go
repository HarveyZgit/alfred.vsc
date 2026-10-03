package workflow

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"path"
	"path/filepath"
	"strings"
)

type Project struct {
	ID        string `json:"id"`
	URI       string `json:"uri"`
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	Path      string `json:"path,omitempty"`
	Authority string `json:"authority,omitempty"`
	Label     string `json:"label,omitempty"`
}

func project(raw, label string) (Project, error) {
	u, e := url.Parse(raw)
	if e != nil {
		return Project{}, e
	}
	p := Project{Label: label}
	switch u.Scheme {
	case "file":
		if u.Host != "" && u.Host != "localhost" {
			return p, fmt.Errorf("unsupported file URI host")
		}
		if !filepath.IsAbs(u.Path) {
			return p, fmt.Errorf("local directory must be absolute")
		}
		p.Path = filepath.Clean(u.Path)
		if resolved, e := filepath.EvalSymlinks(p.Path); e == nil {
			p.Path = resolved
		}
		p.Kind = "folder"
		p.URI = (&url.URL{Scheme: "file", Path: p.Path}).String()
		p.Name = filepath.Base(p.Path)
	case "vscode-remote":
		if u.Host == "" || u.Path == "" || !strings.HasPrefix(u.Path, "/") {
			return p, fmt.Errorf("remote URI needs authority and absolute path")
		}
		if u.User != nil || u.RawQuery != "" || u.Fragment != "" {
			return p, fmt.Errorf("unsupported remote URI components")
		}
		p.Kind = "remote"
		p.Path = u.Path
		p.URI = u.String()
		p.Authority = u.Host
		p.Name = path.Base(strings.TrimSuffix(u.Path, "/"))
	default:
		return p, fmt.Errorf("only local and VS Code remote directory URIs are supported")
	}
	sum := sha256.Sum256([]byte(p.URI))
	p.ID = hex.EncodeToString(sum[:16])
	return p, nil
}
func localProject(dir string) (Project, error) {
	abs, e := filepath.Abs(dir)
	if e != nil {
		return Project{}, e
	}
	return project((&url.URL{Scheme: "file", Path: abs}).String(), "")
}
func merge(groups ...[]Project) []Project {
	total := 0
	for _, group := range groups {
		total += len(group)
	}
	out := make([]Project, 0, total)
	seen := make(map[string]bool, total)
	for _, group := range groups {
		for _, p := range group {
			if !seen[p.ID] {
				out = append(out, p)
				seen[p.ID] = true
			}
		}
	}
	return out
}
func remoteLabel(p Project) string {
	if strings.HasPrefix(p.Authority, "ssh-remote+") {
		return "SSH: " + strings.TrimPrefix(p.Authority, "ssh-remote+")
	}
	if strings.HasPrefix(p.Authority, "dev-container+") || strings.HasPrefix(p.Authority, "attached-container+") {
		if p.Label != "" {
			return "Container: " + p.Label
		}
		return "Container · " + p.Name
	}
	return "Remote: " + p.Authority
}
