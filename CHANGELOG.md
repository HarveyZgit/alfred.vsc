# VSC CHANGELOG

## v2.0.0 (2026-01-18)

### 🎉 重大更新：Python 重构版

这是一个完全重写的大版本更新，从 Node.js 迁移到 Python，带来更好的性能和开发体验。

### ✨ 新功能

- 🐍 **Python 重构**：核心代码从 Node.js 完全迁移到 Python 3，无需额外依赖
- 🔍 **模糊搜索**：支持智能模糊匹配，字符按顺序出现即可匹配
- 🌿 **Git 分支显示**：在搜索结果中显示每个项目的当前 Git 分支
- 📂 **递归发现 Git 仓库**：从配置目录递归查找所有 Git 仓库（最大深度 5 层）
- ⚡ **最近使用排序**：选中的项目会移到列表顶部，下次优先显示
- 🎨 **全新 Logo**：设计了全新的 SVG Logo，包含代码括号、搜索和闪电元素

### 🛠️ 开发工具

- 📦 **构建脚本** (`scripts/build_workflow.py`)：一键打包 `.alfredworkflow` 文件
- 🔧 **开发模式** (`scripts/dev_install.py`)：安装隔离的开发版 workflow
  - 独立关键词：`vscd` / `vscdli`
  - 独立缓存目录：`/tmp/vsc-dev/`
  - 符号链接源码，改代码即时生效
  - 无快捷键冲突

### 🏗️ 架构改进

- 缓存策略优化：Git 分支信息在初始搜索时更新，后续使用缓存
- 使用 `.git/HEAD` 文件读取分支（比 `git branch` 快 10-20 倍）
- 项目使用相对路径命名，便于区分同名项目
- 支持开发模式环境变量 `VSC_DEV_MODE` 和 `VSC_DEV_CACHE_DIR`

### 📁 项目结构

```
├── src/                    # 源码目录
│   ├── vsc.py              # 主搜索脚本
│   ├── cli.py              # CLI 命令处理
│   └── assets/             # 图标资源
├── public/                 # 公共资源
│   ├── info.plist          # Alfred 工作流配置
│   ├── icon.png            # 工作流图标
│   ├── logo.svg            # Logo (正式版)
│   └── logo-dev.svg        # Logo (开发版)
├── scripts/                # 脚本工具
│   ├── build_workflow.py   # 构建脚本
│   └── dev_install.py      # 开发版安装脚本
└── README.md
```

---

## v1.3.0 (2022-08-20) [#12](https://github.com/HarveyZgit/alfred.vsc/pull/12)

### vsc

- 🎉 支持在搜索时按照路径类型对结果进行过滤，了解[更多信息](https://github.com/HarveyZgit/alfred.vsc/blob/feat/1.3.0/README.md#filter-by-path-type)

## v1.2.0 (2022-08-20) [#7](https://github.com/HarveyZgit/alfred.vsc/pull/7)

### vsc

- 🎉 在删除是会保存当前删除的记录，便于 rebuild index 时过滤这一部分

### vscli

- 🎉 vscli 的相关操作会进行通知
- 🎉 新增 completely rebuild index，在 rebuild index 时会忽略所有缓存记录
- 🍀 rebuild index 时会将曾经删除的记录过滤掉
- 🙏 thx alfy and bey ~

### common

- 🍀 VSC_DIRECTORIES support `~` alias

## v1.1.1 (2022-07-16) [#4](https://github.com/HarveyZgit/alfred.vsc/pull/4)

- 🎉 没有关键词的时候展示 vscode 菜单中的最近打开列表
- 🎉 在 `vsc` 进行查找时，支持按住 `command` 键来使用 `vscode-insiders` 打开目录（需要配置环境变量 `VSC_CODE_INSIDERS_BIN`）
- 🍀 优化构建脚本、构建流程

## v1.1.0 (2022-07-10) [#3](https://github.com/HarveyZgit/alfred.vsc/pull/3)

- 🎉 新增 alfred 命令：`vscli` ，用于操作 vsc 的 record list。目前支持 `手动重建索引`
- 🎉 新增环境变量 `VSC_DIRECTORIES`，支持用户将某个目录下的子文件夹加入到 records 里面（多个目录时用英文逗号分隔）
- 🎉 在 `vsc` 进行查找时，支持按住 `control` 键来删除所选项
- 🍀 解耦更新和查找的逻辑，搜索的速度更快了
- 🗑 移除 v0.0.3 新增的特性，默认找 vscode db 中最近的 100 条

## v0.0.3 (2022-05-13) [#2](https://github.com/HarveyZgit/alfred.vsc/pull/2)

- 🎉 支持查找最新 xx 条记录，默认为最近 50 条

## v0.0.2 (2022-04-26) [#1](https://github.com/HarveyZgit/alfred.vsc/pull/1)

- 🎉 支持环境变量配置 & cache db instance

## v0.0.1 (2022-04-26)

- 🎉 A new alfred workflow
