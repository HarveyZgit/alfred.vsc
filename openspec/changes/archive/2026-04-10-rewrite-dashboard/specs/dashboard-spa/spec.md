## ADDED Requirements

### Requirement: SPA 正确提供静态文件服务
系统 SHALL 从 `/web` 目录提供单页应用，正确处理 HTML、CSS、JS 和图片资源，并设置正确的 MIME 类型。

#### Scenario: 根路径返回 index.html
- **WHEN** 用户访问 `http://localhost:3847/`
- **THEN** 系统返回 `index.html`，`Content-Type: text/html`

#### Scenario: CSS 文件以正确 MIME 类型提供服务
- **WHEN** 用户请求 `/styles.css`
- **THEN** 系统返回 CSS 文件，`Content-Type: text/css`

#### Scenario: JS 文件以正确 MIME 类型提供服务
- **WHEN** 用户请求 `/app.js`
- **THEN** 系统返回 JS 文件，`Content-Type: application/javascript`

#### Scenario: 缺失的静态文件返回 404
- **WHEN** 用户请求 `/nonexistent.file`
- **THEN** 系统返回 404 响应，正文为 "Not found"

### Requirement: 管理面板有 5 个导航标签页
SPA SHALL 提供 5 个标签页之间的导航：总览、项目、回收站、目录、设置。

#### Scenario: 标签导航更新内容区域
- **WHEN** 用户点击标签
- **THEN** 内容区域更新显示该标签内容，无页面刷新

#### Scenario: 当前标签视觉高亮
- **WHEN** 用户在某个标签上
- **THEN** 该标签在 UI 中显示为选中/激活状态

### Requirement: 总览页显示统计信息
总览页 SHALL 显示：项目总数、回收站数量、目录数量、Git 仓库数量，以及最近打开的快捷入口。

#### Scenario: 标签激活时获取统计
- **WHEN** 用户点击总览标签
- **THEN** 系统获取 `/api/stats` 并显示数字

#### Scenario: 最近打开列表显示最近 5 个项目
- **WHEN** 总览页加载
- **THEN** 系统显示最多 5 个最近打开的项目，包含名称和最后打开时间

### Requirement: 项目页支持列表和卡片视图
项目页 SHALL 支持在卡片视图和表格/列表视图之间切换。

#### Scenario: 卡片和表格视图切换
- **WHEN** 用户点击视图切换按钮
- **THEN** 项目显示在卡片网格和表格布局之间切换

#### Scenario: 搜索按名称过滤项目
- **WHEN** 用户在搜索框输入
- **THEN** 项目列表过滤，只显示匹配名称的项目

### Requirement: 项目页显示 Git 分支徽章
每个项目卡片/行 SHALL 如果项目是 git 仓库则显示 git 分支名称。

#### Scenario: 项目卡片显示 Git 分支
- **WHEN** 项目有 git 仓库
- **THEN** 卡片显示分支徽章，包含分支名称（如 "⎇ main"）

### Requirement: 回收站页显示已删除项目
回收站页 SHALL 列出所有已删除的项目，包含删除时间戳，并允许恢复或永久删除。

#### Scenario: 回收站显示删除时间
- **WHEN** 用户打开回收站标签
- **THEN** 每个项目显示删除时间（如 "2 小时前"）

#### Scenario: 从回收站恢复项目
- **WHEN** 用户点击恢复
- **THEN** 项目返回到项目列表

#### Scenario: 从回收站永久删除
- **WHEN** 用户点击永久删除
- **THEN** 项目从缓存中永久移除

### Requirement: 目录页显示树形浏览器
目录页 SHALL 显示配置的 VSC_DIRECTORIES 的树形视图，支持一键"添加为项目"。

#### Scenario: 树显示配置的目录
- **WHEN** 用户打开目录标签
- **THEN** 系统获取 `/api/directories/tree` 并显示可展开的树

#### Scenario: 添加目录为项目
- **WHEN** 用户点击树中目录的"添加"
- **THEN** 系统将该目录添加到项目

### Requirement: 设置页允许配置编辑
设置页 SHALL 允许编辑 VSC_DIRECTORIES、重建索引，以及导出/导入数据。

#### Scenario: 编辑 VSC_DIRECTORIES
- **WHEN** 用户修改目录文本框并保存
- **THEN** 系统调用 `PATCH /api/config`，传入新的目录

#### Scenario: 从设置页重建索引
- **WHEN** 用户点击"重建索引"
- **THEN** 系统调用 `POST /api/rebuild`

#### Scenario: 导出数据
- **WHEN** 用户点击"导出"
- **THEN** 系统下载 `records.json` 文件

### Requirement: SPA 轮询更新数据
SPA SHALL 每 30 秒轮询服务器，保持数据新鲜。

#### Scenario: 定时自动刷新数据
- **WHEN** 距离上次获取数据已过去 30 秒
- **THEN** SPA 从服务器获取最新数据
