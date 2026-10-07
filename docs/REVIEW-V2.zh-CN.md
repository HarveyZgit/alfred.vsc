> 本文是 V3 重写前的 V2 审查记录。代码位置和缺陷描述对应旧版本；当前实现以 README 和 ARCHITECTURE 为准。

# alfred.vsc 全面评审与迭代建议

评审日期：2026-10-03。仓库基线：`defac625c88dcffe216415f3ec539d50887adf29`，当前包版本 2.1.0。

后续范围已收敛为“VS Code 目录/远程目录历史 + 配置根下 Git 仓库；其他 IDE 仅用于打开；远程不显示分支”。最新技术选型与连续输入实验见 [聚焦后的设计评估](/workspace/alfred-vsc-review/design-evaluation/DESIGN.zh-CN.md)，其范围优先于本文早期的多 IDE 历史合并建议。

本报告覆盖全部 3 个运行时 Python 模块、2 个开发/构建脚本、Alfred 工作流连接与配置、README、CHANGELOG、OpenSpec 及相关提交历史。结合公开竞品源码、临时数据复现和性能实验。评审期间没有修改产品源码。

用户场景：多个 IDE 混用，同时使用本地、SSH 和容器项目。早期自建记录是为规避当时 JS 实现访问 VS Code 数据库的性能问题，后来迁移到 Python 后保留了这套机制。技术栈可重新选择，但性能必须有可验证的保证。

## 1. 建议结论

将工具定位为 **Alfred 中统一、快速、可靠的多 IDE 项目入口**。

推荐先重构数据来源、打开行为和搜索链路，保留 Python 完成一个可测量的版本。当前没有证据表明 Python 是主要瓶颈。若最小重构后仍无法达到真实 Mac 上的响应预算，或 Python 分发依赖严重影响安装体验，再用同一套契约与基准比较 Swift/Go 实现。

数据架构应调整为：

1. **IDE 历史是权威来源**：VS Code、Cursor 及其他支持的编辑器按各自配置和存储位置读取；保留来源、远程信息、工作区类型。
2. **缓存只是可丢弃的性能优化**：自动失效和恢复，用户无需执行“重建”才能看到刚在 IDE 打开的项目。
3. **只维护工具自己的数据**：收藏、别名、标签、隐藏状态、每项目默认 IDE、用户明确添加但尚未打开的项目。若某项功能不需要，就不新增对应存储。
4. **配置目录发现是补充来源**：发现从未在任何 IDE 打开过的项目，不能用它覆盖 IDE 历史。扫描索引可以重建，但不再是历史记录的权威副本。

推荐产品收益顺序：能正确打开 → 结果新鲜可靠 → 搜索流畅 → 多 IDE/远程选择省心 → 收藏和辅助动作。

## 2. 主要代码发现

P1 表示直接破坏核心操作或显著拖慢常见场景，应进入首批修复；P2 表示场景性缺陷或可维护性风险。以下区分已复现、静态证据与需要 macOS 实测的结论。

### P1：默认发行配置无法普遍使用，打开失败仍更新 MRU

- [public/info.plist:554](/workspace/alfred.vsc/public/info.plist:554) 的默认 IDE 指向作者个人目录下的 Antigravity CLI，备用 IDE 为空；修饰键提示却固定为 Trae。
- [public/info.plist:155](/workspace/alfred.vsc/public/info.plist:155) 和备用打开脚本中的可执行文件变量未加引号。合法的 `.app` 内 CLI 路径包含空格时会被 shell 拆开。
- 打开命令之后无条件执行 `selected`，打开失败也会更新最近使用，最终 shell 还可能返回成功。
- **已复现**：使用临时、无害的假 IDE 可执行文件及实际 plist 脚本；含空格的可执行路径未执行，MRU 步骤却执行且最终退出码为 0。
- 建议：自动发现已安装 IDE，提供明确配置入口；以参数数组启动，支持应用 bundle ID 与 CLI 两种适配；只在启动请求被接受后记录使用行为。启动命令成功与远程连接完成应分别表述。

### P1：每条搜索结果都会重新读取整个缓存

- [src/vsc.py:55](/workspace/alfred.vsc/src/vsc.py:55)、[src/vsc.py:191](/workspace/alfred.vsc/src/vsc.py:191)：`format_for_alfred()` 为每个结果调用 `get_cached_branch()`，后者重新打开并解析完整 JSON。
- 总记录 N、命中 M 时，这部分接近 O(N×M)，宽泛搜索时接近 O(N²)。同时没有结果数量上限。
- **已复现**：渲染 20 个结果，缓存读取 20 次；2,000 条宽泛命中约 3.78 秒。相同数据、相同结果的成对诊断中，单次读取将中位数从 3,619 ms 降到 45.6 ms。
- 建议：一次读取、一次规范化、一次排序，再渲染前 50–100 条；更多结果有明确入口。分支等附加信息不能阻塞首屏。

### P1：初始搜索、重建和历史更新使用不同数据规则

- [src/vsc.py:342](/workspace/alfred.vsc/src/vsc.py:342) 首次只取 VS Code DB/菜单历史，不包含配置目录；缓存非空后不再读取历史。
- [src/cli.py:29](/workspace/alfred.vsc/src/cli.py:29) 重建只取配置目录，覆盖已有记录，不合并 IDE 历史。
- **已复现**：首次历史可见、本地配置目录项目不可见；更新模拟 IDE DB 后新历史不出现；重建后目录项目出现、原历史项目消失。
- 建议：统一所有入口的来源合并规则，更新由来源驱动，禁止“重建目录索引”改变历史数据集合。

### P1：删除最后一条历史会在下一次搜索中重新出现

- [src/cli.py:66](/workspace/alfred.vsc/src/cli.py:66) 删除会记录 trash。
- [src/vsc.py:377](/workspace/alfred.vsc/src/vsc.py:377) 空缓存回退读取 IDE 历史时不应用 trash。
- **已复现**：删除唯一一条历史后，再搜索，它立即被重新导入。
- 建议：将“在本工具隐藏”作为独立偏好，在所有来源合并之后统一过滤；提供撤销/恢复入口。

### P1：远程项目有识别标签，却没有完整的打开协议

- [src/shared.py:119](/workspace/alfred.vsc/src/shared.py:119) 用字符串包含 `remote` 判断远程，本地 `remote-tools` 目录也会误分类。
- [src/shared.py:348](/workspace/alfred.vsc/src/shared.py:348) 导入时把记录压成一个 path，丢弃 `remoteAuthority`、来源 IDE、用户 label 和 workspace 类型。
- plist 将远程 URI 作为普通位置参数传给 CLI，没有 `--folder-uri`、`--file-uri`、`--remote` 或编辑器深链路适配。
- **已复现**：本地名称误分类、实际 shell 参数缺少远程路由。**未在真实 IDE 验证打开结果**。
- VS Code 官方文档明确区分远程文件夹与文件的 CLI 参数，见来源 S6。容器 authority 更不能当成本地路径处理。
- 建议：解析 URI 的 scheme/authority；保留原始远程标识；按“目标类型 × 编辑器能力”构造命令。容器优先交还原 IDE，不默认假设各 VS Code 衍生版完全兼容。

### P1：缓存写入存在覆盖用户操作的风险

- [src/shared.py:619](/workspace/alfred.vsc/src/shared.py:619) 与 [src/shared.py:631](/workspace/alfred.vsc/src/shared.py:631)：直接覆盖 JSON，既无原子替换，也无多进程更新协调；解析失败静默返回空记录。
- 分支刷新、删除、选择、重建都可能执行读取—修改—写回整个文档。
- **确定性交错实验已复现**：先读旧快照，另一写者完成删除，再写回旧快照的分支更新，删除的记录恢复。没有把它描述为真实 Alfred 并发压力测试。
- 建议：派生缓存和用户偏好分离；缓存采用临时文件 + 原子替换，偏好更新使用事务或受锁保护的读改写。原子替换本身不能解决“旧快照覆盖新操作”。

### P2：历史读取对 IDE 版本和存储布局适配不足

- 当前固定一个 `VSC_IDE_PATH`，且只查询 `history.recentlyOpenedPathsList`；只解析 `storage.json` 中旧的菜单结构。
- 公开 VS Code 源码包含 `history.recentlyOpenedPathsList` 和 `recently.opened` 两套存储路径，属于不同实现，不能笼统宣称所有桌面版本都已切换键名。
- Raycast 当前实现还检查 `product.json` 中的共享存储目录配置，以及 backup/profile 相关来源。
- **已复现**：仅包含 `recently.opened` 的数据库在本工具中返回空结果。**尚未证实用户正在使用的具体 IDE 版本受此影响**。
- 建议：按已验证的编辑器/版本建立 provider 兼容矩阵，支持实际存在的存储布局；记录明确错误，而非猜测所有 IDE 都有相同 schema。

### P2：目录发现和浏览使用不一致的排除规则

- [src/shared.py:422](/workspace/alfred.vsc/src/shared.py:422) 的仓库递归发现不使用 `SKIP_DIRS`；遇到根仓库即停止，因此嵌套仓库也不会继续发现。
- [src/shared.py:503](/workspace/alfred.vsc/src/shared.py:503) 的目录 fallback 连隐藏目录也加入。
- **已复现**：依赖目录中的 Git 仓库被索引，隐藏目录进入 fallback 结果。
- 建议：统一配置化排除规则、符号链接去重/循环处理、扫描预算和嵌套仓库策略；网络磁盘或离线目录不阻塞其他来源。

### P2：浏览动作没有成为完整的项目使用流程

- [src/cli.py:103](/workspace/alfred.vsc/src/cli.py:103) 的 `selected` 只移动已存在记录；新浏览到的目录无法加入 MRU。
- 多个根目录下同名目录会生成相同 autocomplete 路径，钻入后混合多个根的结果。[src/vsc.py:255](/workspace/alfred.vsc/src/vsc.py:255)
- 空目录、目录不存在、权限不足都可能显示相同的“没有更多子目录”提示。
- 建议：打开后由 IDE 历史自动收录；浏览状态保留根目录身份；区分空、失效、无权限；始终提供明确的上级导航。

### P2：Git 与路径类型的边界处理不完整

- [src/shared.py:157](/workspace/alfred.vsc/src/shared.py:157) 将相对 `gitdir` 相对于进程工作目录解析，未相对于 `.git` 所在目录解析；临时 fixture 已复现分支缺失。
- 不存在的目录被判为 file；类型来自当前文件系统可达性，而不是 IDE 记录类型。[src/shared.py:124](/workspace/alfred.vsc/src/shared.py:124)
- URI 解码与 ID 生成分散，DB、菜单和目录来源的编码形式可能不同，容易影响去重、特殊字符显示与远程 URI 保真。
- 建议：记录保留类型与原始 URI；失效状态单独记录；Git 查询只针对本地、可见结果，并覆盖 worktree、submodule 和 detached HEAD。

### P2：首次使用与发布质量缺少必要的保障

- `userconfigurationconfig` 为空；空结果没有配置、诊断或恢复入口。
- 主关键词副标题仍提示旧 Node 时代的 `npm install -g vsc`，与当前 Python 包和版本无关。[public/info.plist:235](/workspace/alfred.vsc/public/info.plist:235)
- 固定 `/usr/bin/python3` 没有最低版本检测；源码使用 Python 3.9+ 类型语法。不能从 Linux 上有 Python 推断用户 Mac 已有可用解释器。
- 开发安装假定 Alfred preferences 位于固定目录，可能不适配自定义同步位置；失败布尔值未转换成非零进程退出码。
- 打包脚本读取 package.json 版本用于显示，却不校验 plist 版本；无自动化回归套件、发布兼容矩阵和性能门槛。
- 建议：提供 Alfred 内的配置与诊断入口；制作来源、URI、打开 argv、迁移、并发和性能契约测试，增加真实 macOS 集成检查。

### 文档偏差：目录非递归是有意设计

提交 `0b435b5`（2026-04-09）明确移除递归浏览，目的是避免搜索当前目录时出现所有深层子目录。README、CHANGELOG、OpenSpec 和函数注释没有同步。

因此应修正文档并确认“全局项目搜索”和“当前层级浏览”的分工，保留逐层浏览这一有价值的交互。前一轮环境检查将它称为代码缺陷，完整审查提交历史后应更正为文档/规格不一致。

## 3. 性能证据与旧数据库方案的重新评估

### 当前搜索的增长曲线

Linux 云环境、Python 3.12.14、合成记录、热文件系统。每档执行 5 次完整 CLI 子进程；查询 `project` 命中所有记录，验证输出数量。该实验刻意隔离缓存解析/渲染扩展性，不代表真实项目分布或 Alfred UI 延迟。

| 缓存记录数 | 宽泛搜索中位数 | 单条命中单次样本 |
|---:|---:|---:|
| 100 | 40.05 ms | 31.03 ms |
| 500 | 249.10 ms | 35.79 ms |
| 1,000 | 954.94 ms | 39.82 ms |
| 2,000 | 3,779.61 ms | 37.71 ms |

同一个 2,000 条 fixture、同一个查询的成对比较，交替运行各 5 次：

| 实现 | 中位数 | 输出 |
|---|---:|---|
| 当前代码 | 3,619.43 ms | 2,000 条 |
| 仅诊断进程内替换为单次缓存读取 | 45.59 ms | 与当前代码完全相同 |

这不是已经交付的优化版本，而是瓶颈定位实验。它说明应优先消除重复全量读取，而不是先换语言。

### SQLite 本身是否仍然慢

构造约 16 MiB、含无关条目的 SQLite 数据库，以主键查询单条历史 JSON；保持 WAL 和已有连接，模拟 IDE 正在运行。与同一份历史 JSON 文件比较，每档 15 次，均验证结果一致。

| 历史条目数 | SQLite 打开、查询、解码、关闭中位数 | JSON 文件读取、解码中位数 |
|---:|---:|---:|
| 100 | 0.074 ms | 0.025 ms |
| 1,000 | 0.268 ms | 0.210 ms |
| 2,000 | 0.474 ms | 0.397 ms |
| 5,000 | 1.063 ms | 0.947 ms |

2,000 个实际存在的临时项目目录，同一套现有搜索/渲染代码，修正重复读缓存后：

| 完整诊断子进程 | 中位数 |
|---|---:|
| 直读 SQLite，沿用现有逐记录日志、图标和路径处理 | 119.23 ms |
| 读取一次已经规范化的 JSON 缓存 | 44.14 ms |

两种方式输出完全一致。两者之差还包含规范化、逐记录文件系统检查和日志开销，不能归因于 SQLite 查询本身。WAL 模式下存在未提交写事务时，读取已提交历史也通过验证，中位数 0.488 ms；这不保证所有真实数据库布局和锁状态都同样顺畅。

**决策**：值得优先验证“直接只读 IDE 历史 + 轻量投影”。目前没有必要为了查询一条历史 JSON 维护一套独立、人工刷新的历史数据库。但真实 Mac 冷启动、多 IDE、源 DB 锁竞争、网络卷、数据库布局变化仍要实测。也不能根据这次 Python 实验倒推旧 JS 版本的慢一定是哪一层造成的。

### 性能验收目标（建议值，尚未实测达成）

- 常用规模：合计 1,000 项目、最多 3 个已启用 IDE 来源，查询到 JSON 的 P95 ≤100 ms。
- 大规模：10,000 个合并项目时，缓存命中查询 P95 ≤150 ms，输出前 50–100 条。
- 冷启动：已知本地历史的首批结果 P95 ≤200 ms；初次目录扫描异步补充，不要求扫描完成才出结果。
- 数据新鲜度：IDE 提交历史后，正常情况下下次打开工具或 2 秒内刷新；若源受锁定/离线影响，保留上次有效结果并标明状态。
- 编辑器启动请求的构造/提交目标 ≤100 ms；IDE 冷启动、SSH 建连、容器构建单独计时。
- macOS 实测每档至少 30 次，区分首进程、热文件系统、缓存命中、源更新、慢磁盘与锁竞争；固定设备和 OS 后记录 P50/P95、峰值内存、调用失败率。

## 4. “跟 VS Code 同步”的具体边界

| 需求 | 推荐实现 | 是否需要自己的历史库 |
|---|---|---|
| 展示 VS Code/Cursor 的最新历史 | 读取实际 IDE 历史，按来源更新 | 不需要 |
| 从 Alfred 打开后也进入 IDE 最近使用 | 使用 IDE 支持的打开方式，让 IDE 自己写入 | 不需要维护权威副本 |
| 在不同 IDE 历史里统一搜索 | 只读合并视图，保留多来源和原始元数据 | 只需可丢弃投影/缓存 |
| 尚未打开的目录也可搜索 | 文件系统发现或显式收藏 | 需目录派生索引或用户收藏 |
| 每项目默认 IDE、别名、收藏、隐藏 | 保存独立偏好，不混入 IDE 历史 | 需少量自有偏好数据 |
| 在 Alfred 删除，同时修改 IDE 原生最近列表 | 需要验证该 IDE 的受支持接口和运行中同步行为 | 不建议首版直接改内部 DB |

不建议默认反向修改 IDE 内部 DB：VS Code 源码通过内部 storage service 读写历史，外部 SQL 成功不等于运行中的 IDE 已更新内存状态；多个进程可能覆盖彼此修改，schema 和共享存储位置也有变化。Raycast 有直接修改数据库的实现，这证明存在这种方案，但不证明它在所有版本都可靠；部分操作还明确提示重启编辑器才能同步。

建议首先完成只读实时展示与正常打开后的自然回流。隐藏用工具本地偏好表示。如未来必须做强双向同步，再评估受支持接口或可选 IDE companion；不能假设存在稳定公开的“读取/删除所有最近项目”API。

缓存也不必一次性全部删除。若真实机器上多来源读取超过预算，可保存短期派生快照：来源版本/时间戳、生成时间、schema 版本明确，自动重算。WAL 数据库不能仅监测主 DB 文件 mtime，也不能使用忽略 WAL 的 `immutable=1` 来求快。

## 5. 竞品对比

以下基于本次获取的项目文档、manifest 和源码，非 Mac 上的实机体验评测。不比较未经测量的竞品耗时；“未见”仅表示所审查实现中没有对应能力，不代表整个生态无法实现。公开 main/master 分支会变化，本地保留了源文件与摘要。

| 方案 | 已核实的能力 | 对我们最有价值的启发/边界 |
|---|---|---|
| Raycast Visual Studio Code（S1） | VS Code/Cursor/多种衍生编辑器选择；固定项目；文件夹/文件/workspace/远程分类；远程专用打开方式；Finder 选中项；终端、复制路径、其他应用打开；加载失败的配置入口；分支延后加载与 TTL | 基础体验标杆。当前审查的历史读取按单个 build 偏好选择来源，多 IDE 历史统一合并仍有发挥空间 |
| Raycast Cursor（S2） | 最近项目、固定、类型筛选、远程 URI、Finder 打开、新窗口；manifest 中还有活动工作区与扩展管理命令 | 证明项目入口应覆盖“搜索后做什么”。扩展市场/文档搜索不必进入我们的首批范围 |
| Raycast JetBrains Toolbox（S3） | 跨 JetBrains IDE 的项目列表、收藏、应用筛选、使用频率与最近使用排序、分支、缺失 Toolbox/CLI 的明确诊断 | 借鉴项目与 IDE 关系、收藏和诊断。Toolbox shell scripts 与版本仍有配置约束 |
| Raycast VS Code Project Manager（S4） | 消费 Project Manager 项目；标签分组/过滤；应用选择；终端/Git 客户端/复制路径；远程打开分支 | 应作为可选来源复用用户已整理的项目，而不是要求再维护一份 |
| VS Code Project Manager（S5） | 收藏、别名/项目名、标签、仓库发现、项目 Profile；文档明确支持 Container/SSH/WSL/Codespaces 收藏 | 本地/远程应当都是完整项目对象。但它主要在 IDE 内，Alfred 可以提供更快的跨应用入口 |
| Alfred Open with VS Code（S7） | Finder 当前目录/选中项、手输路径、`codef` 文件搜索、Alfred 原生工作流对象 | 快速打开还包括“已有选中对象”的场景；不必每次都从历史列表找 |
| 当前 alfred.vsc | 模糊搜索、目录发现和逐层浏览、最近使用、分支、默认/备用 IDE | 轻量、可控，但多 IDE 与远程是配置拼接，数据一致性和首次使用体验不足 |

可形成优势的组合是：**一个搜索入口合并多个 IDE 的项目，保留 SSH/容器身份，记住每个项目该怎么打开，并保证结果快而稳定。** 这是产品目标，不是已证明超越竞品的结论。

## 6. 建议的交互

### 首次使用

`vsc` 自动发现已安装 IDE，展示最近项目。没有来源时显示“添加项目目录”“选择 IDE”“诊断”，而不是空白列表。常规用户不应手写 `/Applications/.../bin/code`。

IDE 列表以已安装应用为主，显示能力差异；自定义 CLI 保留为高级配置。Python 不可用时在可运行的启动层提供明确安装提示，或后续用独立二进制解决分发。

### 日常搜索

```text
vsc payments

★ payments       本地 · ~/Code/payments             Cursor
  payments-api   SSH: devbox · /srv/payments-api     VS Code
  payments       容器: payments-dev                 原始 IDE

Enter  使用该项目默认 IDE
Cmd+Enter  选择其他兼容 IDE / 打开方式
```

这是设计草图，未声称 Alfred 已支持图示布局。实际使用标题、副标题、图标与修饰键提示映射。次级动作应通过明确的动作列表或 Alfred 原生动作入口呈现，并在 macOS 检查快捷键冲突。

- 查询匹配名称、路径片段、主机别名、标签和用户别名；强名称匹配优先，再结合近期和频率；收藏可单独分组。
- 保留 `vsc /` 逐层浏览。全局项目搜索由索引完成；不要把每次 `/keyword` 变成无界磁盘扫描。
- 同名项目显示根目录/主机/容器身份；多根浏览要保留根 ID。
- “隐藏”“恢复隐藏”“从最近记录移除”“删除实际文件”是不同动作。首版只提供可恢复的工具内隐藏。
- 常用次级操作：新窗口/复用窗口、切换 IDE、记住该项目 IDE、固定、复制路径/URI、Finder、终端。
- SSH/容器结果不触发每按键一次远程连接；连接交给 IDE。离线项目不会因为一次访问失败被永久清除。
- 标记 `.code-workspace`，支持用户已有 Profile 的显式选择。各 IDE 的 Profile/远程能力逐项验证。

## 7. 推荐架构与技术栈

```text
IDE 历史 providers ─┐
项目管理器导入 ─────┼→ 规范化/来源合并 → 查询与排序 → Alfred JSON
目录发现 ─────────┘         ↑                       ↓
                  收藏/别名/隐藏/IDE偏好       打开意图 → editor adapter
                             ↑                       ↓
                    可丢弃的派生快照            IDE 自己更新历史
```

边界建议：

- `providers/`：按编辑器读取历史，显式报告来源状态；只读连接、短锁等待、失败隔离，支持经过验证的旧/新存储布局。
- `model`：分开保存项目身份、来源观察与打开偏好。项目至少包含 `kind`、规范 URI、原始 URI、remote authority、来源 IDE、label。相同本地目录可合并来源；远程同路径不同 host 必须不同；容器 authority 保持不透明。
- `query`：输入解析、纯函数匹配/排序；一次快照；不在每个结果中重新读全表。
- `actions`：`OpenIntent` 包含目标、IDE、窗口模式和可选 Profile；用参数数组执行，提供可诊断结果。
- `presentation`：集中构造 Alfred items、稳定 UID、修饰键提示和配置/错误状态，避免业务散落在 plist shell 中。
- `state`：用户偏好和派生状态分开。初期偏好小可用原子 JSON + 锁；多进程/多表需求明确后可用标准库 SQLite 事务。不要再建一份手动维护的“权威 IDE 历史库”。

后台刷新应先采用按需、短生命周期工作进程和单实例协调；必要时通过 Alfred 刷新机制补充结果。没有常驻服务的明确收益前，不增加常驻 daemon。

| 技术选项 | 优势 | 代价与建议 |
|---|---|---|
| Python 3 | 已有实现小；标准库有 SQLite/URI/子进程；这次实验表明可达到几十毫秒级轻量搜索 | 需要处理 Mac 解释器可用性。首选验证路线 |
| Swift CLI | 原生 macOS API、应用发现与分发体验可更好，无外部 Python 依赖 | 重建工具链、跨架构产物和测试；在分发成为核心瓶颈时值得做原型 |
| Go CLI | 单二进制、启动成本低、并发和文件扫描方便 | macOS 原生集成与 SQLite 库选型、CGO/纯 Go 权衡；适合单二进制优先目标 |
| TypeScript/Node | 便于复用 Raycast 生态与前端知识 | Alfred 仍需运行时/打包方案；换回 JS 本身不能解决重复 I/O。未来做 Raycast 前端时再评估 |

不建议为了重写而加入 Web UI、大型插件框架、云同步或后台连接管理。当前运行时约 1,195 行 Python，适合按边界逐步替换。若最终换语言，让 provider fixture、打开参数契约和基准保持同一套，保证对比可复现。

## 8. 分阶段执行与验收

成本用相对工作量表示：S 为单一边界改动，M 为多模块和回归验证，L 为跨编辑器/平台联调；不是工期承诺。

| 阶段 | 可交付结果 | 成本 | 验收门槛 |
|---|---|---|---|
| A：先恢复核心可靠性 | 一次快照读取；修复路径空格/失败 MRU；统一来源与隐藏过滤；清理作者路径和旧提示；关键回归 fixture | M | 本报告核心复现转为回归通过；旧 CLI/关键词兼容；宽泛查询不随结果数平方增长 |
| B：验证新数据路线 | Python 最小纵向版本：VS Code+Cursor 历史只读合并、来源诊断、可选短期派生缓存、macOS 性能采样 | M | 在真实 Mac 上达到首批响应预算；IDE 新历史自动出现；关闭缓存仍能正确工作；无手动重建依赖 |
| C：多 IDE 与远程产品化 | 项目默认 IDE；编辑器能力表；SSH/容器/workspace 打开；收藏、动作菜单、恢复隐藏；Finder 入口 | L | 同名本地/远程区分正确；支持的组合有实机记录；不支持组合有明确反馈；打开失败不伪装成功 |
| D：扩展与发布 | JetBrains/Project Manager providers；诊断导出；版本迁移；自动打包与 macOS 回归；按证据决定独立二进制 | M–L | 迁移保留偏好可回滚；包元数据一致；性能不退化；文档与实现一致 |

首个纵向版本应把“VS Code/Cursor 最新历史 → 搜索 → 正确打开本地/SSH 项目”跑通，再扩展容器和其他 IDE。容器是你的核心需求，必须在 C 阶段有明确真实验证，不能用识别出 URI 代替打开成功。

迁移建议：旧 `.records.cache.json` 不再直接当作权威历史导入；能够重新从 IDE 取得的记录重新读取。旧 trash 迁移为隐藏偏好，保留备份和 schema 版本；目录扫描项目通过配置重建。来源失败时保留上次有效结果，不能把一次空读当成全量删除。

质量保障重点：本地路径含空格/中文/百分号/井号；SSH authority 和容器不透明 URI；新旧来源 schema；历史合并与隐藏恢复；多个 IDE 的相同项目；锁竞争/WAL；worktree 相对 gitdir；目录排除和符号链接；包内资源完整性；失败退出码。Linux 覆盖纯逻辑和契约，macOS 覆盖 Alfred 导入、修饰键、应用发现、真实打开和安装依赖。

## 9. 本轮产物和限制

- [review_checks.py](/workspace/alfred-vsc-review/review_checks.py)：核心行为复现和当前性能曲线。
- [results.json](/workspace/alfred-vsc-review/results.json)：行为观察与原始汇总。
- [compare_cache_reads.py](/workspace/alfred-vsc-review/compare_cache_reads.py)、[paired_results.json](/workspace/alfred-vsc-review/paired_results.json)：同输入、同输出的重复读取诊断。
- [compare_history_sources.py](/workspace/alfred-vsc-review/compare_history_sources.py)、[history_source_results.json](/workspace/alfred-vsc-review/history_source_results.json)：SQLite/JSON 来源与完整子进程比较。

运行复现脚本会建立并清理临时 fixtures，写入本评审目录中的结果文件。脚本中的 true 多数表示缺陷已复现，不是产品检查通过；诊断替换只存在于测试子进程，没有修改运行时源码。

未运行真实 Alfred、VS Code、Cursor、JetBrains、SSH 或容器启动，也未测量竞品速度。尚需真实 Mac 的源码存储位置与性能验证。竞品官网访问受当前网络策略限制，采用公开 GitHub 原始文档与代码；没有为了调研修改网络设置。

## 10. 外部依据

- **S1 Raycast Visual Studio Code**：[manifest](https://github.com/raycast/extensions/blob/main/extensions/visual-studio-code-recent-projects/package.json)、[搜索与动作](https://github.com/raycast/extensions/blob/main/extensions/visual-studio-code-recent-projects/src/index.tsx)、[历史读取](https://github.com/raycast/extensions/blob/main/extensions/visual-studio-code-recent-projects/src/lib/db.ts)、[应用发现](https://github.com/raycast/extensions/blob/main/extensions/visual-studio-code-recent-projects/src/utils/editor.ts)、[Git 缓存](https://github.com/raycast/extensions/blob/main/extensions/visual-studio-code-recent-projects/src/utils/git.ts)。
- **S2 Raycast Cursor**：[manifest](https://github.com/raycast/extensions/blob/main/extensions/cursor-recent-projects/package.json)、[搜索与动作](https://github.com/raycast/extensions/blob/main/extensions/cursor-recent-projects/src/index.tsx)、[DB 访问](https://github.com/raycast/extensions/blob/main/extensions/cursor-recent-projects/src/db.ts)。
- **S3 Raycast JetBrains Toolbox**：[README](https://github.com/raycast/extensions/blob/main/extensions/jetbrains/README.md)、[项目列表](https://github.com/raycast/extensions/blob/main/extensions/jetbrains/src/recent.tsx)、[manifest](https://github.com/raycast/extensions/blob/main/extensions/jetbrains/package.json)。
- **S4 Raycast Project Manager**：[manifest](https://github.com/raycast/extensions/blob/main/extensions/visual-studio-code-project-manager/package.json)、[实现](https://github.com/raycast/extensions/blob/main/extensions/visual-studio-code-project-manager/src/search-project-manager-projects.tsx)。
- **S5 VS Code Project Manager**：[官方仓库 README](https://github.com/alefragnani/vscode-project-manager/blob/master/README.md)。
- **S6 VS Code 官方**：[远程 CLI](https://github.com/microsoft/vscode-docs/blob/main/docs/remote/troubleshooting.md#connect-to-a-remote-host-from-the-terminal)、[命令行与 Profile](https://github.com/microsoft/vscode-docs/blob/main/docs/configure/command-line.md)、[Electron 历史服务](https://github.com/microsoft/vscode/blob/main/src/vs/platform/workspaces/electron-main/workspacesHistoryMainService.ts)、[Browser 历史服务](https://github.com/microsoft/vscode/blob/main/src/vs/workbench/services/workspaces/browser/workspacesService.ts)。
- **S7 Alfred Open with VS Code**：[README](https://github.com/alexchantastic/alfred-open-with-vscode-workflow/blob/main/README.md)、[工作流配置](https://github.com/alexchantastic/alfred-open-with-vscode-workflow/blob/main/info.plist)。

来源文件保存在 `/workspace/alfred-vsc-review/sources/`，URL 与 SHA-256 清单见 `sources-manifest.json`。这些来源用于核实功能和接口，没有将第三方实现直接合并入产品。
