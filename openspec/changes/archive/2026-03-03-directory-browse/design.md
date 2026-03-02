## Context

当前 `vsc` workflow 的搜索流程是：`vscli rebuild` 预先扫描 `VSC_DIRECTORIES` → 结果写入 cache → `vsc` keyword 从 cache 读取并 fuzzy match。这种模式对已知项目工作良好，但无法浏览未被缓存的目录。

用户的 `~/Code` 目录实际规模：顶层 12 个目录，递归 5 层（过滤 `node_modules` 等后）约 137 个目录。Python `pathlib.iterdir()` 扫描 depth 1 耗时 ~0.1ms，递归 depth 5 耗时 ~10ms，性能完全满足实时扫描的需求。

## Goals / Non-Goals

**Goals:**
- 通过 `/` 前缀在 `vsc` keyword 中触发目录浏览模式
- 支持层级钻入（`/path/`）和模糊搜索（`/keyword`）两种交互模式
- 实时扫描，不依赖缓存，始终反映文件系统最新状态
- 选中后的动作与现有项目列表完全一致

**Non-Goals:**
- 不修改 `info.plist`（复用现有 keyword 和动作连接）
- 不新增 Alfred keyword 或 trigger
- 不引入外部依赖
- 不缓存目录浏览结果
- 不支持浏览文件（仅目录）

## Decisions

### Decision 1: 在 `vsc.py` 中内联实现，而非新建入口

**选择**：在现有 `vsc.py` 的 `main()` 中检测 `/` 前缀并分流到目录浏览逻辑。

**理由**：
- 复用现有的 Alfred Script Filter 和所有动作连接（open、delete、selected）
- 无需修改 `info.plist`
- 用户无需记忆新的 keyword

**替代方案**：新建独立的 Script Filter（如 `vscdir` keyword）。但会增加 `info.plist` 复杂度，且需要复制所有动作连接。

### Decision 2: 查询解析策略

**选择**：基于 `/` 分割的 segments + search 解析模型。

```
输入              segments        search    行为
─────────────   ──────────      ────────  ─────────────────
/               []              ""        列顶层子目录
/alf            []              "alf"     递归搜索 (depth≤5)
/alfred/        ["alfred"]      ""        钻入，列子目录
/alfred/vs      ["alfred"]      "vs"      钻入 + 搜索
/a/b/c/         ["a","b","c"]   ""        深层钻入（无限制）
/a/b/c/foo      ["a","b","c"]   "foo"     深层钻入 + 搜索
```

**关键逻辑**：query 以 `/` 结尾 → 层级模式（列子目录）；不以 `/` 结尾且有 segments → 在 segments 目录下搜索；无 segments → 全局递归搜索。

### Decision 3: 目录定位策略（segments 到实际路径的映射）

**选择**：在所有 `VSC_DIRECTORIES` 的根目录下，按 segments 路径逐级匹配。第一个 segment 在所有根的直接子目录中匹配，后续 segment 在已定位目录的子目录中继续匹配。

**示例**：`VSC_DIRECTORIES=~/Code,~/Projects`，输入 `/alfred/`
1. 在 `~/Code` 和 `~/Projects` 各自的子目录中找 "alfred"
2. 找到 `~/Code/alfred` → 列其子目录

### Decision 4: 实时扫描而非缓存

**选择**：每次用户输入都实时扫描文件系统。

**理由**：
- 实测性能：depth 1 ~0.1ms，depth 5 ~10ms，远小于 Alfred 的 UI 刷新间隔
- 目录浏览的核心价值在于发现新目录，缓存会破坏这个价值
- 减少代码复杂度（无需缓存失效策略）

### Decision 5: 过滤策略

**选择**：维护 `SKIP_DIRS` 集合，硬编码跳过常见的非项目目录。

```python
SKIP_DIRS = {
    'node_modules', '__pycache__', '.venv', 'venv',
    'dist', 'build', 'vendor', 'target', 'Pods',
    '.git', '.svn', '.hg',
}
```

**理由**：这些目录几乎不会是用户想通过 IDE 打开的项目。跳过后：
- 递归扫描的目录数从 423 降到 137（减少 67%）
- 结果更干净，用户无需在噪音中筛选

### Decision 6: 代码组织

**选择**：
- `shared.py`：新增 `SKIP_DIRS` 常量 + `scan_subdirectories(root, depth)` 实时扫描函数
- `vsc.py`：新增 `parse_browse_query(query)` 解析 + `browse_directories(query)` 主逻辑 + 修改 `main()` 分流

逻辑放在 `vsc.py` 是因为它是搜索端的入口，而扫描函数放 `shared.py` 因为它是通用的目录操作能力。

## Risks / Trade-offs

- **[符号链接循环]** → 使用 `pathlib.iterdir()` + `is_dir()` 默认会跟随 symlink，可能导致无限循环。缓解：递归扫描时设置 max_depth=5 + 记录已访问路径去重。
- **[SKIP_DIRS 硬编码]** → 用户可能有不同的过滤需求。缓解：当前阶段硬编码即可。如果用户反馈强烈，未来可增加 `VSC_SKIP_DIRS` 环境变量。
- **[多根目录名称冲突]** → 多个 `VSC_DIRECTORIES` 根下可能有同名子目录（如两个 `~/Code/utils` 和 `~/Projects/utils`）。缓解：在 subtitle 中显示完整路径以区分。
