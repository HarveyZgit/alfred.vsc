package workflow

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

type MigrationItem struct {
	Position   int    `json:"position"`
	Original   string `json:"original"`
	Target     string `json:"target,omitempty"`
	Status     string `json:"status"`
	Hidden     bool   `json:"hidden,omitempty"`
	Resolution string `json:"how_to_migrate,omitempty"`
}

type MigrationReport struct {
	Version        int             `json:"version"`
	Source         string          `json:"source"`
	Destination    string          `json:"destination"`
	Fingerprint    string          `json:"fingerprint"`
	Backup         string          `json:"backup"`
	Applied        bool            `json:"applied"`
	AlreadyApplied bool            `json:"already_applied"`
	Records        int             `json:"records"`
	Hidden         int             `json:"hidden_records"`
	OrderImported  int             `json:"directory_order_imported"`
	HiddenImported int             `json:"hidden_imported"`
	Items          []MigrationItem `json:"items"`
	Unmigrated     []MigrationItem `json:"unmigrated"`
	Notes          []string        `json:"notes"`
}

// ImportLegacy never modifies the source, source IDE, or destination config.
// A whole backup directory is published before the single atomic preferences commit.
func ImportLegacy(c Config, source string, apply bool) (MigrationReport, error) {
	report := MigrationReport{Version: 1, Destination: c.DataDir, Items: []MigrationItem{}, Unmigrated: []MigrationItem{}, Notes: []string{
		"原记录、最近使用顺序、未知字段及用户配置原样备份；不修改旧版文件或 VS Code 数据库。",
		"普通搜索仍取 VS Code 目录与配置 Git 仓库的并集；范围外记录列入 unmigrated 清单；先加入新版来源，再重新运行可补迁移。",
		"同分项目保留已确认目录的旧使用顺序；VS Code 历史目录或顺序变化后自动恢复以 VS Code 排序。",
		"旧版无法区分的远程文件/目录仅归档；不探测远程，旧分支缓存不用于显示。",
	}}
	name, e := legacySource(source)
	if e != nil {
		return report, e
	}
	report.Source = name
	payload, e := migrationRead(name)
	if e != nil {
		return report, e
	}
	var shape map[string]json.RawMessage
	if e = json.Unmarshal(payload, &shape); e != nil {
		return report, e
	}
	if shape == nil || shape["records"] == nil && shape["trash"] == nil {
		return report, fmt.Errorf("不支持的旧版缓存格式：需要 records 或 trash")
	}
	var old struct {
		Records []legacyRecord          `json:"records"`
		Trash   map[string]legacyRecord `json:"trash"`
	}
	if e = json.Unmarshal(payload, &old); e != nil {
		return report, e
	}
	report.Records = len(old.Records)
	report.Hidden = len(old.Trash)
	userConfig, e := migrationRead(filepath.Join(filepath.Dir(name), ".user.config.json"))
	if os.IsNotExist(e) {
		userConfig = nil
	} else if e != nil {
		return report, e
	}
	sum := sha256.New()
	sum.Write(payload)
	sum.Write([]byte{0})
	sum.Write(userConfig)
	report.Fingerprint = fmt.Sprintf("%x", sum.Sum(nil))
	report.Backup = filepath.Join(c.DataDir, "migrations", report.Fingerprint)
	history, historyWarnings := migrationHistory(c)
	report.Notes = append(report.Notes, historyWarnings...)
	known := map[string]bool{}
	for _, p := range history {
		known[p.ID] = true
	}
	indexed, warnings := scan(c)
	report.Notes = append(report.Notes, warnings...)
	for _, p := range indexed {
		known[p.ID] = true
	}
	order := []string{}
	hidden := map[string]bool{}
	seen := map[string]bool{}
	classify := func(r legacyRecord, position int, trash bool) MigrationItem {
		item := MigrationItem{Position: position, Original: r.Path, Status: "invalid_path", Hidden: trash, Resolution: "修正旧记录中的路径，先用 VS Code 打开对应目录，然后重新运行迁移。"}
		p, err := legacyProject(r)
		// Older trash records lost the source marker. Resolve URI escaping against
		// actual V3 sources, rather than guessing whether an old '%' was encoded.
		if err != nil || !known[p.ID] {
			alternatives := []legacyRecord{r}
			decoded := r
			decoded.Extra.From = "getRecordsFromVscodeDB"
			alternatives = append(alternatives, decoded)
			encoded := r
			encoded.Extra.From = "getRecordsFromVscodeMenu"
			alternatives = append(alternatives, encoded)
			for _, candidate := range alternatives {
				if q, e := legacyProject(candidate); e == nil && known[q.ID] {
					p = q
					err = nil
					break
				}
			}
		}
		if err == nil {
			item.Target = p.URI
			if known[p.ID] {
				item.Status = "eligible"
				item.Resolution = ""
				if trash {
					hidden[p.ID] = true
				} else if !seen[p.ID] {
					order = append(order, p.ID)
					seen[p.ID] = true
				}
			} else if !legacyDirectory(r, p, known) {
				item.Status = "file_or_unconfirmed_directory"
				item.Resolution = "新版不收录文件；若实际是目录，先用 VS Code 打开该目录后重新运行。远程仅凭旧记录无法确认文件/目录，不会主动连接。"
			} else {
				item.Status = "outside_current_sources"
				item.Resolution = "先用 VS Code 打开该目录；若是本地 Git 仓库，也可把其上级目录加入管理面板的 Git 扫描根目录（未使用面板时也可配置 VSC_DIRECTORIES 或 config.json 的 roots）。之后重新预览并执行迁移。"
			}
		}
		if item.Status != "eligible" {
			report.Unmigrated = append(report.Unmigrated, item)
		}
		return item
	}
	for i, r := range old.Records {
		report.Items = append(report.Items, classify(r, i, false))
	}
	keys := []string{}
	for key := range old.Trash {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for i, key := range keys {
		report.Items = append(report.Items, classify(old.Trash[key], i, true))
	}
	report.OrderImported = len(order)
	report.HiddenImported = len(hidden)
	prefs, e := loadPreferences(c)
	if e != nil {
		return report, e
	}
	pending := func(p Preferences) bool {
		if !p.LegacyImports[report.Fingerprint] {
			return true
		}
		for _, id := range order {
			if !p.LegacyImports[report.Fingerprint+"/order/"+id] {
				return true
			}
		}
		for id := range hidden {
			if !p.LegacyImports[report.Fingerprint+"/hidden/"+id] {
				return true
			}
		}
		return false
	}
	report.AlreadyApplied = !pending(prefs)
	if !apply {
		return report, nil
	}
	lock, e := lockFile(filepath.Join(c.DataDir, "preferences.lock"), false)
	if e != nil {
		return report, e
	}
	defer unlock(lock)
	prefs, e = loadPreferences(c)
	if e != nil {
		return report, e
	}
	if !pending(prefs) {
		report.AlreadyApplied = true
		report.Applied = true
		persistMigrationReport(&report)
		return report, nil
	}
	parent := filepath.Dir(report.Backup)
	if e = os.MkdirAll(parent, 0700); e != nil {
		return report, e
	}
	if _, e = os.Stat(report.Backup); os.IsNotExist(e) {
		stage, err := os.MkdirTemp(parent, ".migration-")
		if err != nil {
			return report, err
		}
		defer os.RemoveAll(stage)
		if e = atomicBytes(filepath.Join(stage, "legacy-records.json"), payload); e != nil {
			return report, e
		}
		if userConfig != nil {
			if e = atomicBytes(filepath.Join(stage, "legacy-user-config.json"), userConfig); e != nil {
				return report, e
			}
		}
		prior, err := migrationRead(filepath.Join(c.DataDir, "preferences.json"))
		if err != nil && !os.IsNotExist(err) {
			return report, err
		}
		if err == nil {
			e = atomicBytes(filepath.Join(stage, "preferences-before.json"), prior)
		} else {
			e = atomicJSON(filepath.Join(stage, "preferences-before.json"), prefs)
		}
		if e != nil {
			return report, e
		}
		if e = atomicJSON(filepath.Join(stage, "report.json"), report); e != nil {
			return report, e
		}
		if e = os.Rename(stage, report.Backup); e != nil {
			return report, e
		}
	} else if e != nil {
		return report, e
	} else {
		// Resume a crash after the backup rename but before the preferences commit.
		prior, err := migrationRead(filepath.Join(report.Backup, "legacy-records.json"))
		if err != nil || string(prior) != string(payload) {
			return report, fmt.Errorf("已有迁移备份不匹配，未修改偏好")
		}
	}
	if prefs.LegacyImports == nil {
		prefs.LegacyImports = map[string]bool{}
	}
	for id := range hidden {
		key := report.Fingerprint + "/hidden/" + id
		if !prefs.LegacyImports[key] {
			prefs.Hidden[id] = true
			prefs.LegacyImports[key] = true
		}
	}
	if len(prefs.LegacyOrder) == 0 {
		prefs.LegacyHistory = historyOrderKey(history)
	}
	existing := map[string]bool{}
	for _, id := range prefs.LegacyOrder {
		existing[id] = true
	}
	for _, id := range order {
		key := report.Fingerprint + "/order/" + id
		if !prefs.LegacyImports[key] {
			if !existing[id] {
				prefs.LegacyOrder = append(prefs.LegacyOrder, id)
				existing[id] = true
			}
			prefs.LegacyImports[key] = true
		}
	}
	prefs.LegacyImports[report.Fingerprint] = true
	prefs.Migrated = true
	if e = atomicJSON(filepath.Join(c.DataDir, "preferences.json"), prefs); e != nil {
		return report, e
	}
	report.Applied = true
	persistMigrationReport(&report)
	return report, nil
}

// Keep the original backup plan immutable, but expose the latest incremental result.
func persistMigrationReport(report *MigrationReport) {
	if err := atomicJSON(filepath.Join(report.Backup, "latest-report.json"), report); err != nil {
		report.Notes = append(report.Notes, "迁移偏好已写入，但保存最新报告失败："+err.Error())
	}
}
