package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

// HEAD contains the branch name even when refs are packed or the branch is unborn.
func gitBranch(directory string) string {
	current := directory
	for depth := 0; depth < 64; depth++ {
		gitdir := filepath.Join(current, ".git")
		if info, e := os.Stat(gitdir); e == nil {
			if !info.IsDir() {
				b, e := os.ReadFile(gitdir)
				if e != nil {
					return ""
				}
				pointer := strings.TrimRight(string(b), "\r\n")
				if !strings.HasPrefix(pointer, "gitdir: ") {
					return ""
				}
				gitdir = strings.TrimPrefix(pointer, "gitdir: ")
				if !filepath.IsAbs(gitdir) {
					gitdir = filepath.Join(current, gitdir)
				}
			}
			if b, e := os.ReadFile(filepath.Join(gitdir, "HEAD")); e == nil {
				head := strings.TrimSpace(string(b))
				if strings.HasPrefix(head, "ref: refs/heads/") {
					return strings.TrimPrefix(head, "ref: refs/heads/")
				}
				if len(head) == 40 || len(head) == 64 {
					return head[:7]
				}
				return ""
			}
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return ""
}

type localStatus struct {
	branch   string
	missing  bool
	reason   string
	timedOut bool
}

func visibleBranches(projects []Project) map[string]localStatus {
	// Slow/offline mounts must not hold the entire query hostage. Workers never
	// mutate returned state; late responses go to a buffered channel and are ignored.
	type result struct {
		id     string
		status localStatus
	}
	out := map[string]localStatus{}
	ch := make(chan result, len(projects))
	jobs := make(chan Project, len(projects))
	count := 0
	for _, p := range projects {
		if p.Kind == "folder" {
			jobs <- p
			count++
			out[p.ID] = localStatus{timedOut: true}
		}
	}
	close(jobs)
	done := make(chan struct{})
	defer close(done)
	for i := 0; i < 4; i++ {
		go func() {
			for p := range jobs {
				select {
				case <-done:
					return
				default:
				}
				info, e := os.Stat(p.Path)
				status := localStatus{missing: e != nil || !info.IsDir()}
				switch {
				case os.IsNotExist(e):
					status.reason = "目录不存在"
				case os.IsPermission(e):
					status.reason = "无权访问目录"
				case e != nil:
					status.reason = "目录读取失败：" + e.Error()
				case !info.IsDir():
					status.reason = "路径不是目录"
				}
				if !status.missing {
					status.branch = gitBranch(p.Path)
				}
				ch <- result{p.ID, status}
			}
		}()
	}
	timer := time.NewTimer(30 * time.Millisecond)
	defer timer.Stop()
	for i := 0; i < count; i++ {
		select {
		case r := <-ch:
			out[r.id] = r.status
		case <-timer.C:
			return out
		}
	}
	return out
}
