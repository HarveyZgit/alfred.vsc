# 发布正式版本

推送 `X.Y.Z` 格式的标签会触发 `.github/workflows/ci.yml`。标签不带 `v`，例如 `3.0.3`。只支持正式版本，不支持预发布后缀。

## 发布步骤

1. 从最新 `main` 创建版本分支，同步以下版本号：根目录与 `web/` 的 `package.json`、`internal/workflow/config.go`、`THIRD_PARTY_NOTICES.txt`。在 `CHANGELOG.md` 添加对应 `## vX.Y.Z` 小节。
2. 执行 `python3 scripts/generate_workflow.py` 更新 `public/info.plist`，执行 `python3 scripts/release.py check`，提交 PR，等待 CI 通过并合并。
3. 在合并后的提交上创建并推送标签。以下以 **尚未发布的 3.0.3** 为例，实际发布时替换版本号：

   ```sh
   git switch main
   git pull --ff-only origin main
   python3 scripts/release.py check --tag 3.0.3
   git tag -a 3.0.3 -m 'VSC 3.0.3'
   git push origin 3.0.3
   ```

通过终端或有权限的 GitHub 客户端推送标签即可。不要先在网页发布空的 Release；Actions 会创建并发布它。由另一个 Actions 工作流使用默认 `GITHUB_TOKEN` 推送的标签通常不会触发新工作流，因此不要用这种方式替代上述标签推送。

## 自动完成的工作

- 校验标签、CLI、前端、Workflow 和第三方声明的版本号一致，且对应 CHANGELOG 小节非空。
- 在 Linux 和 macOS 上运行现有测试、构建、二进制验收与包验证。
- 从已验证的 Linux 构建暂存正式 universal Workflow、迁移脚本和只包含这两个附件的 `SHA256SUMS`。普通 PR 也会生成 `vsc-release` artifact，便于合并前检查。
- 两个平台全部成功后，发布任务检查标签提交属于 `main` 历史，使用该次运行的 `vsc-release` artifact，不重新构建。
- 根据 CHANGELOG 创建草稿，上传三个附件，下载并逐字节核对，再公开为正式版本并标记 Latest。

发布任务单独获得 `contents: write`，通过 GitHub Actions 自带的 `GITHUB_TOKEN` 调用 GitHub CLI，无需额外 device 登录或个人 token。仓库/组织策略需要允许该任务写入仓库内容。普通测试任务只具有读取权限。

## 失败与重试

测试失败时不会开始发布。上传或核对失败时保留草稿，可在该次 Actions 运行中选择 **Re-run failed jobs**；重试会补齐、替换同名草稿附件，再核对后发布。草稿含额外附件时流程会停止，请先人工核查。

已经公开的同名 Release 会拒绝覆盖，即使重新运行也不会替换正式附件。需要修复已发布版本时，递增版本号并走新的发布流程。不要通过移动旧标签重发。

标签提交不在 `main` 历史、标签与版本不符、缺失 CHANGELOG 或误用 dev 包都会阻止发布。发布失败后先修复原因；涉及代码变更时，应合并修复并使用新的版本标签。

自动发布不包含 Apple Developer ID 签名或公证，Gatekeeper 的安装处理仍见[安装指南](INSTALLATION.zh-CN.md)。
