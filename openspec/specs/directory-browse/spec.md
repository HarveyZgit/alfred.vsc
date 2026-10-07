# Directory browse (V3)

## Requirement: Activation and root identity
以 `/` 开头的查询 SHALL 浏览配置根目录。单根目录 `/` 展示直接子目录；多个根目录 `/` SHALL 先展示根目录选择，autocomplete 分别为 `/1/`、`/2/`，同名子目录不会混合。

## Requirement: Hierarchical browse and matching
`/path/` SHALL 展示当前层级的直接子目录；`/path/keyword` SHALL 对该层子目录做 Unicode 模糊匹配。多根目录在路径前保留根目录编号。查询 SHALL NOT 递归扫描全部目录树。Git 项目的跨层搜索由普通查询和后台索引提供。

## Requirement: Filtering and traversal
隐藏目录、依赖目录和构建产物目录 SHALL 被跳过，结果受 VSC_LIMIT 限制。钻取后的真实路径 SHALL 保持在选定根目录内；越界路径返回错误。只有目录可以打开，不展示文件。

## Requirement: Navigation and actions
Tab SHALL 进入选定子目录，返回上级项 SHALL 提供明确 autocomplete。目录项目 SHALL 使用统一的 JSON 打开请求，支持默认 IDE、新窗口、IDE 选择、隐藏和固定操作。当前目录的分支读取 SHALL 与普通本地结果一致。浏览本身 SHALL NOT 将非 Git 目录写入持久历史。

## Requirement: Empty state
未配置根目录 SHALL 显示配置提示；无匹配项 SHALL 显示不可执行的提示项；I/O 错误 SHALL 返回有效的 Alfred 错误反馈。
