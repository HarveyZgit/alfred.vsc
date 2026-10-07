# macOS 安装排查

## 导入后看到多个 VSC

显示名称不能证明是同一工作流。在 Alfred Preferences → Workflows 中，右键每个 VSC → Edit Details，核对 Bundle ID：

| Bundle ID | 用途与处理 |
| --- | --- |
| `com.harvey.alfredapp.vsc.v3` | 当前正式版，入口 `vsc` / `vscli` |
| `com.harvey.alfredapp.vsc.v3.dev` | 隔离开发版，入口 `vscd` / `vscdli`；不用时可先禁用 |
| `com.harvey.alfredapp.vsc` | 保留的旧版；需要保留时使用名称 `VSC Old`、入口 `vsco` / `vscoli` |
| 其他或空 ID | 可能是早期手工安装副本；核对版本、关键词和数据来源后处理 |

正式包的 Bundle ID 固定为 `com.harvey.alfredapp.vsc.v3`。不同 ID 会共存，不会互相替换；正式版不会自动删除旧版或开发版。导入时检查 Alfred 是否提示更新现有工作流。

如果两个副本的 ID 都是正式 ID，先确认哪个目录是刚导入的版本，并禁用多余副本，避免关键词/快捷键冲突。不要为了处理重复项而删除 Workflow Data：同 ID 副本可能共用该数据目录。确认新版本的配置、隐藏/固定记录和项目 IDE 均正常后，再清理多余工作流。

## Apple 无法验证 vsc

当前预览安装包没有使用 Developer ID Application 证书签名，也没有提交 Apple notarization。浏览器下载的文件可能带有 `com.apple.quarantine` 隔离标记；macOS 因而提示无法验证开发者/恶意软件检查。该提示本身不表示检测到了恶意软件。

**新下载或更新二进制后可能再次出现。** 同一个已经放行的文件通常无需每次运行都放行，但之前的允许不保证对新文件继续有效。

确认安装包来自本项目对应提交的 CI artifact 后，优先使用系统提供的允许流程：

1. 尝试运行一次以触发提示。
2. 打开系统设置 → 隐私与安全性，在安全性区域找到本次阻止记录，选择「仍要打开」（Open Anyway），按系统要求确认。
3. 回到 Alfred 再运行工作流。

如果命令行程序没有出现可用的允许按钮，可只清除已确认来源的工作流二进制上的隔离标记。先在 Alfred 中右键目标正式版工作流 → Open in Finder，再在该目录打开终端：

```sh
# 先确认当前目录中的 Bundle ID 是 com.harvey.alfredapp.vsc.v3
/usr/libexec/PlistBuddy -c 'Print :bundleid' ./info.plist
# 仅处理当前工作流的 vsc 文件
xattr -d com.apple.quarantine ./vsc
```

这需要手动对已确认来源的文件执行，不要用 `sudo`、全局关闭 Gatekeeper，也不要递归修改整个 Alfred 配置目录。若提示没有该属性，说明这个文件未带隔离标记，需要根据实际 macOS 错误进一步定位。再次从浏览器下载安装的新文件仍可能带上该属性。

SHA256SUMS 用于核对对应 CI 产物的完整性，不能替代系统签名或 Apple 公证。

## 长期发布方案

面向日常用户分发，需要在 macOS 发布流程中用 Apple Developer 账号的 **Developer ID Application** 证书签署所有分发架构的二进制（含 hardened runtime 和可信时间戳），再提交 Apple notarization 并验证结果。发布流水线需要证书私钥与公证凭据；当前仓库未配置这些凭据，也未声称已公证。

Go 对 arm64 默认生成的 ad-hoc 签名只满足该架构的代码签名要求，不证明开发者身份，不能消除此类 Gatekeeper 提示。普通 ZIP/Alfred workflow 不能像受支持的 app、pkg、dmg 那样直接装订公证票据；正式发布还需选择和验证合适的分发容器与联网/离线体验。
