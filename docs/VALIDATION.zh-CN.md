# 成品验收与报告

验收报告是每次构建的产物，不作为源码入库。原始 JSON、日志、二进制与 SHA-256 位于 `build/`，GitHub Actions 通过 `vsc-native` 和 `validation-Linux` / `validation-macOS` artifact 保存。PR 测试评论链接到对应提交的运行和产物；历史性能数字不能作为新二进制的成绩。

## 本地重跑

需要 Go 1.26+、Node.js 22+、pnpm 11.19.0、Python 3.9+、Git。成品使用者无需这些工具。

```sh
pnpm --dir web install --frozen-lockfile
pnpm --dir web run build
go mod verify
go test -race ./...
go vet ./...
python3 scripts/build_workflow.py
python3 scripts/build_workflow.py --target macos --skip-build --dev
python3 scripts/verify_package.py build/vsc.alfredworkflow build/vsc-dev.alfredworkflow
python3 tests/e2e/acceptance.py --benchmark --output build/acceptance-linux.json
python3 tests/e2e/panel_acceptance.py
```

Go embed 依赖前端产物，所以干净检出必须先构建页面，再执行 Go 测试。打包脚本也会按锁文件安装并构建页面。配置 JSON 与 pnpm 锁文件属于必要源码，和自动生成的验收 JSON 用途不同。

## 各层检查的用途

- **Go race / vet / 模块校验**：验证核心行为、并发和依赖一致性。
- **CLI 成品验收**：真实 SQLite/WAL、Git/worktree、分支切换、30 个并发偏好写进程、四个 IDE 的 argv、来源范围和补迁移；验证没有语言运行时的 PATH 也能查询。
- **面板 API 验收**：编译后服务的会话与来源校验、静态资源、配置/偏好、迁移、正式数据目录复用及关闭。
- **浏览器验收**：实际点击 React 页面，覆盖分页、搜索、隐藏/恢复、固定、IDE、刷新保留会话、配置、迁移、窄屏布局与关闭；检查资源加载、外联和 CSP。该脚本发现的前端回归不能被 Go/API 测试替代。
- **安装包检查**：ZIP 权限与 CRC、Mach-O 双架构、正式/开发 Bundle ID、关键词、Cmd+Shift+V 接线、开发变量隔离。

浏览器检查需要开发机预装 Playwright 与 Chromium，不随工作流分发，也不属于 Go 测试。执行：

```sh
CHROMIUM_PATH=/path/to/chromium node tests/e2e/panel_browser_acceptance.cjs build/vsc-linux-amd64
```

当前 CI 在 Linux/macOS 运行 Go、CLI、API 和打包检查；真实浏览器检查由开发环境运行。诊断日志在检查失败时也上传，方便定位，不通过删除断言绕过失败。

## 性能及实机边界

`--benchmark` 包含新进程启动、SQLite 读取、缓存、搜索、可见本地 HEAD 和 JSON 解析，分别记录首次导入、热查询以及间隔 30ms 的连续输入。每次测量应避免并行编译，报告保留二进制摘要与原始样本。云端尾延迟会波动，不能承诺所有机器/挂载都低于 30ms；管理面板不参与查询调用路径。

Linux 测试无法验证 Alfred 的焦点、快捷键冲突、实际 macOS 应用启动或 SSH/容器连接。macOS CI 验证原生二进制和 arm64 ad-hoc 签名，但不能替代 [Alfred 实机清单](MACOS-ACCEPTANCE.zh-CN.md)。正式版安装包没有 Developer ID 公证。
