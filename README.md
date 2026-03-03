# VSC Alfred Workflow

<p align="center">
  <img src="public/logo.svg" alt="VSC Logo" width="128" height="128">
</p>

<p align="center">
  <strong>快速搜索和打开代码项目的 Alfred Workflow</strong>
</p>

## ✨ 特性

- 🔍 **智能搜索** — 支持模糊匹配，快速定位项目
- 📂 **目录浏览** — 输入 `/` 实时浏览项目目录，支持层级钻入和模糊搜索
- 📂 **自动发现** — 递归扫描目录，自动发现 Git 仓库
- ⚡ **最近使用** — 记住使用顺序，常用项目优先显示
- 🌿 **Git 分支** — 显示当前分支名称
- 🛠️ **开发友好** — 支持隔离的开发调试模式

## 📦 安装

### 方式一：下载安装
1. 下载 [最新版本](https://github.com/HarveyZgit/alfred.vsc/releases) 的 `.alfredworkflow` 文件
2. 双击安装到 Alfred

### 方式二：从源码构建
```bash
git clone https://github.com/HarveyZgit/alfred.vsc.git
cd alfred.vsc
python3 scripts/build_workflow.py
# 输出: build/vsc.alfredworkflow
```

## 🚀 使用方式

| 关键字 | 功能 |
|-------|------|
| `vsc` | 搜索项目 |
| `vsc <关键字>` | 模糊搜索项目 |
| `vsc d <关键字>` | 只搜索文件夹类型 |
| `vsc /` | 浏览目录（详见下方） |
| `vscli` | 管理命令 |
| `vscli rebuild index` | 重建索引 |

**快捷操作**：
- `Enter` — 用默认 IDE 打开
- `Cmd + Enter` — 用备选 IDE 打开
- `Ctrl + Enter` — 从列表中删除
- `Tab` — 在浏览模式下钻入子目录

## ⚙️ 配置

在 Alfred Workflow 设置中配置环境变量：

| 变量 | 说明 | 示例 |
|-----|------|------|
| `VSC_DIRECTORIES` | 项目目录列表（逗号分隔） | `~/Code,~/Projects` |
| `VSC_OPEN_DEFAULT` | 默认打开方式（Enter 键） | `/usr/local/bin/code` |
| `VSC_OPEN_WITH_CMD` | 按住 Command 打开 | `/Applications/Cursor.app/Contents/MacOS/Cursor` |
| `VSC_TAB_TAIL_SLASH` | Tab 补全是否追加 `/`（默认 `1`） | `0` 或 `1` |

## 🔍 搜索功能

### 模糊匹配
字符按顺序出现即可匹配：

| 搜索词 | 匹配示例 |
|-------|---------|
| `myp` | **my**-**p**roject |
| `api` | backend-**api** |
| `fe` | frontend, core-**fe** |

### 路径类型过滤
| 类型 | 别名 |
|-----|------|
| folder | `folder`, `fo`, `ff`, `dir`, `d` |
| file | `file`, `fi`, `f` |
| remote | `remote`, `re`, `r` |

### Git 分支显示
```
my-project
⎇ main | ~/Code/my-project
```

### 📁 目录浏览模式

输入 `/` 进入目录浏览模式，实时扫描 `VSC_DIRECTORIES` 下的目录：

| 输入 | 行为 |
|------|------|
| `vsc /` | 列出所有顶层子目录 |
| `vsc /alf` | 递归模糊搜索（最深 5 层） |
| `vsc /alfred/` | 钻入 alfred，列出子目录 |
| `vsc /alfred/vs` | 在 alfred 下搜索 "vs" |

- 按 **Tab** 可快速钻入选中的目录
- 到达叶子目录时会提示「没有更多子目录了」，按 Tab 返回上级
- 自动跳过 `node_modules`、`__pycache__`、`venv`、`dist`、`build` 等非项目目录

> ⚠️ **Tab 键打开了类似 Finder 的面板？** 需要修改 Alfred 设置：Alfred Preferences → Features → Universal Actions → 将 "Show Actions" 的快捷键从 Tab 改为其他键（如 `→`）。

## 🛠️ 开发

### 项目结构
```
├── src/
│   ├── shared.py        # 共享工具、配置、数据源
│   ├── vsc.py           # 主搜索脚本
│   ├── cli.py           # CLI 命令处理
│   └── assets/          # 图标资源
├── public/
│   ├── info.plist       # Alfred 工作流配置
│   ├── icon.png         # 工作流图标
│   ├── logo.svg         # Logo (SVG)
│   └── logo-dev.svg     # 开发版 Logo
├── scripts/
│   ├── build_workflow.py   # 构建 .alfredworkflow
│   └── dev_install.py      # 安装开发版
└── README.md
```

### 本地开发

安装开发版（与正式版完全隔离）：

```bash
python3 scripts/dev_install.py
```

开发版特性：
- **关键词**: `vscd` / `vscdli`（不与正式版冲突）
- **缓存目录**: `/tmp/vsc-dev/`（独立缓存）
- **符号链接**: 改代码即时生效

```bash
# 查看开发版状态
python3 scripts/dev_install.py --status

# 卸载开发版
python3 scripts/dev_install.py --remove
```

### 构建发布

```bash
# 构建 .alfredworkflow
python3 scripts/build_workflow.py

# 预览构建内容
python3 scripts/build_workflow.py --dry-run
```

## 📄 License

MIT © Harvey Zhang
