## MODIFIED Requirements

### Requirement: rebuild 命令由 Alfred workflow 内部处理
`rebuild` 命令 SHALL 由 Alfred workflow 在内部处理，从配置的 VSC_DIRECTORIES 和 VSCode DB 获取数据写入 cache.json。管理面板通过调用 `POST /api/rebuild` 触发此过程。

#### Scenario: rebuild 命令执行时从目录收集
- **WHEN** `vscdli rebuild` 被 Alfred workflow 执行
- **THEN** 系统扫描 VSC_DIRECTORIES 中的 git 仓库并写入缓存

#### Scenario: rebuild 命令执行时从 VSCode DB 收集
- **WHEN** `vscdli rebuild` 被 Alfred workflow 执行
- **THEN** 系统从 VSCode 的 `state.vscdb` 获取记录并写入缓存

#### Scenario: 回收站被保留
- **WHEN** 用户运行 `rebuild`（不带 --drop-all）
- **THEN** 现有回收站记录被保留

#### Scenario: --drop-all 清空回收站
- **WHEN** 用户运行 `vscdli rebuild --drop-all`
- **THEN** 在重建前回收站被清空

### Requirement: rebuild 命令可通过 Alfred 访问
rebuild 命令 SHALL 可通过 Alfred 中 `vscdli` 关键字访问。

#### Scenario: Alfred 中可见 Rebuild 选项
- **WHEN** 用户在 Alfred 中输入 `vscdli`
- **THEN** "rebuild index" 选项出现在列表中
