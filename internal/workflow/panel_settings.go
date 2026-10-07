package workflow

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
)

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

func panelSaveSettings(c Config, w http.ResponseWriter, r *http.Request) (any, error) {
	var err error
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
}
