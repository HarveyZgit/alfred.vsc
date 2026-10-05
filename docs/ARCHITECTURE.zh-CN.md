# V3 技术设计与选型

## 目标与边界

项目集合严格取 VS Code 最近打开的本地 / 远程目录和配置根目录下 Git 仓库的并集。VS Code 是历史事实来源，不导入其他 IDE 的历史，不建立独立 MRU，不收录文件或 workspace 文件。Zed、Trae、Cursor 作为打开方式。

可见本地项目每次查询读取当前 HEAD；远程项目从不探测分支或连接网络。历史排序随 VS Code 的记录更新：用其他 IDE 打开不会伪造 VS Code 历史顺序。

## Rust 与 Go

同一 Linux 机器、同一数据、每次启动新进程、前 50 个结果读取 HEAD 的原型测量：

| 项目数 | Rust 原型 P95 | Go 原型 P95 |
| --- | ---: | ---: |
| 1000 | 2.365ms | 4.835ms |
| 10000 | 9.578ms | 22.471ms |

Rust 在这一原型中更快。原型只包含快照查询和分支读取，不能直接当作完整产品成绩。两者都可交付无需语言运行时的单二进制。

本次选择 Go 的理由是完整实现、并发维护和 Linux 到 macOS 两种架构的交叉编译更直接。`modernc.org/sqlite` 将 SQLite 编入程序，`CGO_ENABLED=0` 不依赖额外 SQLite 动态库或跨平台 C 编译器。Rust 若使用 bundled SQLite，通常需要额外配置 macOS 目标的 C 编译与链接工具链。此处是交付复杂度的取舍，不是 Go 的原始性能优于 Rust。

最终是否满足性能要求由编译成品测试决定。后续如果真实项目量、网络挂载或 macOS 测量无法满足预算，应先依据 profile 优化，再判断是否迁移 Rust，而不是只更换语言。

## 查询数据流

```text
一次查询
  ├─ 偏好：隐藏 / 固定 / 项目 IDE（JSON）
  ├─ VS Code SQLite 只读获取当前历史 JSON
  │    └─ 内容哈希相同：读取规范化二进制缓存
  │       内容改变：解析目录 → 更新可丢弃缓存
  ├─ 读取 Git 仓库索引；过期则启动独立后台扫描
  ├─ 合并去重 → 模糊匹配 → 排序 → 前 50 项
  ├─ 仅本地结果：并发读取当前 HEAD，30ms 总预算
  └─ Alfred JSON（打开请求携带 URI、IDE、新窗口标记）
```

缓存采用带版本头的 Go gob 格式，偏好仍是可读 JSON。缓存不是另一份历史权威；数据库每次重新读取，内容哈希用于避免重复解析和规范化。普通查询没有递归扫描、没有每项目启动 Git 进程、没有网络请求或常驻 daemon。

SQLite 使用只读 URI、query_only、30ms busy timeout 和 100ms context。支持 WAL 的已提交视图。新键存在但历史为空时，以空结果为准；共享数据库没有历史键才继续尝试旧存储。数据库暂时不可用时只回退同一路径的快照，并附带诊断。

仓库索引按根目录和扫描深度生成键；更换配置不复用错误来源的索引。扫描有时间、目录数量和深度限制，不完整扫描保留上次结果并记录警告。启动锁和五秒间隔抑制连续输入引起的重复扫描进程。

查询快照原子替换；偏好额外使用文件锁及读取后修改，避免多个 Alfred 操作并发丢失更新。缓存损坏可重建，偏好损坏会报错而非静默覆盖。旧版使用显式迁移脚本：备份完整输入，只导入新版来源范围内的隐藏状态和旧使用顺序，范围外逐项列出补迁移方法。旧顺序只在 VS Code 历史与迁移时一致时用于同分排序。每条迁移有标记，补齐来源后可重跑，不重置用户之后取消隐藏的操作。

## 搜索和打开

名称、完整路径、远程 authority 和标签参与 Unicode 模糊搜索。名称匹配优先，其次固定项目，最后保留来源顺序。合并和匹配阶段预分配容量，匹配结果保存项目下标，避免重复复制完整项目对象。

四个 IDE 共用打开计划，但参数由适配器决定。程序直接以 argv 调用可执行文件，不将路径插入 shell 命令。远程 URI 保留语义；Zed 仅接受可可靠转换的 SSH URI。容器和未知远程 authority 交由来源 IDE VS Code。界面可以报告 IDE 不可用或参数不支持，而不是假装成功。只有启动命令成功后才记住 IDE 偏好。

Alfred 负责输入变化时终止前一查询并显示当前查询结果。每次 CLI 调用没有共享可变的搜索结果。Linux 并发测试验证进程与数据完整性，不能验证 Alfred 的旧结果取消和界面焦点行为。

## 验收方法

`go test -race ./...` 验证源码行为与并发；`scripts/acceptance.py` 用实际二进制、真实 SQLite / WAL、真实 Git / worktree，以及替身 IDE 子进程验证集成。

性能样本包含进程启动、SQLite 获取历史、缓存读取、匹配、可见 HEAD、JSON 输出和测试端接收解析。每规模 36 次顺序查询，随后 40 次、间隔 30ms 的连续输入；返回上限 50。记录首次导入和缓存已存在两种情况。首次大量历史导入涉及所有目录的规范化，不应以热查询延迟宣传。具体成品结果见 [验收记录](VALIDATION.zh-CN.md) 和 `build/acceptance.json`。

打包从 Linux 交叉编译 macOS arm64 / amd64，将两份不改动的 Mach-O slice 合并为 universal binary。验证架构、偏移对齐、内容一致、ZIP 权限、CRC、Alfred 动作图和版本。没有在 Linux 假执行 macOS 二进制；真实 Alfred UI / IDE / SSH / Container 由 [macOS 清单](MACOS-ACCEPTANCE.zh-CN.md) 验收。


## 按需记录管理面板

`vsc manage` 启动独立的本机 HTTP 子进程并打开默认浏览器；`manage --serve` 用于前台运行和验收。页面通过 Go embed 随二进制分发，无前端运行时或外部资源。仅绑定 127.0.0.1 随机端口，API 要求 256 位随机令牌，同时校验 Host 与 Origin；页面禁止嵌入和执行内联脚本，用户路径全部通过 textContent 渲染。20 分钟没有 API 操作自动退出，也支持手动关闭。

管理列表读取同一 VS Code 历史 / 索引和偏好，每页最多 50 条本地状态探测，远程不探测。隐藏、固定和 IDE 更新继续使用 preferences.lock 与原子替换。递归扫描及迁移由显式按钮触发，在管理服务内部串行执行，不加入 query 流程。搜索不依赖面板服务。

config.json 新增可选 managed 对象，仅承载面板根目录、默认 IDE 与 CLI 路径，在 Alfred 环境变量之后应用；未使用面板时沿用原有优先级。配置写入使用单独文件锁，保留不认识的高级字段，可一键移除 managed 恢复原有配置。每个请求重新读取配置，使其他窗口和 CLI 修改可见。

迁移保留最初 report.json 不变，成功执行后另写 latest-report.json 记录最新补迁移结果。预览不落盘；旧版只有首次报告时标注历史快照。报告保存失败不会假称已提交的偏好迁移失败，而是在结果 notes 中说明。


## 源码职责与文件组织

Go 核心保留 `internal/workflow` 包，按职责分文件，跨文件仍使用未导出函数；这次整理不改变数据格式、锁、来源范围和查询流程。

| 位置 | 职责 |
| --- | --- |
| `cmd/vsc/main.go` | 进程入口 |
| `internal/workflow/cli.go`、`config.go` | CLI 分派与配置优先级 |
| `model.go`、`history.go`、`index.go`、`git.go` | 项目模型、VS Code 历史、Git 索引与分支 |
| `query.go`、`search.go`、`feedback.go`、`browse.go` | 查询编排、排名、Alfred JSON 和目录浏览 |
| `open.go` | IDE 解析、打开计划与执行 |
| `preferences.go`、`storage.go` | 用户偏好、文件锁、原子写入和缓存编码 |
| `migration.go`、`migration_source.go` | 迁移事务、旧数据解析与来源判断 |
| `panel.go`、`panel_routes.go`、`panel_records.go`、`panel_settings.go`、`panel_migration.go` | 面板进程生命周期、HTTP 路由和各业务接口 |
| `internal/workflow/panel/` | HTML/CSS，以及 `api` / `dom` / `records` / `settings` / `migration` 原生 JS 模块；`app.js` 只负责初始化 |
| `*_test.go`、`test_helpers_test.go` | 按领域命名的测试和共享 fixture |
| `public/`、`public/assets/` | Alfred 模板、工作流图标和结果图标 |
| `scripts/` | 生成、打包、CLI / 面板 / 浏览器验收与包验证；旧版迁移 shell 入口 |
| `docs/validation/` | 与二进制 SHA-256 对应的原始验收报告 |

JS 模块直接由二进制内嵌服务返回，无打包器、CDN 或新增运行依赖。静态 HTTP 路由使用明确文件白名单，并检查 MIME 类型和非法路径。

正式包 ID 固定为 `com.harvey.alfredapp.vsc.v3`，开发包为 `.v3.dev`；正式包不携带开发模式变量或临时数据路径。默认 Hotkey 使用 Alfred 已有导出协议（V 键码 9，⌘⇧ 修饰掩码 1179648），连接项目 Script Filter，输入参数为空；开发包不包含该热键。
