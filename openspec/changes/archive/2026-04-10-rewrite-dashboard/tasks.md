## 1. 修复 shared 模块

- [x] 1.1 在 shared.py 中添加 `get_git_branch_for_browse()` 函数
- [x] 1.2 在 shared.py 中添加 `update_branches_cache()` 函数

## 2. 修复 server 模块

- [x] 2.1 修复 server.py 中的 WEB_DIR 路径，改为指向 `src/web/`
- [x] 2.2 验证根路径正确提供 index.html
- [x] 2.3 验证静态资源（CSS、JS、图片）以正确的 MIME 类型提供服务
- [x] 2.4 验证 API 端点返回正确的信封格式 `{ok: bool, data: ...}`
- [x] 2.5 为所有响应添加 CORS 头

## 3. 修复 dev_install 脚本

- [x] 3.1 在 dev_install.py 的符号链接列表中添加 `server.py`
- [x] 3.2 在 dev_install.py 的符号链接列表中添加 `web/` 目录
- [x] 3.3 在 dev_install.py 的状态检查中添加 `server.py` 和 `web/`
- [x] 3.4 重新安装并验证所有文件正确符号链接

## 4. 修复 Alfred 集成

- [x] 4.1 在 info.plist 的 List Filter 项目中添加 "manage" 选项
- [x] 4.2 Alfred List Filter items 包含 "🚀 启动管理面板" 和 "⏹ 停止管理面板"
- [x] 4.3 更新脚本以处理 `manage start` 和 `manage stop` 参数
- [x] 4.4 修复脚本路径，使用 `./cli.py` 而不是绝对路径

## 5. 实现 manage start/stop 命令

- [x] 5.1 重构 CLI 支持 `--manage-start` 和 `--manage-stop` 标志
- [x] 5.2 实现 `manage start` 启动服务器并自动打开浏览器
- [x] 5.3 如果服务器已运行，只打开浏览器不重启
- [x] 5.4 实现 `manage stop` 停止服务器

## 6. 前端实现

- [x] 6.1 创建现代化的 index.html
- [x] 6.2 创建现代化的 styles.css
- [x] 6.3 创建现代化的 app.js
- [x] 6.4 创建 api.js API 封装

## 7. 测试完整流程

- [x] 7.1 运行 `vscli manage start` 并验证服务器启动
- [x] 7.2 验证浏览器自动打开 http://localhost:3847
- [x] 7.3 再次运行 `vscli manage start` 验证只打开浏览器不报错
- [x] 7.4 打开浏览器访问并验证 SPA 加载
- [x] 7.5 验证总览页显示统计数字
- [x] 7.6 验证项目页显示缓存中的记录
- [x] 7.7 测试删除一个项目并验证出现在回收站页
- [x] 7.8 测试从回收站恢复一个项目
- [x] 7.9 运行 `vscli manage stop` 并验证服务器停止
- [ ] 7.10 验证 Alfred 中 `vscli` 关键字显示所有选项
