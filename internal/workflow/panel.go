package workflow

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/subtle"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

//go:embed panel/*
var panelAssets embed.FS

type panel struct {
	dataDir, token, host string
	stop                 func()
	touched              atomic.Int64
	busy                 sync.Mutex // Serialize explicit scans and migration, outside the query process.
}

func launchPanel(c Config) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if err = os.MkdirAll(c.CacheDir, 0700); err != nil {
		return err
	}
	log, err := os.OpenFile(filepath.Join(c.CacheDir, "panel.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return err
	}
	defer log.Close()
	cmd := exec.Command(exe, "manage", "--serve")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	cmd.Stderr = log
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err = cmd.Start(); err != nil {
		return err
	}
	ready := make(chan string, 1)
	go func() { line, _ := bufio.NewReader(stdout).ReadString('\n'); ready <- strings.TrimSpace(line) }()
	var address string
	select {
	case address = <-ready:
	case <-time.After(5 * time.Second):
	}
	stdout.Close()
	if !strings.HasPrefix(address, "http://127.0.0.1:") {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return fmt.Errorf("管理面板启动失败，请查看 %s", filepath.Join(c.CacheDir, "panel.log"))
	}
	opener := "xdg-open"
	if c.OS == "darwin" {
		opener = "open"
	}
	if err = exec.Command(opener, address).Run(); err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return fmt.Errorf("无法打开浏览器：%w；可在终端运行 vsc manage --serve 后手动打开输出的网址", err)
	}
	// Reap when embedded in a longer lived caller; the short CLI can exit immediately.
	go func() { _ = cmd.Wait() }()
	return nil
}

func servePanel(c Config, out io.Writer) error {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return err
	}
	defer listener.Close()
	random := make([]byte, 32)
	if _, err = rand.Read(random); err != nil {
		return err
	}
	p := &panel{dataDir: c.DataDir, token: hex.EncodeToString(random), host: listener.Addr().String()}
	p.touched.Store(time.Now().Unix())
	server := &http.Server{Handler: p, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 60 * time.Second, IdleTimeout: 30 * time.Second, MaxHeaderBytes: 8192}
	stopped := make(chan struct{})
	var stopOnce sync.Once
	p.stop = func() {
		stopOnce.Do(func() {
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				defer cancel()
				_ = server.Shutdown(ctx)
				close(stopped)
			}()
		})
	}
	done := make(chan struct{})
	defer close(done)
	go func() {
		timer := time.NewTicker(time.Minute)
		defer timer.Stop()
		for {
			select {
			case <-done:
				return
			case <-timer.C:
				if time.Now().Unix()-p.touched.Load() >= 20*60 {
					p.stop()
					return
				}
			}
		}
	}()
	if _, err = fmt.Fprintf(out, "http://%s/#%s\n", p.host, p.token); err != nil {
		return err
	}
	err = server.Serve(listener)
	if err == http.ErrServerClosed {
		<-stopped // Finish in-flight responses before the CLI exits.
		return nil
	}
	return err
}

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
		name := map[string]string{"/": "index.html", "/app.js": "app.js", "/style.css": "style.css"}[r.URL.Path]
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
			http.Error(w, "asset", 500)
			return
		}
		types := map[string]string{"index.html": "text/html; charset=utf-8", "app.js": "text/javascript; charset=utf-8", "style.css": "text/css; charset=utf-8"}
		w.Header().Set("Content-Type", types[name])
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
		files, err := filepath.Glob(filepath.Join(c.DataDir, "migrations", "*", "report.json"))
		if err != nil {
			return nil, err
		}
		reports := []any{}
		for _, file := range files {
			var report MigrationReport
			latest := filepath.Join(filepath.Dir(file), "latest-report.json")
			historical := false
			if err := readJSON(latest, &report); os.IsNotExist(err) {
				historical = true
				if err = readJSON(file, &report); err != nil {
					return nil, err
				}
			} else if err != nil {
				return nil, err
			}
			reports = append(reports, map[string]any{"report": report, "historical": historical})
		}
		return reports, nil
	case "POST /api/settings":
		var settings ManagedSettings
		if err = panelBody(w, r, &settings); err != nil {
			return nil, err
		}
		if err = validateManaged(settings); err != nil {
			return nil, err
		}
		if err = saveManaged(c, &settings); err != nil {
			return nil, err
		}
		return map[string]bool{"ok": true}, nil
	case "POST /api/settings/reset":
		if err = saveManaged(c, nil); err != nil {
			return nil, err
		}
		return map[string]bool{"ok": true}, nil
	case "POST /api/record":
		var req struct {
			ID      string `json:"id"`
			Action  string `json:"action"`
			Enabled bool   `json:"enabled"`
			Editor  string `json:"editor"`
		}
		if err = panelBody(w, r, &req); err != nil {
			return nil, err
		}
		history, _ := loadHistory(c)
		idx, _ := loadIndex(c)
		found := false
		for _, project := range merge(history, idx.Projects) {
			if project.ID == req.ID {
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("记录已不在当前来源中，请刷新列表")
		}
		err = changePreferences(c, func(prefs *Preferences) error {
			switch req.Action {
			case "hidden":
				if req.Enabled {
					prefs.Hidden[req.ID] = true
				} else {
					delete(prefs.Hidden, req.ID)
				}
			case "pinned":
				if req.Enabled {
					prefs.Pinned[req.ID] = true
				} else {
					delete(prefs.Pinned, req.ID)
				}
			case "editor":
				if req.Editor == "" {
					delete(prefs.Editors, req.ID)
				} else if validEditor(req.Editor) {
					prefs.Editors[req.ID] = req.Editor
				} else {
					return fmt.Errorf("无效 IDE")
				}
			default:
				return fmt.Errorf("无效操作")
			}
			return nil
		})
		return map[string]bool{"ok": err == nil}, err
	case "POST /api/rebuild":
		if !p.busy.TryLock() {
			return nil, fmt.Errorf("正在扫描或迁移，请稍后重试")
		}
		defer p.busy.Unlock()
		return BuildIndex(c, false)
	case "POST /api/migrate":
		var req struct {
			Source string `json:"source"`
			Apply  bool   `json:"apply"`
		}
		if err = panelBody(w, r, &req); err != nil {
			return nil, err
		}
		if req.Source == "" {
			return nil, fmt.Errorf("请填写旧版数据目录或缓存文件路径")
		}
		if !p.busy.TryLock() {
			return nil, fmt.Errorf("正在扫描或迁移，请稍后重试")
		}
		defer p.busy.Unlock()
		return ImportLegacy(c, expand(req.Source, c.Home), req.Apply)
	case "POST /api/close":
		p.stop()
		return map[string]bool{"ok": true}, nil
	default:
		return nil, fmt.Errorf("未知接口或请求方法")
	}
}

func saveManaged(c Config, settings *ManagedSettings) error {
	lock, err := lockFile(filepath.Join(c.DataDir, "config.lock"), false)
	if err != nil {
		return err
	}
	defer unlock(lock)
	name := filepath.Join(c.DataDir, "config.json")
	raw := map[string]json.RawMessage{}
	if err = readJSON(name, &raw); err != nil && !os.IsNotExist(err) {
		return err
	}
	if raw == nil {
		return fmt.Errorf("config.json 必须是 JSON 对象")
	}
	if settings == nil {
		delete(raw, "managed")
	} else {
		raw["managed"], err = json.Marshal(settings)
		if err != nil {
			return err
		}
	}
	return atomicJSON(name, raw)
}

type panelRecord struct {
	Project
	Hidden bool   `json:"hidden"`
	Pinned bool   `json:"pinned"`
	Editor string `json:"editor"`
	Status string `json:"status"`
	Branch string `json:"branch,omitempty"`
}

func panelRecords(c Config, r *http.Request) (any, error) {
	prefs, err := loadPreferences(c)
	if err != nil {
		return nil, err
	}
	history, warnings := loadHistory(c)
	idx, stale := loadIndex(c)
	projects := merge(history, idx.Projects)
	query := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	filter := r.URL.Query().Get("filter")
	filtered := make([]Project, 0, len(projects))
	hidden, pinned := 0, 0
	for _, item := range projects {
		if prefs.Hidden[item.ID] {
			hidden++
		}
		if prefs.Pinned[item.ID] {
			pinned++
		}
		if filter == "hidden" && !prefs.Hidden[item.ID] || filter == "pinned" && !prefs.Pinned[item.ID] || filter == "remote" && item.Kind != "remote" || filter == "local" && item.Kind != "folder" {
			continue
		}
		if query != "" && !strings.Contains(strings.ToLower(item.Name+" "+item.Path+" "+item.Authority), query) {
			continue
		}
		filtered = append(filtered, item)
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 0 {
		page = 0
	}
	pages := max(1, (len(filtered)+49)/50)
	if page >= pages {
		page = pages - 1
	}
	start := page * 50
	end := min(start+50, len(filtered))
	statuses := visibleBranches(filtered[start:end])
	rows := make([]panelRecord, 0, end-start)
	for _, item := range filtered[start:end] {
		row := panelRecord{Project: item, Hidden: prefs.Hidden[item.ID], Pinned: prefs.Pinned[item.ID], Editor: prefs.Editors[item.ID], Status: "可用"}
		if item.Kind == "remote" {
			row.Status = "远程 · 未探测"
		} else {
			s := statuses[item.ID]
			row.Branch = s.branch
			if s.timedOut {
				row.Status = "读取超时"
			} else if s.missing {
				row.Status = s.reason
			}
		}
		rows = append(rows, row)
	}
	return map[string]any{"items": rows, "total": len(filtered), "all": len(projects), "hidden": hidden, "pinned": pinned, "page": page, "pages": pages, "warnings": warnings, "index_stale": stale, "data_dir": c.DataDir}, nil
}
