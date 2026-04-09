## 为什么需要这个改变

当前 Web 管理面板存在结构性缺陷——缺失函数、文件路径错误、Alfred 集成不完整。管理面板虽然存在但无法正常工作。这次重写将建立可视化管理工作面板的坚实基础，参考 wanted.md 中的设计规范。

## 改动内容

- 重写 `server.py`，正确提供 SPA 服务，包含正确的路由和静态文件处理
- 修复 `shared.py` 中缺失的函数（`get_git_branch_for_browse`、`update_branches_cache`）
- 修复 `dev_install.py`，确保所有必需文件都被符号链接（`server.py`、`web/`）
- 修改命令结构：`vscli manage start` 启动服务，`vscli manage stop` 关闭服务
- 管理面板仅读写 cache.json 文件，不直接访问 VSCode DB
- 实现正确的 CORS 和 API 响应格式一致性

## 功能模块

### 新增功能模块
- `dashboard-spa`：单页 Web 应用，用于管理 VSC cache.json 记录，包含：
  - 总览页：统计数字（项目总数/回收站/目录数/Git 仓库）、最近打开快捷入口
  - 项目页：卡片 + 表格双视图、搜索过滤、批量删除、拖拽排序、Git 分支 Badge
  - 回收站页：恢复/永久删除/清空，显示删除时间
  - 目录页：可视化树形目录，任意目录一键"添加为项目"
  - 设置页：可视化编辑 VSC_DIRECTORIES、重建索引、导出/导入数据
- `dashboard-api`：RESTful API 端点，用于 cache.json 的增删改查操作
- `dashboard-manage`：CLI 命令 `vscli manage start/stop` 管理 HTTP 服务器

### 修改功能模块
无

## 命令设计

- `vscli manage start` - 启动 HTTP 服务器（默认 3847 端口），自动打开浏览器；如果服务已启动则直接打开浏览器
- `vscli manage stop` - 停止 HTTP 服务器

## 影响范围

- **修改的文件**：`server.py`、`shared.py`、`cli.py`、`dev_install.py`、`info.plist`
- **新增文件**：`src/web/` 目录，包含 SPA 资源（index.html、app.js、styles.css）
- **API 变更**：所有 API 响应使用 `{ok: bool, data: ...}` 格式
- **Alfred 集成**：`vscli` 关键字现在包含 `manage start` 和 `manage stop` 选项
- **数据源**：管理面板只操作 cache.json，不直接读取 VSCode 任何数据
