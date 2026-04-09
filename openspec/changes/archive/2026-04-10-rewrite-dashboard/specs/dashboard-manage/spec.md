## ADDED Requirements

### Requirement: manage start 命令启动 HTTP 服务器
`manage start` 命令 SHALL 在指定端口启动 Web 管理面板 HTTP 服务器，并自动打开浏览器。

#### Scenario: 默认端口 3847
- **WHEN** 用户运行 `vscli manage start`
- **THEN** 服务器在 3847 端口启动

#### Scenario: 服务器已运行时只打开浏览器
- **WHEN** 服务器已在运行，用户运行 `vscli manage start`
- **THEN** 服务器不重启，只打开浏览器到 http://localhost:3847

#### Scenario: manage start 写入 PID 文件
- **WHEN** 服务器启动
- **THEN** PID 文件被创建在 `CACHE_DIR/.server.pid`

### Requirement: manage stop 命令停止服务器
`manage stop` 命令 SHALL 停止由 `manage start` 启动的 HTTP 服务器。

#### Scenario: 服务器成功停止
- **WHEN** 服务器正在运行，用户运行 `vscli manage stop`
- **THEN** 服务器进程被终止，PID 文件被删除

#### Scenario: 服务器未运行时无操作
- **WHEN** 没有服务器运行，用户运行 `vscli manage stop`
- **THEN** 命令正常退出，不报错

### Requirement: manage 命令可通过 Alfred 访问
`manage start` 和 `manage stop` 命令 SHALL 可通过 Alfred 中 `vscli` 关键字访问。

#### Scenario: Alfred 中可见 start 选项
- **WHEN** 用户在 Alfred 中输入 `vscli`
- **THEN** "🚀 启动管理面板" 选项出现在列表中

#### Scenario: Alfred 中可见 stop 选项
- **WHEN** 用户在 Alfred 中输入 `vscli`
- **THEN** "⏹ 停止管理面板" 选项出现在列表中

### Requirement: 端口冲突自动顺延
如果端口被占用，服务器 SHALL 尝试后续端口直到找到可用端口。

#### Scenario: 端口 3847 被占用
- **WHEN** 端口 3847 正在使用
- **THEN** 服务器尝试 3848、3849 等，直到找到可用端口

### Requirement: 浏览器自动打开
`manage start` SHALL 在服务器启动后自动在默认浏览器中打开管理面板。

#### Scenario: 浏览器自动打开
- **WHEN** 用户运行 `vscli manage start`
- **THEN** 系统在默认浏览器中打开 http://localhost:3847
