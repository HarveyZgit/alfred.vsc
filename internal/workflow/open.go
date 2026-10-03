package workflow

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type OpenRequest struct {
	Target    string `json:"target"`
	Editor    string `json:"editor,omitempty"`
	NewWindow bool   `json:"new_window,omitempty"`
}
type OpenPlan struct {
	Executable string   `json:"executable"`
	Args       []string `json:"args"`
	Editor     string   `json:"editor"`
}

var appNames = map[string]string{"vscode": "Visual Studio Code", "zed": "Zed", "trae": "Trae", "cursor": "Cursor"}
var cliNames = map[string]string{"vscode": "code", "zed": "zed", "trae": "trae", "cursor": "cursor"}

func executable(name string) bool {
	info, e := os.Stat(name)
	return e == nil && !info.IsDir() && info.Mode()&0111 != 0
}
func resolveEditor(c Config, editor string) (string, bool, error) {
	if !validEditor(editor) {
		return "", false, fmt.Errorf("unknown editor %q", editor)
	}
	if override := c.Editors[editor]; override != "" {
		if name, e := exec.LookPath(override); e == nil {
			return name, false, nil
		}
		return "", false, fmt.Errorf("%s executable unavailable: %s", editor, override)
	}
	if name, e := exec.LookPath(cliNames[editor]); e == nil {
		return name, false, nil
	}
	if c.OS == "darwin" {
		for _, base := range []string{filepath.Join(c.Home, "Applications"), "/Applications"} {
			app := filepath.Join(base, appNames[editor]+".app")
			candidate := filepath.Join(app, "Contents/Resources/app/bin", cliNames[editor])
			if editor == "zed" {
				candidate = filepath.Join(app, "Contents/MacOS/cli")
			}
			if executable(candidate) {
				return candidate, false, nil
			}
			if info, e := os.Stat(app); e == nil && info.IsDir() {
				return app, true, nil
			}
		}
	}
	return "", false, fmt.Errorf("未找到 %s；请安装应用或设置 VSC_EDITOR_%s", appNames[editor], strings.ToUpper(editor))
}
func PlanOpen(c Config, request OpenRequest) (OpenPlan, error) {
	p, e := project(request.Target, "")
	if e != nil {
		return OpenPlan{}, e
	}
	editor := request.Editor
	if editor == "" {
		editor = c.Editor
	}
	name, isApp, e := resolveEditor(c, editor)
	if e != nil {
		return OpenPlan{}, e
	}
	plan := OpenPlan{Executable: name, Editor: editor, Args: []string{}}
	if p.Kind == "folder" {
		info, e := os.Stat(p.Path)
		if e != nil || !info.IsDir() {
			return plan, fmt.Errorf("目录不存在或不可访问: %s", p.Path)
		}
		if isApp {
			plan.Executable = "/usr/bin/open"
			plan.Args = []string{"-a", name, p.Path}
			if request.NewWindow {
				if editor == "trae" {
					return plan, fmt.Errorf("Trae 新窗口需要配置可用 CLI")
				}
				flag := "--new-window"
				if editor == "zed" {
					flag = "-n"
				}
				plan.Args = append(plan.Args, "--args", flag)
			}
			return plan, nil
		}
		if request.NewWindow {
			flag := "--new-window"
			if editor == "zed" {
				flag = "--new"
			}
			plan.Args = append(plan.Args, flag)
		}
		plan.Args = append(plan.Args, "--", p.Path)
		return plan, nil
	}
	// Remote directories never use local filesystem operations or branch probes.
	if isApp {
		return plan, fmt.Errorf("远程打开需要 %s CLI；请设置对应 VSC_EDITOR_*", editor)
	}
	if editor == "zed" {
		if !strings.HasPrefix(p.Authority, "ssh-remote+") {
			return plan, fmt.Errorf("Zed 只能转换明确的 SSH 目录；容器请使用 VS Code")
		}
		host := strings.TrimPrefix(p.Authority, "ssh-remote+")
		if host == "" || strings.ContainsAny(host, "/\\?# \t\n") || strings.HasPrefix(host, "{") {
			return plan, fmt.Errorf("该 SSH authority 无法可靠转换为 Zed 地址，请使用 VS Code")
		}
		// VS Code also stores encoded JSON SSH authorities. Never treat those as a host.
		if len(host) > 32 && !strings.Contains(host, ".") {
			allHex := true
			for _, r := range host {
				if !strings.ContainsRune("0123456789abcdefABCDEF", r) {
					allHex = false
					break
				}
			}
			if allHex {
				return plan, fmt.Errorf("编码的 SSH authority 请使用 VS Code")
			}
		}
		uri, e := url.Parse(p.URI)
		if e != nil {
			return plan, e
		}
		uri.Scheme = "ssh"
		uri.Host = host
		if request.NewWindow {
			plan.Args = append(plan.Args, "--new")
		}
		plan.Args = append(plan.Args, uri.String())
		return plan, nil
	}
	if editor != "vscode" && !strings.HasPrefix(p.Authority, "ssh-remote+") {
		return plan, fmt.Errorf("容器等远程目录优先使用来源 IDE VS Code")
	}
	if request.NewWindow {
		plan.Args = append(plan.Args, "--new-window")
	}
	plan.Args = append(plan.Args, "--folder-uri", p.URI)
	return plan, nil
}
func ExecuteOpen(plan OpenPlan) error {
	cmd := exec.Command(plan.Executable, plan.Args...)
	output, e := cmd.CombinedOutput()
	if e != nil {
		return fmt.Errorf("启动 %s 失败: %w (%s)", plan.Editor, e, strings.TrimSpace(string(output)))
	}
	return nil
}
func EditorChoices(c Config, target, query string) Feedback {
	items := []Item{}
	for _, editor := range []string{"vscode", "zed", "trae", "cursor"} {
		if query != "" && !strings.Contains(strings.ToLower(appNames[editor]), strings.ToLower(query)) {
			continue
		}
		request := OpenRequest{Target: target, Editor: editor}
		_, e := PlanOpen(c, request)
		subtitle := "用 " + appNames[editor] + " 打开此目录"
		if e != nil {
			subtitle = e.Error()
		}
		b, _ := json.Marshal(request)
		items = append(items, Item{Title: appNames[editor], Subtitle: subtitle, Arg: string(b), Valid: e == nil, Mods: map[string]Modifier{"cmd": {Subtitle: "记住此项目的默认 IDE", Valid: e == nil}}, Variables: map[string]string{"vsc_target": target, "vsc_editor": editor}})
	}
	return Feedback{Items: items}
}
