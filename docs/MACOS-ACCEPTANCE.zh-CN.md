# macOS / Alfred 验收清单

Linux 的二进制与参数验收已自动化。以下测试需要真实 Alfred 和 IDE；交叉编译成功不能证明这些行为已经通过。

1. 导入 `vsc-dev.alfredworkflow`，确认旧 `vsc` 仍正常，测试关键词为 `vscd`，设置两个包含 Git 项目的根目录。
2. 输入 `vscd`：VS Code 本地目录、SSH / 容器目录和根目录下 Git 仓库均可找到；重复目录只有一条；本地文件、远程文件、workspace 文件不出现。
3. 新开 / 关闭 VS Code 最近项目后再次唤醒，确认历史变化；不要只测试程序自建缓存。特别记录 VS Code 版本与 `vsc doctor` 显示的实际数据库位置。
4. 对同一仓库切换分支，再次唤醒或修改查询，分支应更新；测试 worktree 和 detached HEAD。远程任何时候都不显示分支，也不引发远程连接。
5. 快速连续输入、退格、清空、再次唤醒，最终结果应对应最后查询，无旧结果覆盖。Alfred Script Filter 设置应为终止前一次脚本后运行新查询；检查实际 Alfred 版本对导入配置的解释。
6. Enter 分别打开含中文、空格、`#`、`%` 的路径；Cmd + Enter 展示四个 IDE，可用 IDE 应能打开；第二级菜单 Cmd + Enter 后再次查询应记住选择。未安装 IDE 应明确提示。
7. Shift + Enter 验证各 IDE 新窗口行为；VS Code / Cursor / Trae CLI 的版本差异以真实应用为准。
8. VS Code 验证 SSH 和 Dev Container；Cursor / Trae 验证已安装远程支持的 SSH；Zed 验证明文 SSH alias。容器或编码 SSH authority 不应被错误转为 Zed SSH 地址。
9. Ctrl + Enter 隐藏，刷新仓库后仍隐藏；管理菜单恢复；Alt + Enter 固定后排序生效。确认 Cmd、Ctrl、Alt、Shift 的修饰键连接与提示一致。
10. `vscd /` 用 Tab 选择根目录 / 子目录、返回上级；相同子目录名不会跨根目录串线。
11. 首次无缓存、增加仓库、临时拔掉外置盘时观察后台索引和提示；确认输入不等待全盘扫描。慢挂载属于需实机验证的性能场景。
12. 对 Apple Silicon 和 Intel 的可用机器分别检查二进制；可执行 `lipo -archs vsc`、`codesign --verify vsc`。包未做 Developer ID 公证，若系统阻止，按系统的可信软件允许流程处理。

出现问题时保存：系统 / Alfred / IDE 版本、脱敏后的 `doctor`、Alfred Debugger 输出、输入文本与按键。请勿提交包含真实主机名、私人路径的数据库或完整偏好文件。
