# VSC Dashboard — 功能开发交接文档

> 编写时间：2026-04-09
> 负责人：OpenClaw Assistant（由 Harvey Z 发起）

---

## 一、项目概述

**VSC** 是一个 Alfred Workflow，用于快速搜索和打开 VS Code 项目。本次开发在原有基础上新增了**本地 Web 可视化管理面板**功能（`vscli serve` 命令）。

核心思路：用 `vscli serve` 启动一个本地 HTTP 服务器 + 单页 Web 应用，在浏览器中提供图形化管理界面，替代纯 CLI 操作。

---

## 二、代码改动清单

### 2.1 新增文件

| 文件 | 说明 |
|------|------|
| `src/server.py` | Python HTTP 服务器，提供 REST API 和静态文件服务 |
| `src/web/api.js` | 前端 API 客户端，封装所有 fetch 调用 |
| `src/web/app.js` | 前端 SPA 主逻辑（导航、渲染、事件处理） |
| `src/web/index.html` | HTML 入口（CSS/JS 已内联，打包时合并） |
| `src/web/styles.css` | 完整 CSS 样式 |

### 2.2 修改文件

| 文件 | 主要改动 |
|------|------|
| `src/cli.py` | 新增 `cmd_serve()`（后台启动）、`cmd_stop()`（停止服务）、`cmd_selected()` 增强（空缓存自动重建） |
| `src/shared.py` | 合入了 `feat/mack-some-new` 的目录浏览优化（只列直接子目录） |
| `src/vsc.py` | 小幅改动（与 shared.py 保持一致） |
| `scripts/build_workflow.py` | 新增 `server.py` 和 `web/` 文件到打包清单 |

---

## 三、Web 面板功能

### 3.1 五大模块

| 模块 | 功能 |
|------|------|
| 📊 总览 | 统计数字、最近打开快捷入口 |
| 📁 项目 | 卡片/表格双视图、搜索过滤、拖拽排序、Git 分支 Badge、批量删除 |
| 🗑️ 回收站 | 恢复/永久删除/清空，显示删除时间 |
| 📂 目录浏览 | 树形目录，点击展开，"添加为项目"按钮 |
| ⚙️ 设置 | VSC_DIRECTORIES 可视化编辑（增删目录）、导出/导入数据 |

### 3.2 服务端 API 路由

```
GET    /api/records          — 获取项目列表
DELETE /api/records/<id>      — 删除单个项目
POST   /api/records/reorder   — 拖拽排序
POST   /api/records/batch-delete — 批量删除
GET    /api/trash             — 获取回收站
POST   /api/trash/<id>/restore — 恢复记录
DELETE /api/trash/<id>        — 永久删除
DELETE /api/trash             — 清空回收站
POST   /api/trash/restore-all — 恢复全部
POST   /api/rebuild           — 重建索引
GET    /api/stats             — 统计数据
GET    /api/config            — 获取配置
PATCH  /api/config            — 更新配置
GET    /api/directories/tree  — 目录树
POST   /api/directories       — 添加目录
```

### 3.3 CLI 命令

```bash
vscli serve          # 启动服务（后台运行，写 .server.pid）
vscli serve --port 8080  # 指定端口
vscli serve --no-open    # 不自动打开浏览器
vscli stop           # 停止服务（读 .server.pid，杀进程）
```

---

## 四、当前状态

### 已完成 ✅
- Web 面板完整功能（总览/项目/回收站/目录/设置）
- 拖拽排序（卡片 + 表格）
- Git 分支主动查询（后端 eager lookup）
- Git 仓库计数（基于 .git 目录检测）
- 设置页可视化编辑（目录增删、导出/导入 JSON）
- CLI 启停命令 + PID 文件
- 构建脚本支持打包 server.py 和 web/ 文件
- 合入 `feat/mack-some-new` 最新代码

### 待处理 ⚠️
- 需要添加 `serve` 和 `stop` 的 Alfred 热键绑定（目前只有 keyword `vscdli serve/stop`）
- `info.plist` 里的热键配置需要补充
- 单元测试未覆盖（项目 README 注明"手动验证"）
- server.py HTTP 服务器绑定在 `0.0.0.0`（外网可访问，需注意安全）

---

## 五、测试方法

```bash
# 安装 dev 版 workflow（会自动创建独立 bundle ID）
cd /root/code/alfred.vsc
python3 scripts/dev_install.py

# 或直接运行（不需要 Alfred）
cd src
python3 server.py --no-open
# 浏览器打开 http://localhost:3847

# 测试 CLI
python3 cli.py serve --no-open
python3 cli.py stop
```

---

## 六、常见问题

**Q: 服务器无法从外网访问？**
A: 检查 AWS 安全组是否开放了对应端口（默认 3847）

**Q: 拖拽排序后刷新数据丢失？**
A: `POST /api/records/reorder` 会直接写 cache.json，Alfred 下次搜索时会读到最新顺序

**Q: Git 分支显示"—"？**
A: 后端会主动查询 `.git/HEAD`，如果没有 `.git` 目录则显示"—"，这是预期行为
