package workflow

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func panelFixture(t *testing.T) (Config, *panel) {
	t.Helper()
	c := fixture(t)
	for _, key := range []string{"VSC_DEV_MODE", "VSC_DIRECTORIES", "VSC_IDE_PATH", "VSC_DB_PATH", "VSC_DEFAULT_EDITOR", "VSC_EDITOR_VSCODE", "VSC_EDITOR_ZED", "VSC_EDITOR_TRAE", "VSC_EDITOR_CURSOR", "VSC_OPEN_DEFAULT", "VSC_LIMIT", "VSC_SCAN_DEPTH", "VSC_REFRESH_SECONDS"} {
		t.Setenv(key, "")
	}
	t.Setenv("VSC_DATA_DIR", c.DataDir)
	t.Setenv("VSC_CACHE_DIR", c.CacheDir)
	t.Setenv("VSC_NO_BACKGROUND", "1")
	if err := atomicJSON(filepath.Join(c.DataDir, "config.json"), c); err != nil {
		t.Fatal(err)
	}
	return c, &panel{dataDir: c.DataDir, token: "test-token", host: "127.0.0.1:12345", stop: func() {}}
}

func panelRequest(t *testing.T, p *panel, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	data, _ := json.Marshal(body)
	r := httptest.NewRequest(method, "http://"+p.host+path, strings.NewReader(string(data)))
	r.Header.Set("Authorization", "Bearer "+p.token)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	p.ServeHTTP(w, r)
	return w
}

func TestPanelProtectsPreferencesAndServesEmbeddedUI(t *testing.T) {
	c, p := panelFixture(t)
	for _, test := range []struct{ host, origin, auth string }{{p.host, "", ""}, {"evil.test:12345", "", "Bearer " + p.token}, {p.host, "http://evil.test", "Bearer " + p.token}} {
		r := httptest.NewRequest("POST", "http://"+test.host+"/api/settings", strings.NewReader(`{"roots":[],"editor":"zed","editors":{}}`))
		r.Header.Set("Authorization", test.auth)
		r.Header.Set("Origin", test.origin)
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		p.ServeHTTP(w, r)
		if w.Code != 401 && w.Code != 403 {
			t.Fatalf("unprotected request: %d", w.Code)
		}
	}
	cfg, err := loadConfigAt(c.DataDir)
	if err != nil || cfg.Managed != nil {
		t.Fatalf("unauthorized mutation: %+v %v", cfg, err)
	}
	for _, path := range append([]string{"/"}, panelAssetPaths(t, p)...) {
		w := panelRequest(t, p, "GET", path, nil)
		if w.Code != 200 || w.Body.Len() < 100 || !strings.Contains(w.Header().Get("Content-Security-Policy"), "frame-ancestors 'none'") {
			t.Fatalf("asset %s: %d", path, w.Code)
		}
	}
	if w := panelRequest(t, p, "GET", "/../config.json", nil); w.Code != 404 {
		t.Fatal("served arbitrary path")
	}
}

func TestPanelSettingsOverrideAlfredAndPreserveAdvancedConfig(t *testing.T) {
	c, p := panelFixture(t)
	t.Setenv("VSC_DIRECTORIES", filepath.Join(c.Home, "old-root"))
	t.Setenv("VSC_DEFAULT_EDITOR", "vscode")
	t.Setenv("VSC_EDITOR_ZED", "/old/zed")
	raw := map[string]any{"database": "/custom/state.vscdb", "unknown_future_option": true}
	if err := atomicJSON(filepath.Join(c.DataDir, "config.json"), raw); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(c.Home, "new-root")
	project := repo(t, filepath.Join(root, "panel-project"), "main")
	settings := ManagedSettings{Roots: []string{root}, Editor: "zed", Editors: map[string]string{"zed": "/new/zed"}}
	if w := panelRequest(t, p, "POST", "/api/settings", settings); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	cfg, err := loadConfigAt(c.DataDir)
	if err != nil || len(cfg.Roots) != 1 || cfg.Roots[0] != root || cfg.Editor != "zed" || cfg.Editors["zed"] != "/new/zed" {
		t.Fatalf("overrides ignored: %+v %v", cfg, err)
	}
	if err = readJSON(filepath.Join(c.DataDir, "config.json"), &raw); err != nil || raw["unknown_future_option"] != true || cfg.Database != "/custom/state.vscdb" {
		t.Fatal("advanced config lost")
	}
	if _, err = BuildIndex(cfg, false); err != nil {
		t.Fatal(err)
	}
	f, err := Query(cfg, "panel-project")
	if err != nil || len(targets(f)) != 1 || targets(f)[0] != project.URI {
		t.Fatalf("query not using panel roots: %+v %v", f, err)
	}
	settings.Roots = []string{"relative-path"}
	if w := panelRequest(t, p, "POST", "/api/settings", settings); w.Code != 400 {
		t.Fatal("invalid settings saved")
	}
	if w := panelRequest(t, p, "POST", "/api/settings/reset", map[string]any{}); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	cfg, err = loadConfigAt(c.DataDir)
	if err != nil || cfg.Managed != nil || cfg.Editor != "vscode" || cfg.Editors["zed"] != "/old/zed" {
		t.Fatalf("reset failed: %+v %v", cfg, err)
	}
}

func TestPanelHiddenRecordsRestoreAndFreshStatus(t *testing.T) {
	c, p := panelFixture(t)
	db := database(t, c)
	local := repo(t, filepath.Join(c.Home, "project"), "main")
	missing, _ := localProject(filepath.Join(c.Home, "missing"))
	file := filepath.Join(c.Home, "file")
	write(t, file, "not a directory")
	notdir, _ := localProject(file)
	remote, _ := project("vscode-remote://ssh-remote+host"+local.Path, "")
	setHistory(t, db, "history.recentlyOpenedPathsList", []any{map[string]string{"folderUri": local.URI}, map[string]string{"folderUri": missing.URI}, map[string]string{"folderUri": notdir.URI}, map[string]string{"folderUri": remote.URI}})
	if w := panelRequest(t, p, "POST", "/api/record", map[string]any{"id": local.ID, "action": "hidden", "enabled": true}); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	cfg, _ := loadConfigAt(c.DataDir)
	f, err := Query(cfg, "project")
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range targets(f) {
		if target == local.URI {
			t.Fatal("hidden project remains in query")
		}
	}
	w := panelRequest(t, p, "GET", "/api/records?filter=hidden", nil)
	var rows struct {
		Items []panelRecord `json:"items"`
		Total int           `json:"total"`
	}
	if err = json.Unmarshal(w.Body.Bytes(), &rows); err != nil || rows.Total != 1 || !rows.Items[0].Hidden {
		t.Fatalf("hidden unavailable: %s", w.Body.String())
	}
	write(t, filepath.Join(local.Path, ".git/HEAD"), "ref: refs/heads/changed\n")
	w = panelRequest(t, p, "GET", "/api/records", nil)
	_ = json.Unmarshal(w.Body.Bytes(), &rows)
	for _, r := range rows.Items {
		switch r.ID {
		case local.ID:
			if r.Branch != "changed" {
				t.Fatal("stale branch")
			}
		case missing.ID:
			if r.Status != "目录不存在" {
				t.Fatal(r.Status)
			}
		case notdir.ID:
			if r.Status != "路径不是目录" {
				t.Fatal(r.Status)
			}
		case remote.ID:
			if r.Branch != "" || r.Status != "远程 · 未探测" {
				t.Fatal("remote probed local path")
			}
		}
	}
	if w = panelRequest(t, p, "POST", "/api/record", map[string]any{"id": local.ID, "action": "hidden", "enabled": false}); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	f, err = Query(cfg, "project")
	found := false
	for _, target := range targets(f) {
		found = found || target == local.URI
	}
	if err != nil || !found {
		t.Fatal("restored project missing")
	}
	if w = panelRequest(t, p, "POST", "/api/record", map[string]any{"id": "outside", "action": "hidden", "enabled": true}); w.Code != 400 {
		t.Fatal("accepted unknown source")
	}
}

func TestPanelMigrationShowsLatestIncrementalReport(t *testing.T) {
	c, p := panelFixture(t)
	a := repo(t, filepath.Join(c.Home, "root/a"), "main")
	b := repo(t, filepath.Join(c.Home, "other/b"), "main")
	source := legacyFixture(t, c, []any{map[string]string{"path": a.Path, "type": "folder"}, map[string]string{"path": b.Path, "type": "folder"}}, nil)
	c.Roots = []string{filepath.Dir(a.Path)}
	first, err := ImportLegacy(c, source, true)
	if err != nil {
		t.Fatal(err)
	}
	c.Roots = append(c.Roots, filepath.Dir(b.Path))
	second, err := ImportLegacy(c, source, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Unmigrated) != 1 || len(second.Unmigrated) != 0 {
		t.Fatal("bad fixture")
	}
	w := panelRequest(t, p, "GET", "/api/migrations", nil)
	var reports []struct {
		Report     MigrationReport `json:"report"`
		Historical bool            `json:"historical"`
	}
	if err = json.Unmarshal(w.Body.Bytes(), &reports); err != nil || len(reports) != 1 || reports[0].Historical || len(reports[0].Report.Unmigrated) != 0 || !reports[0].Report.Applied {
		t.Fatalf("stale report %s", w.Body.String())
	}
	if err = os.Remove(filepath.Join(second.Backup, "latest-report.json")); err != nil {
		t.Fatal(err)
	}
	w = panelRequest(t, p, "GET", "/api/migrations", nil)
	_ = json.Unmarshal(w.Body.Bytes(), &reports)
	if !reports[0].Historical || len(reports[0].Report.Unmigrated) != 1 {
		t.Fatal("old report not marked historical")
	}
	// Exercise the real HTTP transport (not only the recorder).
	server := httptest.NewUnstartedServer(p)
	parsed, _ := url.Parse("http://" + server.Listener.Addr().String())
	p.host = parsed.Host
	server.Start()
	defer server.Close()
	req, _ := http.NewRequest("GET", server.URL+"/api/migrations", nil)
	req.Header.Set("Authorization", "Bearer "+p.token)
	resp, err := server.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != 200 {
		t.Fatal(resp.Status)
	}
}

func TestPanelStaticRoutesRestrictPathsAndTypes(t *testing.T) {
	_, p := panelFixture(t)
	for _, assetPath := range panelAssetPaths(t, p) {
		name := assetPath
		w := panelRequest(t, p, http.MethodGet, assetPath, nil)
		wantType := "text/javascript; charset=utf-8"
		if strings.HasSuffix(name, ".css") {
			wantType = "text/css; charset=utf-8"
		}
		if w.Code != http.StatusOK || w.Header().Get("Content-Type") != wantType {
			t.Fatalf("asset %s: status=%d type=%q", name, w.Code, w.Header().Get("Content-Type"))
		}
	}
	for _, path := range []string{"/missing.js", "/assets/", "/.hidden", "/assets/.hidden", "/nested/app.js", "/../app.js", "/%2e%2e/app.js", "/panel/app.js", "/config.json"} {
		if w := panelRequest(t, p, http.MethodGet, path, nil); w.Code != http.StatusNotFound {
			t.Fatalf("unexpected static route %s: %d", path, w.Code)
		}
	}
	if w := panelRequest(t, p, http.MethodPost, panelAssetPaths(t, p)[0], nil); w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("static mutation: %d", w.Code)
	}
}

func panelAssetPaths(t *testing.T, p *panel) []string {
	t.Helper()
	w := panelRequest(t, p, http.MethodGet, "/", nil)
	matches := regexp.MustCompile(`(?:src|href)=["']([^"']+\.(?:js|css))["']`).FindAllStringSubmatch(w.Body.String(), -1)
	paths := make([]string, 0, len(matches))
	var hasJS, hasCSS bool
	for _, match := range matches {
		asset := match[1]
		if !strings.HasPrefix(asset, "/") || strings.HasPrefix(asset, "//") {
			t.Fatalf("nonlocal panel asset %q", asset)
		}
		paths = append(paths, asset)
		hasJS = hasJS || strings.HasSuffix(asset, ".js")
		hasCSS = hasCSS || strings.HasSuffix(asset, ".css")
	}
	if !hasJS || !hasCSS {
		t.Fatalf("panel HTML lacks built JS/CSS: %s", w.Body.String())
	}
	return paths
}
