package workflow

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

func encode(w io.Writer, v any) error {
	e := json.NewEncoder(w)
	e.SetEscapeHTML(false)
	return e.Encode(v)
}

func errorFeedback(err error) Feedback {
	return Feedback{Items: []Item{{Title: "VSC 需要处理一个问题", Subtitle: err.Error(), Valid: false}}}
}

func Run(args []string, out, stderr io.Writer) int {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" {
		fmt.Fprintln(out, "vsc query [text] | index | open --target URI [--editor NAME] [--dry-run] | editors --target URI | hide/unhide/pin --target URI | migrate-legacy --source PATH [--data-dir PATH] [--apply] | manage [--serve] | restore-all | doctor | config-init | version")
		return 0
	}
	if args[0] == "version" || args[0] == "--version" {
		fmt.Fprintln(out, Version)
		return 0
	}
	c, e := LoadConfig()
	if e != nil {
		if args[0] == "query" {
			_ = encode(out, errorFeedback(e))
		}
		fmt.Fprintln(stderr, e)
		return 1
	}
	command := args[0]
	args = args[1:]
	fail := func(e error) int { fmt.Fprintln(stderr, e); return 1 }
	switch command {
	case "manage":
		if len(args) == 1 && args[0] == "--serve" {
			if err := servePanel(c, out); err != nil {
				return fail(err)
			}
		} else if len(args) == 0 {
			if err := launchPanel(c); err != nil {
				return fail(err)
			}
			fmt.Fprintln(out, "记录管理面板已打开")
		} else {
			return fail(fmt.Errorf("用法：vsc manage [--serve]"))
		}

	case "migrate-legacy":
		flags := flag.NewFlagSet(command, flag.ContinueOnError)
		flags.SetOutput(stderr)
		source := flags.String("source", os.Getenv("VSC_LEGACY_CACHE"), "旧工作流目录或 .records.cache.json")
		destination := flags.String("data-dir", "", "新版工作流用户数据目录")
		apply := flags.Bool("apply", false, "实际迁移；默认仅预览")
		dry := flags.Bool("dry-run", false, "仅预览")
		if flags.Parse(args) != nil {
			return 2
		}
		if *source == "" || flags.NArg() != 0 || *apply && *dry {
			return fail(fmt.Errorf("需要 --source；--apply 与 --dry-run 不能同时使用"))
		}
		if *destination != "" {
			c, e = loadConfigAt(*destination)
			if e != nil {
				return fail(e)
			}
		}
		report, err := ImportLegacy(c, expand(*source, c.Home), *apply)
		if err != nil {
			return fail(err)
		}
		if err = encode(out, report); err != nil {
			return fail(err)
		}
	case "query":
		query := strings.Join(args, " ")
		feedback, e := Query(c, query)
		if e != nil {
			_ = encode(out, errorFeedback(e))
			return fail(e)
		}
		if e = encode(out, feedback); e != nil {
			return fail(e)
		}
	case "index", "rebuild":
		flags := flag.NewFlagSet(command, flag.ContinueOnError)
		flags.SetOutput(stderr)
		background := flags.Bool("background", false, "")
		drop := flags.Bool("drop-all", false, "")
		if flags.Parse(args) != nil {
			return 2
		}
		if *drop {
			if e = changePreferences(c, func(p *Preferences) error { p.Hidden = map[string]bool{}; return nil }); e != nil {
				return fail(e)
			}
		}
		index, e := BuildIndex(c, *background)
		if e != nil {
			if *background && errors.Is(e, syscall.EWOULDBLOCK) {
				return 0
			}
			return fail(e)
		}
		if command == "rebuild" {
			fmt.Fprintf(out, "索引已更新：%d 个 Git 仓库\n", len(index.Projects))
		} else {
			_ = encode(out, index)
		}
	case "open", "editors", "hide", "delete", "unhide", "pin", "prefer":
		flags := flag.NewFlagSet(command, flag.ContinueOnError)
		flags.SetOutput(stderr)
		target := flags.String("target", "", "")
		legacy := flags.String("record", "", "")
		editor := flags.String("editor", "", "")
		requestText := flags.String("request", "", "")
		dry := flags.Bool("dry-run", false, "")
		newWindow := flags.Bool("new-window", false, "")
		remember := flags.Bool("remember", false, "")
		query := flags.String("query", "", "")
		if flags.Parse(args) != nil {
			return 2
		}
		if *target == "" {
			*target = *legacy
		}
		request := OpenRequest{Target: *target, Editor: *editor, NewWindow: *newWindow}
		if *requestText != "" {
			if e = json.Unmarshal([]byte(*requestText), &request); e != nil {
				return fail(e)
			}
		}
		if request.Target == "" {
			return fail(fmt.Errorf("--target or --request is required"))
		}
		if command == "editors" {
			_ = encode(out, EditorChoices(c, request.Target, *query))
			return 0
		}
		p, e := project(request.Target, "")
		if e != nil {
			return fail(e)
		}
		if command == "open" {
			if request.Editor == "" {
				prefs, err := loadPreferences(c)
				if err != nil {
					return fail(err)
				}
				request.Editor = prefs.Editors[p.ID]
			}
			plan, e := PlanOpen(c, request)
			if e != nil {
				return fail(e)
			}
			if *dry {
				_ = encode(out, plan)
				return 0
			}
			if e = ExecuteOpen(plan); e != nil {
				return fail(e)
			}
			if *remember {
				if e = changePreferences(c, func(prefs *Preferences) error { prefs.Editors[p.ID] = plan.Editor; return nil }); e != nil {
					return fail(e)
				}
			}
			return 0
		}
		e = changePreferences(c, func(prefs *Preferences) error {
			switch command {
			case "hide", "delete":
				prefs.Hidden[p.ID] = true
			case "unhide":
				delete(prefs.Hidden, p.ID)
			case "pin":
				if prefs.Pinned[p.ID] {
					delete(prefs.Pinned, p.ID)
				} else {
					prefs.Pinned[p.ID] = true
				}
			case "prefer":
				if !validEditor(request.Editor) {
					return fmt.Errorf("invalid editor")
				}
				prefs.Editors[p.ID] = request.Editor
			}
			return nil
		})
		if e != nil {
			return fail(e)
		}
		fmt.Fprintln(out, "项目偏好已更新")
	case "restore-all":
		if e = changePreferences(c, func(p *Preferences) error { p.Hidden = map[string]bool{}; return nil }); e != nil {
			return fail(e)
		}
		fmt.Fprintln(out, "已恢复所有隐藏项目")
	case "doctor":
		editors := map[string]string{}
		for _, name := range []string{"vscode", "zed", "trae", "cursor"} {
			exe, _, err := resolveEditor(c, name)
			if err != nil {
				editors[name] = "unavailable"
			} else {
				editors[name] = exe
			}
		}
		sources := map[string]bool{}
		for _, name := range historyPaths(c) {
			_, err := os.Stat(name)
			sources[name] = err == nil
		}
		idx, stale := loadIndex(c)
		_ = encode(out, map[string]any{"version": Version, "platform": c.OS, "roots": c.Roots, "data_dir": c.DataDir, "history_sources": sources, "editors": editors, "index_count": len(idx.Projects), "index_stale": stale})
	case "config-init":
		name := filepath.Join(c.DataDir, "config.json")
		if _, e = os.Stat(name); os.IsNotExist(e) {
			if e = atomicJSON(name, c); e != nil {
				return fail(e)
			}
		} else if e != nil {
			return fail(e)
		}
		fmt.Fprintln(out, name)
	default:
		return fail(fmt.Errorf("unknown command %q", command))
	}
	return 0
}
