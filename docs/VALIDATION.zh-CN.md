# V3 成品验收记录

2026-10-03，Linux x86_64 云环境，AMD EPYC 9V74，Go 1.27.1，`CGO_ENABLED=0`。以下首版基线对应下方明确标注 SHA-256 的二进制；迁移更新后的验收见文末。基线实测，不能直接推断 macOS / Alfred 或慢网络盘的延迟。

## 已完成

- 20 个 Go 功能测试通过，race detector 未报告竞争；`go vet ./...` 通过。
- `go mod verify` 验证所有模块与锁定校验和一致。
- 实际 Linux 二进制的 8 组集成验收通过：目录并集与排除、真实 Git/worktree、最新分支与 WAL 历史、隐藏恢复和 30 个并发写进程、四个 IDE argv 与错误退出、只读已提交视图和空历史、目录浏览、后台扫描、无语言工具 PATH 查询（部分归为同组）。
- `file` / `readelf` 确认 Linux ELF 静态链接，无动态依赖节。
- macOS 13+ 的 arm64、amd64 交叉编译成功，universal slices 与原始二进制逐字节一致；工作流 ZIP、可执行权限、CRC、Alfred 动作图和版本一致性验证通过。

## 完整查询性能

测量包含新进程启动、SQLite 历史读取、缓存、搜索、可见本地 HEAD、结果输出与测试端解析。数据为真实本地目录和 HEAD 文件；默认最多 50 条结果。每规模 36 个顺序样本、40 个间隔 30ms 的输入样本。测量时没有并行编译。

| 项目数 | 首次历史导入 | 已有缓存查询 P95 | 连续输入 P95 | 连续输入最大值 | 超过 30ms |
| --- | ---: | ---: | ---: | ---: | ---: |
| 1000 | 11.841ms | 5.833ms | 6.798ms | 7.613ms | 0 / 40 |
| 10000 | 70.462ms | 21.685ms | 22.796ms | 27.397ms | 0 / 40 |

首次导入不属于热查询。一万目录首次导入约 70ms；这不是“所有场景零延迟”的承诺。数据库锁定使用受限等待和历史回退，本地分支读取预算 30ms，超时显示提示而不展示缓存旧分支。慢挂载和首次大量历史变更仍需实际机器确认。

原始数据：`build/acceptance.json`。测量二进制 SHA-256：

```
164a14475af9a6ef70ed5d0e82974d6d3f001e581c7b3d0aae7fb6a85309beb5
```

重新编译后如摘要变化，应重新运行相关验收，不应把本报告的延迟标为新二进制成绩。

## 尚未在此环境执行

Alfred UI 的快速输入取消、第二级 IDE 菜单和修饰键；真实 macOS 应用启动及新窗口；真实 SSH / Dev Container / Zed 连接；Intel / Apple Silicon 上的系统兼容性与签名验证。CI 已配置 Linux 和 macOS 作业，但本地新增 CI 尚未在 GitHub 执行，不计入已通过项。

请使用隔离包 `build/vsc-dev.alfredworkflow`（关键词 `vscd`），按 [macOS 清单](MACOS-ACCEPTANCE.zh-CN.md) 验收。正式包和开发包均未做 Developer ID 公证。

## 2026-10-04 迁移更新验收

新增 6 个迁移测试，总计 26 个 Go 功能测试通过，race detector 和 go vet 通过。新版实际二进制通过 9 组集成验收，新增脚本预览不落盘、仅迁移当前范围、未迁移清单、逐字节备份、重跑补迁移和用户取消隐藏不被重置。

原始结果：[linux-migration-acceptance-2026-10-04.json](validation/linux-migration-acceptance-2026-10-04.json)。本次 SHA-256：`5746d345f4695f504ea42bf00e501cbe259553a721b7b6b05382b7dd8a03ec04`。

| 项目数 | 首次导入 | 热查询 P95 | 连续输入 P95 | 连续输入最大值 | 超过 30ms |
| --- | ---: | ---: | ---: | ---: | ---: |
| 1000 | 11.398ms | 5.819ms | 6.141ms | 10.047ms | 0 / 40 |
| 10000 | 82.751ms | 25.45ms | 55.661ms | 71.561ms | 8 / 40 |

本轮一万项目连续输入出现了延迟尖峰，未达到每次都在 30ms 内完成；不能沿用首版基线的“0 次超时”结论。保留原始数据供 CI 与 Mac 实机对比，暂不把云端调度波动或代码开销中的任一个认定为唯一原因。

GitHub macOS runner 已通过该提交的 26 个 Go 测试、race、vet 和原生集成验收。首次后续检查把 Go 默认无签名的 amd64 slice 当成已签名产物，现将检查明确限定为 Go 自带 ad-hoc 签名的 arm64，并单独执行 universal binary；新 CI 状态以 PR 检查为准。


## 2026-10-04 管理面板验收

新增 4 个 Go 场景测试，总计 30 个功能测试及 race、vet、模块校验通过；Linux 实际二进制通过原有 9 组验收和新增 7 组面板集成验收。后者包含空 PATH 下启动内嵌页面 / API、会话与来源校验、配置覆盖、分支与目录状态、隐藏恢复影响实际 query、迁移预览与最新结果、服务关闭，以及替身浏览器验证 Alfred 启动命令及时退出。macOS 双架构交叉编译和包结构验证通过。

另外使用 Linux Chromium 151.0.7922.173 + Playwright 执行真实页面交互：分页、输入搜索、隐藏恢复、固定、项目 IDE、浏览器刷新保持会话、配置保存、迁移预览/执行、390px 宽度布局和关闭服务；无页面异常。浏览器验收为开发机可选测试，需预装 Playwright / Chromium，可用 `node scripts/panel_browser_acceptance.cjs [binary]` 重跑，`CHROMIUM_PATH` 指定浏览器；这些不是工作流运行依赖。截图见 README。

最终测量二进制 SHA-256：`739e9510d561bce31a7fb0e89b40ac0bae1ead5e3a8f674a2497ee0be8cb863a`。

| 项目数 | 首次导入 | 热查询 P95 | 连续输入 P95 | 连续输入最大值 | 超过 30ms |
| --- | ---: | ---: | ---: | ---: | ---: |
| 1000 | 12.896ms | 7.303ms | 10.012ms | 32.060ms | 1 / 40 |
| 10000 | 88.794ms | 28.667ms | 29.353ms | 54.283ms | 2 / 40 |

面板服务不在搜索调用路径内；本次云端仍出现尾部延迟，不能承诺任何环境都低于 30ms。关闭服务响应收尾修正后重新构建并测量，以上对应最终本地二进制，不沿用前一轮结果。真实 Alfred、Mac 默认浏览器和 IDE / 远程连接仍按 Mac 清单验收。

原始报告：[查询及性能](validation/linux-panel-acceptance-2026-10-04.json)、[原生面板](validation/panel-api-acceptance-2026-10-04.json)、[浏览器交互](validation/panel-browser-acceptance-2026-10-04.json)。


## 2026-10-05 快捷键、安装身份与结构整理验收

正式包固定 `com.harvey.alfredapp.vsc.v3`、入口 `vsc` / `vscli`，开发包固定 `.v3.dev`、入口 `vscd` / `vscdli`。包验证检查正式热键 Cmd+Shift+V 精确连接 SEARCH、开发包无热键、正式包无开发变量 / 临时数据路径，并继续验证 Mach-O 切片、ZIP 权限与图标资源。旧 ID、错误关键词、热键误连 OPEN、开发变量或临时路径污染的篡改包均被验证器拒绝。

整理后本地通过 31 个 Go 测试及 race、vet、模块校验；原有 9 组 CLI 集成验收通过，面板集成扩展为 8 组，新增通过 Alfred 正式版数据目录直接复用配置 / 项目 IDE 偏好且不修改配置偏好文件的验证。Chromium 真实页面交互继续通过，原生 ES 模块的路由、类型与拒绝非法路径另有 Go 测试。Go / Ruff / Prettier 格式检查通过。

测量二进制 SHA-256：`ed6fda3eb7776836468a90ababcc0b8f1484f8c5b95ee2c62505efef23ef281f`。

| 项目数 | 首次导入 | 热查询 P95 | 连续输入 P95 | 连续输入最大值 | 超过 30ms |
| --- | ---: | ---: | ---: | ---: | ---: |
| 1000 | 13.551ms | 10.453ms | 9.651ms | 13.202ms | 0 / 40 |
| 10000 | 87.055ms | 29.638ms | 22.995ms | 23.236ms | 0 / 40 |

这是当前二进制的一轮云端样本，不作为结构整理改善性能的因果结论，也不保证所有机器和负载都低于 30ms。真实 Mac 导入升级、热键注册 / 冲突和 Alfred 唤醒仍按实机清单验证。

原始报告：[CLI / 性能](validation/linux-upgrade-acceptance-2026-10-05.json)、[正式目录与面板](validation/panel-upgrade-acceptance-2026-10-05.json)、[浏览器交互](validation/browser-refactor-acceptance-2026-10-05.json)。

本轮开始时 PR 最新提交的 Linux / macOS 检查均已通过；历史红色运行对应早期 Mac 临时目录规范化和签名检查问题，已有修复。CI 本轮新增 Go 格式和生成 plist 一致性检查，PR 只运行一轮，取消已被新提交替代的运行，保留历史失败供追溯。人工 Review 要求属于合并规则，不等同于 CI 失败。
