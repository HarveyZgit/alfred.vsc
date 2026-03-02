## Why

当前 `vsc` workflow 只能搜索已缓存的项目（VSCode 最近打开记录 + `VSC_DIRECTORIES` 下已发现的 git 仓库和直接子目录）。用户需要先 `vscli rebuild` 才能看到新目录，且无法浏览未被缓存收录的深层子目录。

用户希望能通过输入 `/` 前缀，实时浏览和搜索 `VSC_DIRECTORIES` 配置的目录树，获得类似文件管理器的层级导航体验，同时支持模糊搜索快速定位目标目录。

## What Changes

- **新增目录浏览模式**：在 `vsc` keyword 中，输入 `/` 前缀进入目录浏览模式，实时扫描 `VSC_DIRECTORIES` 下的子目录
- **支持层级钻入**：通过 `/path/` 语法逐级进入子目录（无深度限制）
- **支持模糊搜索**：输入 `/keyword` 在 `VSC_DIRECTORIES` 下递归搜索（depth ≤ 5），输入 `/path/keyword` 在指定目录下搜索
- **智能过滤**：自动跳过 `node_modules`、`__pycache__`、`.venv` 等非项目目录
- 选中目录后的动作与现有项目列表完全一致（Enter 打开、Cmd+Enter 用备选 IDE 打开）

## Capabilities

### New Capabilities
- `directory-browse`: 通过 `/` 前缀触发的实时目录浏览与搜索能力，包含层级钻入和模糊搜索两种交互模式

### Modified Capabilities
_(无已有 capability 需要修改)_

## Impact

- **`src/vsc.py`**：修改 `parse_input()` 识别 `/` 前缀；`main()` 增加分流逻辑；新增目录浏览相关函数
- **`src/shared.py`**：新增实时目录扫描函数和 `SKIP_DIRS` 常量
- **`public/info.plist`**：无需修改（复用现有 `vsc` keyword 和动作连接）
- **无新增依赖**：仅使用 Python 标准库 `pathlib` / `os`
