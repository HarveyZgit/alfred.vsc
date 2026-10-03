# V3 成品验收记录

2026-10-03，Linux x86_64 云环境，AMD EPYC 9V74，Go 1.27.1，`CGO_ENABLED=0`。以下是当前二进制的实测，不能直接推断 macOS / Alfred 或慢网络盘的延迟。

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
