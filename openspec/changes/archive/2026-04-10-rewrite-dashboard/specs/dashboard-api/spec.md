## ADDED Requirements

### Requirement: API 响应使用一致的信封格式
所有 API 响应 SHALL 在成功时使用格式 `{ok: true, data: ...}`，失败时使用 `{ok: false, error: "message"}`。

#### Scenario: 成功的 GET 返回 ok:true
- **WHEN** 客户端调用 `/api/records`
- **THEN** 响应为 `{ok: true, data: {records: [...], total: N}}`

#### Scenario: 失败的请求返回 ok:false
- **WHEN** 客户端调用 `/api/records/invalid-id`
- **THEN** 响应为 `{ok: false, error: "Not found"}`，状态码 404

### Requirement: GET /api/records 返回所有记录
该端点 SHALL 返回 cache.json 中所有记录。

#### Scenario: 返回 records 数组中的所有记录
- **WHEN** 客户端调用 `GET /api/records`
- **THEN** 响应包含 `records` 数组，每项有 `__vsc_id__`、`name`、`path`、`icon`、`branch`、`display_path`

#### Scenario: 支持 limit 参数
- **WHEN** 客户端调用 `GET /api/records?limit=10`
- **THEN** 只返回 10 条记录

#### Scenario: 支持 search 参数
- **WHEN** 客户端调用 `GET /api/records?search=myproject`
- **THEN** 只返回名称匹配 "myproject" 的记录

### Requirement: DELETE /api/records/:id 移动到回收站
该端点 SHALL 将记录移动到回收站，而非永久删除。

#### Scenario: 记录移动到回收站
- **WHEN** 客户端调用 `DELETE /api/records/rec_xxx`
- **THEN** 记录从 `records` 数组移除，加入 `trash{}` 到缓存

### Requirement: POST /api/reorder 重新排序记录
该端点 SHALL 接受记录 ID 数组并更新缓存中的顺序。

#### Scenario: 记录重新排序
- **WHEN** 客户端发送 `POST /api/reorder`，body 为 `{ids: ["a", "b", "c"]}`
- **THEN** records 数组按提供的顺序重新排序

### Requirement: GET /api/trash 返回回收站项目
该端点 SHALL 返回回收站中所有记录，包含删除时间戳。

#### Scenario: 返回回收站项目
- **WHEN** 客户端调用 `GET /api/trash`
- **THEN** 响应包含 `trash{}` 对象，将记录 ID 映射到 `{path, deleted_at}`

### Requirement: POST /api/trash/:id/restore 从回收站恢复
该端点 SHALL 从回收站移除记录并添加回 records。

#### Scenario: 记录恢复
- **WHEN** 客户端调用 `POST /api/trash/rec_xxx/restore`
- **THEN** 记录从 trash 移除，添加回 records 数组

### Requirement: DELETE /api/trash/:id 永久删除
该端点 SHALL 从回收站永久移除记录。

#### Scenario: 永久删除
- **WHEN** 客户端调用 `DELETE /api/trash/rec_xxx`
- **THEN** 记录从缓存中永久移除

### Requirement: DELETE /api/trash 清空回收站
该端点 SHALL 清空整个回收站。

#### Scenario: 回收站清空
- **WHEN** 客户端调用 `DELETE /api/trash`
- **THEN** trash 对象被清空

### Requirement: POST /api/rebuild 触发 Alfred rebuild
该端点 SHALL 调用 `vscdli rebuild` 命令触发 Alfred workflow 重新构建索引。

#### Scenario: rebuild 触发
- **WHEN** 客户端调用 `POST /api/rebuild`
- **THEN** 系统调用 Alfred CLI 触发重建索引

### Requirement: GET /api/stats 返回管理面板统计
该端点 SHALL 返回总览页的计数和摘要数据。

#### Scenario: 返回统计
- **WHEN** 客户端调用 `GET /api/stats`
- **THEN** 响应包含 `total_records`、`total_trash`、`total_directories`、`total_git_repos`、`recent_opens`

### Requirement: GET /api/config 返回当前配置
该端点 SHALL 返回 VSC_DIRECTORIES 和其他设置。

#### Scenario: 返回配置
- **WHEN** 客户端调用 `GET /api/config`
- **THEN** 响应包含 `directories`（数组）和其他设置

### Requirement: PATCH /api/config 更新配置
该端点 SHALL 更新配置设置。

#### Scenario: 目录更新
- **WHEN** 客户端发送 `PATCH /api/config`，body 为 `{directories: ["/path1", "/path2"]}`
- **THEN** VSC_DIRECTORIES 被更新

### Requirement: GET /api/directories/tree 返回目录结构
该端点 SHALL 返回配置的 VSC_DIRECTORIES 的树形结构。

#### Scenario: 返回树结构
- **WHEN** 客户端调用 `GET /api/directories/tree`
- **THEN** 响应包含嵌套目录结构，带文件夹图标

### Requirement: 所有响应包含 CORS 头
所有 API 响应 SHALL 包含 `Access-Control-Allow-Origin: *` 头。

#### Scenario: CORS 头存在
- **WHEN** 客户端发起任何 API 请求
- **THEN** 响应包含 `Access-Control-Allow-Origin: *` 头
