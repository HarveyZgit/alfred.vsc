# VSC CHANGELOG

## v1.2.0 (2022-08-20) [!7](https://github.com/HarveyZgit/alfred.vsc/pull/7)

### vsc

- 🎉 新增 completely rebuild index，在 rebuild index 时会忽略所有缓存记录
- 🍀 vsc: 保存删除的记录

### vscli

- 🎉 vscli 的相关操作会进行通知
- 🎉 新增 completely rebuild index，在 rebuild index 时会忽略所有缓存记录
- 🍀 rebuild index 时会将曾经删除的记录过滤掉
- 🙏 thx alfy and bey ~

### common

- 🍀 VSC_DIRECTORIES support `~` alias

## v1.1.1 (2022-07-16) [!4](https://github.com/HarveyZgit/alfred.vsc/pull/4)

- 🎉 没有关键词的时候展示 vscode 菜单中的最近打开列表
- 🎉 在 `vsc` 进行查找时，支持按住 `command` 键来使用 `vscode-insiders` 打开目录（需要配置环境变量 `VSC_CODE_INSIDERS_BIN`）
- 🍀 优化构建脚本、构建流程

## v1.1.0 (2022-07-10) [!3](https://github.com/HarveyZgit/alfred.vsc/pull/3)

- 🎉 新增 alfred 命令：`vscli` ，用于操作 vsc 的 record list。目前支持 `手动重建索引`
- 🎉 新增环境变量 `VSC_DIRECTORIES`，支持用户将某个目录下的子文件夹加入到 records 里面（多个目录时用英文逗号分隔）
- 🎉 在 `vsc` 进行查找时，支持按住 `control` 键来删除所选项
- 🍀 解耦更新和查找的逻辑，搜索的速度更快了
- 🗑 移除 v0.0.3 新增的特性，默认找 vscode db 中最近的 100 条

## v0.0.3 (2022-05-13) [!2](https://github.com/HarveyZgit/alfred.vsc/pull/2)

- 🎉 支持查找最新 xx 条记录，默认为最近 50 条

## v0.0.2 (2022-04-26) [!1](https://github.com/HarveyZgit/alfred.vsc/pull/1)

- 🎉 支持环境变量配置 & cache db instance

## v0.0.1 (2022-04-26)

- 🎉 A new alfred workflow
