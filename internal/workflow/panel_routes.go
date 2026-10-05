package workflow

import (
	"crypto/subtle"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

//go:embed panel/*
var panelAssets embed.FS

func (p *panel) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'self'; style-src 'self'; connect-src 'self'; img-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'none'")
	if r.Host != p.host || (r.Header.Get("Origin") != "" && r.Header.Get("Origin") != "http://"+p.host) {
		http.Error(w, "禁止跨站访问", http.StatusForbidden)
		return
	}
	if !strings.HasPrefix(r.URL.Path, "/api/") {
		name := map[string]string{
			"/":             "index.html",
			"/app.js":       "app.js",
			"/style.css":    "style.css",
			"/api.js":       "api.js",
			"/dom.js":       "dom.js",
			"/records.js":   "records.js",
			"/settings.js":  "settings.js",
			"/migration.js": "migration.js",
		}[r.URL.Path]
		if name == "" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method", 405)
			return
		}
		body, err := panelAssets.ReadFile("panel/" + name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		contentType := "text/javascript; charset=utf-8"
		if name == "index.html" {
			contentType = "text/html; charset=utf-8"
		} else if name == "style.css" {
			contentType = "text/css; charset=utf-8"
		}
		w.Header().Set("Content-Type", contentType)
		_, _ = w.Write(body)
		return
	}
	if subtle.ConstantTimeCompare([]byte(r.Header.Get("Authorization")), []byte("Bearer "+p.token)) != 1 {
		http.Error(w, "会话失效，请从 Alfred 重新打开面板", http.StatusUnauthorized)
		return
	}
	p.touched.Store(time.Now().Unix())
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	result, err := p.api(w, r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = encode(w, map[string]string{"error": err.Error()})
		return
	}
	_ = encode(w, result)
}

func panelBody(w http.ResponseWriter, r *http.Request, dst any) error {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		return fmt.Errorf("需要 JSON 请求")
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return fmt.Errorf("请求只能包含一个 JSON 对象")
	}
	return nil
}

func (p *panel) api(w http.ResponseWriter, r *http.Request) (any, error) {
	c, err := loadConfigAt(p.dataDir)
	if err != nil {
		return nil, err
	}
	switch r.Method + " " + r.URL.Path {
	case "GET /api/settings":
		return c, nil
	case "GET /api/records":
		return panelRecords(c, r)
	case "GET /api/migrations":
		return panelMigrations(c)
	case "POST /api/settings":
		return panelSaveSettings(c, w, r)
	case "POST /api/settings/reset":
		if err = saveManaged(c, nil); err != nil {
			return nil, err
		}
		return map[string]bool{"ok": true}, nil
	case "POST /api/record":
		return panelChangeRecord(c, w, r)
	case "POST /api/rebuild":
		if !p.busy.TryLock() {
			return nil, fmt.Errorf("正在扫描或迁移，请稍后重试")
		}
		defer p.busy.Unlock()
		return BuildIndex(c, false)
	case "POST /api/migrate":
		return p.panelMigrate(c, w, r)
	case "POST /api/close":
		p.stop()
		return map[string]bool{"ok": true}, nil
	default:
		return nil, fmt.Errorf("未知接口或请求方法")
	}
}
