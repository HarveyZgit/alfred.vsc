package workflow

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

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
