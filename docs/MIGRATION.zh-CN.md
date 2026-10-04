# 从旧版迁移到 V3

迁移工具支持仓库历史中的 V1 JavaScript / V2 Python `.records.cache.json` 格式。它迁移新版范围内项目的隐藏状态和旧版最近使用顺序，保留完整备份，生成未迁移列表。旧版没有可靠使用次数，因此不会生成虚构统计。

**范围仍为 VS Code 目录历史 ∪ 配置根目录下 Git 仓库。** 不在这个范围内的记录不导入。文件、无法确定类型的远程记录及无效路径也不导入。迁移不会修改 VS Code 数据库，不连接远程，不使用旧分支缓存。

## 1. 找到旧缓存并配置新版

替换旧正式工作流前，在 Alfred Workflows 里右键旧 VSC，选择 Open in Finder，复制整个目录作为备份。通常 `.records.cache.json` 位于该目录，源码开发版可能位于 `src/`。不要先删除旧工作流，Alfred 升级时可能清理旧目录。迁移源可传目录或这个缓存文件的完整路径。

先导入隔离开发包并配置根目录。在 `vscdli` 中执行“生成高级配置”，把当前 Alfred 配置写入开发数据目录的 `config.json`。若该文件已存在，工具不会覆盖；请确认其中 `roots` 与工作流配置一致。终端并不会自动继承 Alfred 工作流变量，必要时在迁移命令前显式设置 `VSC_DIRECTORIES` / `VSC_DB_PATH`。

新版工作流右键 Open in Finder，在其目录打开终端。包内已包含 `vsc` 和 `scripts/migrate-v2.sh`，不需要 Python、Node、Go 或 sqlite3。

## 2. 预览

```sh
./scripts/migrate-v2.sh \
  --source "/旧版工作流备份的完整路径" \
  --data-dir "/tmp/vsc-dev-v3" \
  --dry-run > migration-preview.json
```

默认不加 `--apply` 也是预览。预览不写偏好、迁移备份或程序缓存。结果为 JSON：

- `records` / `hidden_records`：旧记录和隐藏记录总数。
- `directory_order_imported` / `hidden_imported`：本次范围内可处理的目录顺序和隐藏状态数量；不是新增量。
- `items`：逐条映射，保留原始列表位置和原路径。
- `unmigrated`：未迁移列表，每项含 `original`、`target`、`status`、`how_to_migrate`，隐藏记录另有 `hidden: true`。
- `backup`：实际执行后生成备份的位置；预览时尚未创建。
- `already_applied`：同一输入、当前范围内的所有可迁移项是否已处理。

普通预览不会递归改变源目录，仓库发现只读扫描配置根目录，沿用索引的深度、数量和时间预算。若根目录不可读或数据库不可用，报告会注明；先排除来源问题再执行，避免把暂时不可见的目录误认为永久范围外。

## 3. 执行与验证

确认预览后执行相同命令并加 `--apply`：

```sh
./scripts/migrate-v2.sh \
  --source "/旧版工作流备份的完整路径" \
  --data-dir "/tmp/vsc-dev-v3" \
  --apply > migration-result.json
```

在 `vscd` 中检查隐藏状态和顺序。迁移时，同等匹配与固定优先级下，已有范围内项目按旧最近使用顺序展示；当 VS Code 历史目录或顺序与迁移时不同时，恢复使用 VS Code 顺序。旧顺序仍留在备份中，不维护第二份持续更新的 MRU。

确认后可对正式版数据目录再次执行。Alfred 默认正式数据目录通常是：

```text
~/Library/Application Support/Alfred/Workflow Data/com.harvey.alfredapp.vsc
```

以正式工作流 `vscli` 的“生成高级配置”或 `doctor` 输出为准。显式指定 `--data-dir`，避免终端默认目录与 Alfred 数据目录不同。测试包和正式包迁移互相隔离。

也可直接调用二进制，两个入口行为相同：

```sh
./vsc migrate-legacy --source "/旧缓存/.records.cache.json" --data-dir "/目标数据目录" --apply
```

从源码目录运行脚本时，脚本会寻找 `build/vsc`（macOS）或 `build/vsc-linux-amd64`。也可通过 `VSC_BINARY` 指定新版可执行文件。

## 4. 如何自行补迁移未导入项目

查看 `migration-result.json` 的 `unmigrated` 列表，每条都有具体说明：

| 原因 | 处理方法 |
| --- | --- |
| `outside_current_sources` | 用 VS Code 打开该目录，或者把本地 Git 仓库的上级目录加入新版根目录配置 |
| `file_or_unconfirmed_directory` | 文件不支持；若实际是目录，先用 VS Code 的“打开文件夹”打开，远程同样在 VS Code 中打开远程目录 |
| `invalid_path` | 确认目录的新位置和 URI；在自己的旧缓存副本中修正 `path`，并先加入新版来源，再重新预览 |

来源补齐后，使用原缓存重复运行预览和 `--apply` 即可补迁移。同一输入的已处理记录有独立标记，不会重复添加，也不会把迁移后已手动取消隐藏的项目再次隐藏。范围外的隐藏记录也只有进入新版来源后才迁移。

若修改了旧缓存内容，会被视为一份新输入；请先预览，新输入的隐藏项会再次按源文件导入。完全不需要的旧记录可以留在未迁移列表，不会进入普通搜索。

## 5. 备份与恢复

执行时，在目标数据目录的 `migrations/<输入摘要>/` 中创建：

- `legacy-records.json`：旧缓存逐字节副本，包括所有范围外记录、原排序和未知字段，仅供备份。
- `legacy-user-config.json`：同目录旧用户配置的逐字节副本（若存在），不覆盖新版配置。
- `preferences-before.json`：第一次迁移前的新版偏好。
- `report.json`：第一次执行的迁移计划；补迁移后的最新状态以命令输出 `migration-result.json` 为准。

完整备份落盘后才原子提交新版偏好；输入损坏或目标偏好损坏会报错，不覆盖原文件。重复执行不会覆盖第一次备份。一次迁移中断后可用同一命令重试。

如果要回退，先关闭 VSC 的 Alfred 查询/操作，把当前 `preferences.json` 另存为备份，再用该次 `preferences-before.json` 替换它。这样会同时回退那之后的固定、隐藏和 IDE 偏好，因此恢复前务必保存当前文件。程序不自动回滚已存在的用户改动。


## 从管理面板预览和补迁移

`vscli`（开发包 `vscdli`）→ 打开记录管理面板 → 旧版迁移。填写旧版数据目录或 `.records.cache.json` 路径，点击预览；未迁移清单逐条给出处理办法。到「目录与 IDE」修改 Git 根目录，或先用 VS Code 打开对应目录，再重新预览并执行补迁移。面板设置和命令行迁移读取同一目标数据目录中的 managed 配置。

已保存结果会区分最新执行结果与首次迁移历史快照。`migrations/<fingerprint>/report.json` 保留最初备份计划，`latest-report.json` 保存最近一次成功执行报告；二者都不是实时来源清单，当前范围以重新预览为准。原来源已移动时，填写当前实际路径再预览即可；不要将首次报告中的未迁移数量当成补迁移后的结果。
