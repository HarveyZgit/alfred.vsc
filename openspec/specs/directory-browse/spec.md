## ADDED Requirements

### Requirement: Browse mode activation
当用户在 `vsc` keyword 中输入以 `/` 开头的 query 时，系统 SHALL 进入目录浏览模式，不走现有的缓存搜索逻辑。

#### Scenario: Entering browse mode
- **WHEN** 用户输入 `vsc /`
- **THEN** 系统进入目录浏览模式，列出所有 `VSC_DIRECTORIES` 的直接子目录

#### Scenario: Normal search unaffected
- **WHEN** 用户输入 `vsc myproject`（不以 `/` 开头）
- **THEN** 系统走现有缓存搜索逻辑，行为不变

### Requirement: List top-level directories
当 query 仅为 `/` 时，系统 SHALL 列出所有 `VSC_DIRECTORIES` 根目录的直接子目录（depth 1），跳过隐藏目录和 `SKIP_DIRS` 中的目录。

#### Scenario: Single root directory
- **WHEN** `VSC_DIRECTORIES=~/Code` 且 `~/Code` 下有子目录 `alfred/`, `work/`, `.hidden/`, `node_modules/`
- **THEN** 结果列表包含 `alfred` 和 `work`，不包含 `.hidden` 和 `node_modules`

#### Scenario: Multiple root directories
- **WHEN** `VSC_DIRECTORIES=~/Code,~/Projects`
- **THEN** 结果列表合并两个根目录下的所有直接子目录（去重）

### Requirement: Flat recursive search
当 query 为 `/keyword`（以 `/` 开头，不以 `/` 结尾，无中间 `/`）时，系统 SHALL 递归扫描所有 `VSC_DIRECTORIES`（depth ≤ 5），对结果使用现有 `fuzzy_match()` 匹配 keyword。

#### Scenario: Recursive fuzzy search
- **WHEN** 用户输入 `vsc /alf`
- **THEN** 系统扫描 `VSC_DIRECTORIES` 下所有目录（depth ≤ 5），fuzzy match "alf"
- **THEN** 匹配结果按 fuzzy match score 降序排列

#### Scenario: Search respects skip dirs
- **WHEN** 用户输入 `vsc /mod`
- **THEN** 结果不包含任何 `node_modules` 内部的子目录

### Requirement: Hierarchical drill-down
当 query 包含中间 `/`（如 `/alfred/` 或 `/alfred/vsc/`）时，系统 SHALL 定位到 segments 指定的目录路径，然后列出该目录的直接子目录。

#### Scenario: Single level drill-down
- **WHEN** 用户输入 `vsc /alfred/`
- **THEN** 系统在 `VSC_DIRECTORIES` 的所有根下查找名为 "alfred" 的直接子目录
- **THEN** 列出找到的 "alfred" 目录下的所有直接子目录

#### Scenario: Multi-level drill-down
- **WHEN** 用户输入 `vsc /alfred/vsc/`
- **THEN** 系统定位到 `alfred/vsc` 目录，列出其直接子目录

#### Scenario: Drill-down beyond depth limit
- **WHEN** 用户输入 `vsc /a/b/c/d/e/f/g/`（depth > 5）
- **THEN** 系统正常列出该目录的子目录，无深度限制

### Requirement: Drill-down with search
当 query 为 `/path/keyword`（以 `/` 开头，不以 `/` 结尾，有中间 `/`）时，系统 SHALL 定位到 path 指定的目录，在其直接子目录中 fuzzy match keyword。

#### Scenario: Search within drilled-down directory
- **WHEN** 用户输入 `vsc /alfred/vs`
- **THEN** 系统定位到 "alfred" 目录，在其直接子目录中 fuzzy match "vs"
- **THEN** 结果包含匹配的子目录（如 `vsc/`）

### Requirement: Result item format
每个目录浏览结果 SHALL 遵循 Alfred Script Filter JSON 格式，包含 title、subtitle、arg 和 icon。

#### Scenario: Result item structure
- **WHEN** 浏览结果包含目录 `~/Code/alfred/vsc`
- **THEN** 该 item 的 title 为目录名（如 `vsc`）
- **THEN** subtitle 为完整路径（如 `~/Code/alfred/vsc`）
- **THEN** arg 为 `file:///Users/harvey/Code/alfred/vsc`
- **THEN** icon 使用 `folder.png`

### Requirement: Selection actions
目录浏览模式下选中目录后的动作 SHALL 与现有缓存列表的动作完全一致。

#### Scenario: Open with default IDE
- **WHEN** 用户在目录浏览结果中选中一个目录并按 Enter
- **THEN** 使用 `VSC_OPEN_DEFAULT` 打开该目录

#### Scenario: Open with alternative IDE
- **WHEN** 用户在目录浏览结果中选中一个目录并按 Cmd+Enter
- **THEN** 使用 `VSC_OPEN_WITH_CMD` 打开该目录

### Requirement: Directory filtering
系统 SHALL 在扫描时跳过以下类型的目录：
- 隐藏目录（以 `.` 开头的目录名）
- `SKIP_DIRS` 集合中的目录（`node_modules`, `__pycache__`, `venv`, `.venv`, `dist`, `build`, `vendor`, `target`, `Pods`）

#### Scenario: Hidden directories excluded
- **WHEN** 扫描 `~/Code` 且存在 `.git/`, `.vscode/` 子目录
- **THEN** 结果不包含这些隐藏目录

#### Scenario: Skip dirs excluded from recursion
- **WHEN** 递归扫描且存在 `project/node_modules/some-package/`
- **THEN** `node_modules` 及其子目录均不出现在结果中，也不被递归进入

### Requirement: Empty VSC_DIRECTORIES handling
当 `VSC_DIRECTORIES` 未配置或为空时，目录浏览模式 SHALL 返回空结果列表。

#### Scenario: No directories configured
- **WHEN** `VSC_DIRECTORIES` 环境变量为空
- **THEN** 输入 `vsc /` 时返回空的 Alfred items 列表
