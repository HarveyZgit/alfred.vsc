package workflow

import (
	"context"
	"errors"
	"net/http"
	"os"
	"syscall"
	"time"
)

type cleanupResult struct {
	Hidden   int      `json:"hidden"`
	Skipped  int      `json:"skipped"`
	Warnings []string `json:"warnings,omitempty"`
}

// Only confirmed missing/non-directory paths are candidates. Permission errors,
// other I/O failures and unfinished probes must never become hidden records.
func unavailableProjects(ctx context.Context, projects []Project, stat func(string) (os.FileInfo, error)) ([]string, int) {
	type result struct {
		id          string
		unavailable bool
		uncertain   bool
	}
	jobs := make(chan Project, len(projects))
	results := make(chan result, len(projects))
	count := 0
	for _, project := range projects {
		if project.Kind == "folder" {
			jobs <- project
			count++
		}
	}
	close(jobs)
	for i := 0; i < min(4, count); i++ {
		go func() {
			for project := range jobs {
				if ctx.Err() != nil {
					return
				}
				info, err := stat(project.Path)
				missing := os.IsNotExist(err) || errors.Is(err, syscall.ENOTDIR)
				results <- result{
					id:          project.ID,
					unavailable: missing || err == nil && !info.IsDir(),
					uncertain:   err != nil && !missing,
				}
			}
		}()
	}
	ids := []string{}
	skipped := 0
	for i := 0; i < count; i++ {
		select {
		case result := <-results:
			if result.unavailable {
				ids = append(ids, result.id)
			}
			if result.uncertain {
				skipped++
			}
		case <-ctx.Done():
			return ids, skipped + count - i
		}
	}
	return ids, skipped
}

func panelCleanupRecords(c Config, w http.ResponseWriter, r *http.Request) (any, error) {
	if err := panelBody(w, r, &struct{}{}); err != nil {
		return nil, err
	}
	prefs, err := loadPreferences(c)
	if err != nil {
		return nil, err
	}
	history, warnings := loadHistory(c)
	index, _ := loadIndex(c)
	projects := []Project{}
	for _, project := range merge(history, index.Projects) {
		if project.Kind == "folder" && !prefs.Hidden[project.ID] {
			projects = append(projects, project)
		}
	}
	// This explicit panel action scans source records, never directory trees or Git
	// branches. A slow mount cannot indefinitely hold the request or preference lock.
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	ids, skipped := unavailableProjects(ctx, projects, os.Stat)
	if err := r.Context().Err(); err != nil {
		return nil, err
	}
	report := cleanupResult{Skipped: skipped, Warnings: append(warnings, index.Warnings...)}
	if len(ids) == 0 {
		return report, nil
	}
	err = changePreferences(c, func(current *Preferences) error {
		for _, id := range ids {
			if !current.Hidden[id] {
				current.Hidden[id] = true
				report.Hidden++
			}
		}
		return nil
	})
	return report, err
}
