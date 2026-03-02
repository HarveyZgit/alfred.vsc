## 1. shared.py — 新增扫描基础设施

- [x] 1.1 新增 `SKIP_DIRS` 常量集合（`node_modules`, `__pycache__`, `venv`, `.venv`, `dist`, `build`, `vendor`, `target`, `Pods`, `.git`, `.svn`, `.hg`）
- [x] 1.2 新增 `scan_subdirectories(roots, segments, search, max_depth)` 函数：接收 VSC_DIRECTORIES 根列表、path segments、搜索关键字和最大递归深度，返回匹配的目录 Path 列表
- [x] 1.3 在 `scan_subdirectories` 中实现目录过滤逻辑：跳过隐藏目录（`.` 开头）和 `SKIP_DIRS` 中的目录

## 2. vsc.py — 查询解析

- [x] 2.1 新增 `parse_browse_query(query)` 函数：将 `/` 后的内容解析为 `(segments, search)` 元组
- [x] 2.2 修改 `main()` 函数：检测 query 以 `/` 开头时分流到目录浏览逻辑

## 3. vsc.py — 目录浏览核心逻辑

- [x] 3.1 新增 `browse_directories(query)` 函数：协调解析、扫描、过滤、格式化的完整流程
- [x] 3.2 实现顶层列出模式：query 为 `/` 时，列出所有根的直接子目录
- [x] 3.3 实现平铺递归搜索模式：query 为 `/keyword` 时，递归扫描（depth ≤ 5）+ fuzzy match
- [x] 3.4 实现层级钻入模式：query 为 `/path/` 时，定位到 path 目录，列其直接子目录
- [x] 3.5 实现钻入搜索模式：query 为 `/path/keyword` 时，定位到 path 目录，在子目录中 fuzzy match

## 4. vsc.py — 结果格式化

- [x] 4.1 新增 `format_browse_results(dirs, root_path)` 函数：将目录 Path 列表转换为 Alfred Script Filter JSON 格式（title=目录名, subtitle=完整路径, arg=file://路径, icon=folder.png）

## 5. 验证

- [x] 5.1 手动测试：`vsc /` 列出顶层目录
- [x] 5.2 手动测试：`vsc /alf` 递归搜索匹配
- [x] 5.3 手动测试：`vsc /alfred/` 钻入子目录
- [x] 5.4 手动测试：`vsc /alfred/vs` 钻入 + 搜索
- [x] 5.5 手动测试：选中目录后 Enter 正常打开 IDE
