# 前端设计规范 v2

## 概述

现代化单页应用，采用 Glassmorphism + Neumorphism 混合风格，大量留白，流畅动效，细腻的交互反馈。

## 设计语言

### 理念
- **轻盈**：大量留白，呼吸感
- **层次**：通过阴影和模糊建立层级
- **动效**：60fps 流畅动画，微交互丰富
- **克制**：配色精简，强调色点缀

## 配色

```css
:root {
  /* 主色 - 优雅蓝 */
  --primary: #3B82F6;
  --primary-light: #60A5FA;
  --primary-dark: #2563EB;

  /* 背景层次 */
  --bg-base: #0F0F14;
  --bg-surface: #18181F;
  --bg-elevated: #1F1F28;
  --bg-hover: #27272F;

  /* 边框 */
  --border-subtle: rgba(255, 255, 255, 0.06);
  --border-default: rgba(255, 255, 255, 0.1);

  /* 文字 */
  --text-primary: #F4F4F5;
  --text-secondary: #A1A1AA;
  --text-muted: #71717A;

  /* 状态色 */
  --success: #22C55E;
  --warning: #F59E0B;
  --error: #EF4444;

  /* Git 分支 */
  --git: #10B981;

  /* 焦点 */
  --focus-ring: rgba(59, 130, 246, 0.5);
}
```

## 字体

```css
--font-sans: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
--font-mono: 'JetBrains Mono', 'SF Mono', Monaco, Consolas, monospace;

--text-xs: 11px;
--text-sm: 12px;
--text-base: 14px;
--text-lg: 16px;
--text-xl: 20px;
--text-2xl: 24px;

--font-normal: 400;
--font-medium: 500;
--font-semibold: 600;
```

## 圆角

```css
--radius-sm: 6px;
--radius-md: 10px;
--radius-lg: 14px;
--radius-xl: 20px;
--radius-full: 9999px;
```

## 阴影

```css
--shadow-sm: 0 1px 2px rgba(0, 0, 0, 0.3);
--shadow-md: 0 4px 12px rgba(0, 0, 0, 0.4);
--shadow-lg: 0 8px 24px rgba(0, 0, 0, 0.5);
--shadow-glow: 0 0 20px rgba(59, 130, 246, 0.15);
```

## 布局

```
┌──────────────────────────────────────────────────────────────┐
│  TopBar (56px)                                              │
│  ┌─────────────────────┐              ┌────┐ ┌────┐       │
│  │ 🔍 搜索项目...       │              │重建│ │打开│       │
│  └─────────────────────┘              └────┘ └────┘       │
├──────────┬───────────────────────────────────────────────────┤
│          │                                                   │
│ Sidebar  │   Main Content (padding: 32px)                   │
│ (72px)   │                                                   │
│          │   ┌──────────────────────────────────────────┐   │
│  [📊]    │   │                                          │   │
│          │   │         动态内容区域                        │   │
│  [📁]    │   │                                          │   │
│          │   │                                          │   │
│  [🗑️]    │   └──────────────────────────────────────────┘   │
│          │                                                   │
│  [📂]    │                                                   │
│          │                                                   │
│  [⚙️]    │                                                   │
│          │                                                   │
└──────────┴───────────────────────────────────────────────────┘
```

### 侧边栏
- 宽度：72px
- 背景：透明（融入底层）
- 图标尺寸：22px
- 选中项：背景高亮 + 左侧竖条指示器
- 圆角：全圆角（pill shape）选中态

### 主内容区
- 内边距：32px
- 最大宽度：1200px（居中）
- 卡片间距：20px

## 页面原型 v2

### 总览页
```
┌────────────────────────────────────────────────────────────┐
│                                                            │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐ │
│  │    24    │  │    3     │  │    2     │  │    18    │ │
│  │   项目    │  │   回收站  │  │   目录   │  │  Git仓库 │ │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘ │
│                                                            │
│  最近打开                                                  │
│  ┌──────────────────────────────────────────────────────┐ │
│  │  🟢  vsc                  ⎇ main       10 分钟前   │ │
│  ├──────────────────────────────────────────────────────┤ │
│  │  🔵  claude-code           ⎇ main       2 小时前   │ │
│  ├──────────────────────────────────────────────────────┤ │
│  │  ⚪  my-api                ⎇ feat/api   3 天前    │ │
│  └──────────────────────────────────────────────────────┘ │
│                                                            │
└────────────────────────────────────────────────────────────┘
```

### 项目页
```
┌────────────────────────────────────────────────────────────┐
│                                                            │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐        │
│  │ 卡片视图 │ │ 表格视图 │ │         │ │  搜索... │        │
│  └─────────┘ └─────────┘ └─────────┘ └─────────┘        │
│                                                            │
│  ┌──────────────────┐  ┌──────────────────┐             │
│  │                  │  │                  │             │
│  │   🟢  vsc       │  │   🔵  api        │             │
│  │                  │  │                  │             │
│  │   ⎇ main        │  │   ⎇ dev         │             │
│  │   ~/Code/vsc    │  │   ~/Code/api    │             │
│  │                  │  │                  │             │
│  │   ────────────  │  │   ────────────  │             │
│  │   🗑️            │  │   🗑️            │             │
│  └──────────────────┘  └──────────────────┘             │
│                                                            │
└────────────────────────────────────────────────────────────┘
```

### 项目卡片样式
```
.project-card {
  background: var(--bg-elevated);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-lg);
  padding: 20px;
  transition: all 0.2s ease;

  /* Hover: 浮起效果 */
  &:hover {
    transform: translateY(-2px);
    box-shadow: var(--shadow-lg);
    border-color: var(--border-default);
  }
}

.project-card .branch-badge {
  background: color-mix(in srgb, var(--git) 15%, transparent);
  color: var(--git);
  font-size: var(--text-xs);
  font-weight: var(--font-medium);
  padding: 4px 10px;
  border-radius: var(--radius-full);
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.project-card .delete-btn {
  opacity: 0;
  transition: opacity 0.15s;
}
.project-card:hover .delete-btn {
  opacity: 1;
}
```

### 回收站页
```
┌────────────────────────────────────────────────────────────┐
│                                                            │
│  回收站                                        [清空全部]  │
│                                                            │
│  ┌──────────────────────────────────────────────────────┐ │
│  │                                                      │ │
│  │   old-project                                        │ │
│  │   ~/Code/old                         2 小时前        │ │
│  │                                                      │ │
│  │                            [恢复]  [永久删除]        │ │
│  │                                                      │ │
│  └──────────────────────────────────────────────────────┘ │
│                                                            │
│  ┌──────────────────────────────────────────────────────┐ │
│  │                                                      │ │
│  │   temp-backup                                       │ │
│  │   ~/temp                            3 天前          │ │
│  │                                                      │ │
│  │                            [恢复]  [永久删除]        │ │
│  │                                                      │ │
│  └──────────────────────────────────────────────────────┘ │
│                                                            │
└────────────────────────────────────────────────────────────┘
```

### 目录页
```
┌────────────────────────────────────────────────────────────┐
│                                                            │
│  📂 目录浏览                                              │
│                                                            │
│  ┌──────────────────────────────────────────────────────┐ │
│  │                                                      │ │
│  │   ~/Code                                       ▼    │ │
│  │                                                      │ │
│  │   ├── 🗂️  alfred                                   │ │
│  │   │    └── 🗂️  vsc                        [+ 添加] │ │
│  │   │         └── 🗂️  workflow                       │ │
│  │   │                                                │ │
│  │   ├── 🗂️  lover                                   │ │
│  │   │                                                │ │
│  │   └── 🗂️  temp                                    │ │
│  │                                                      │ │
│  └──────────────────────────────────────────────────────┘ │
│                                                            │
└────────────────────────────────────────────────────────────┘
```

### 设置页
```
┌────────────────────────────────────────────────────────────┐
│                                                            │
│  ⚙️ 设置                                                  │
│                                                            │
│  ─────────────────────────────────────────────────────    │
│                                                            │
│  📁 目录配置                                              │
│                                                            │
│  ┌──────────────────────────────────────────────────────┐ │
│  │                                                      │ │
│  │   ~/Code                                    ✕        │ │
│  │   ~/Projects                                 ✕        │ │
│  │   ~/Dev                                      ✕        │ │
│  │                                                      │ │
│  └──────────────────────────────────────────────────────┘ │
│                                                            │
│  [+ 添加目录]                                              │
│                                                            │
│  ─────────────────────────────────────────────────────    │
│                                                            │
│  📤 数据管理                                              │
│                                                            │
│  ┌────────┐  ┌────────┐  ┌────────────────┐              │
│  │ 导出   │  │ 导入   │  │    重建索引    │              │
│  └────────┘  └────────┘  └────────────────┘              │
│                                                            │
│  ─────────────────────────────────────────────────────    │
│                                                            │
│  ℹ️ 关于                                                  │
│  VSC Dashboard v1.0                                       │
│                                                            │
└────────────────────────────────────────────────────────────┘
```

## 组件规范 v2

### 按钮

```css
.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 10px 18px;
  font-size: var(--text-base);
  font-weight: var(--font-medium);
  border-radius: var(--radius-md);
  border: none;
  cursor: pointer;
  transition: all 0.15s ease;
}

.btn-primary {
  background: var(--primary);
  color: white;
}
.btn-primary:hover {
  background: var(--primary-light);
  transform: translateY(-1px);
}
.btn-primary:active {
  transform: translateY(0);
}

.btn-secondary {
  background: var(--bg-elevated);
  color: var(--text-primary);
  border: 1px solid var(--border-default);
}
.btn-secondary:hover {
  background: var(--bg-hover);
  border-color: var(--border-default);
}

.btn-ghost {
  background: transparent;
  color: var(--text-secondary);
}
.btn-ghost:hover {
  background: var(--bg-hover);
  color: var(--text-primary);
}

.btn-danger {
  background: color-mix(in srgb, var(--error) 15%, transparent);
  color: var(--error);
}
.btn-danger:hover {
  background: var(--error);
  color: white;
}

.btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
  transform: none;
}
```

### 输入框

```css
.input {
  background: var(--bg-surface);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  padding: 10px 14px;
  font-size: var(--text-base);
  color: var(--text-primary);
  transition: all 0.15s ease;
  width: 100%;
}

.input:focus {
  outline: none;
  border-color: var(--primary);
  box-shadow: 0 0 0 3px var(--focus-ring);
}

.input::placeholder {
  color: var(--text-muted);
}
```

### 卡片

```css
.card {
  background: var(--bg-elevated);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-lg);
  padding: 20px;
  transition: all 0.2s ease;
}

.card:hover {
  border-color: var(--border-default);
}

.card-elevated {
  background: var(--bg-surface);
  box-shadow: var(--shadow-md);
}
```

### Toast

```css
.toast-container {
  position: fixed;
  bottom: 24px;
  right: 24px;
  display: flex;
  flex-direction: column;
  gap: 8px;
  z-index: 1000;
}

.toast {
  background: var(--bg-elevated);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  padding: 12px 16px;
  display: flex;
  align-items: center;
  gap: 10px;
  box-shadow: var(--shadow-lg);
  animation: slideIn 0.3s cubic-bezier(0.16, 1, 0.3, 1);
  min-width: 280px;
}

.toast.success { border-left: 3px solid var(--success); }
.toast.error { border-left: 3px solid var(--error); }
.toast.warning { border-left: 3px solid var(--warning); }

@keyframes slideIn {
  from {
    transform: translateX(100%);
    opacity: 0;
  }
  to {
    transform: translateX(0);
    opacity: 1;
  }
}

@keyframes slideOut {
  from {
    transform: translateX(0);
    opacity: 1;
  }
  to {
    transform: translateX(100%);
    opacity: 0;
  }
}

.toast.hiding {
  animation: slideOut 0.2s ease forwards;
}
```

### 对话框

```css
.dialog-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.7);
  backdrop-filter: blur(4px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  animation: fadeIn 0.15s ease;
}

.dialog {
  background: var(--bg-surface);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-xl);
  padding: 28px;
  max-width: 400px;
  width: 90%;
  box-shadow: var(--shadow-lg);
  animation: scaleIn 0.2s cubic-bezier(0.16, 1, 0.3, 1);
}

@keyframes scaleIn {
  from {
    transform: scale(0.95);
    opacity: 0;
  }
  to {
    transform: scale(1);
    opacity: 1;
  }
}

.dialog .title {
  font-size: var(--text-lg);
  font-weight: var(--font-semibold);
  margin-bottom: 12px;
}

.dialog .body {
  color: var(--text-secondary);
  margin-bottom: 24px;
}

.dialog .actions {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}
```

### 空状态

```css
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 64px 32px;
  text-align: center;
}

.empty-state .icon {
  font-size: 48px;
  margin-bottom: 16px;
  opacity: 0.6;
}

.empty-state .title {
  font-size: var(--text-lg);
  font-weight: var(--font-medium);
  margin-bottom: 8px;
}

.empty-state .description {
  color: var(--text-muted);
  font-size: var(--text-sm);
}
```

### 侧边栏项

```css
.sidebar-item {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: 44px;
  border-radius: var(--radius-md);
  color: var(--text-muted);
  cursor: pointer;
  transition: all 0.15s ease;
  position: relative;
}

.sidebar-item:hover {
  background: var(--bg-hover);
  color: var(--text-secondary);
}

.sidebar-item.active {
  background: color-mix(in srgb, var(--primary) 15%, transparent);
  color: var(--primary);
}

.sidebar-item.active::before {
  content: '';
  position: absolute;
  left: 0;
  top: 50%;
  transform: translateY(-50%);
  width: 3px;
  height: 20px;
  background: var(--primary);
  border-radius: 0 2px 2px 0;
}
```

### 标签栏（Tabs）

```css
.tabs {
  display: flex;
  gap: 4px;
  background: var(--bg-surface);
  padding: 4px;
  border-radius: var(--radius-md);
}

.tab {
  padding: 8px 16px;
  font-size: var(--text-sm);
  font-weight: var(--font-medium);
  color: var(--text-muted);
  background: transparent;
  border: none;
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: all 0.15s ease;
}

.tab:hover {
  color: var(--text-secondary);
}

.tab.active {
  background: var(--bg-elevated);
  color: var(--text-primary);
  box-shadow: var(--shadow-sm);
}
```

## 动效规范

| 场景 | 动画 |
|------|------|
| 页面切换 | 淡入淡出，150ms |
| 卡片 hover | translateY(-2px)，200ms ease |
| 按钮 hover | translateY(-1px)，150ms |
| Toast 入场 | translateX(100%) → 0，300ms cubic-bezier(0.16, 1, 0.3, 1) |
| Toast 退场 | translateX(0) → 100%，200ms ease |
| 对话框 | scale(0.95) → 1，200ms cubic-bezier(0.16, 1, 0.3, 1) |
| 列表项加载 | stagger 动画，每项延迟 50ms |
| 骨架屏 | shimmer 动画，1.5s 循环 |

## 状态定义

### 加载状态
```css
.skeleton {
  background: linear-gradient(
    90deg,
    var(--bg-surface) 0%,
    var(--bg-elevated) 50%,
    var(--bg-surface) 100%
  );
  background-size: 200% 100%;
  animation: shimmer 1.5s infinite;
  border-radius: var(--radius-sm);
}

@keyframes shimmer {
  0% { background-position: 200% 0; }
  100% { background-position: -200% 0; }
}
```

### 空状态文案

| 页面 | 图标 | 标题 | 描述 |
|------|------|------|------|
| 项目列表 | 📭 | 暂无项目 | 使用 Alfred 打开一些项目，它们会出现在这里 |
| 回收站 | 🗑️ | 回收站是空的 | 删除的项目会出现在这里 |
| 目录 | 📁 | 未配置目录 | 在设置中添加要浏览的目录 |

## 快捷键

| 按键 | 功能 |
|------|------|
| `1-5` | 切换标签页 |
| `Cmd+K` 或 `/` | 聚焦搜索 |
| `Esc` | 清除搜索 / 关闭对话框 |
| `Enter` | 打开选中项目 |
| `Delete` | 删除选中项目 |

## 技术实现

### 文件结构
```
src/web/
├── index.html
├── styles.css
├── app.js
└── assets/
    └── icons.svg   # SVG 图标集
```

### 图标方案
使用 Lucide Icons（开源、线条风格、现代）
CDN: `https://unpkg.com/lucide@latest`

### API 调用封装
```javascript
class ApiClient {
  constructor(baseUrl = '') {
    this.baseUrl = baseUrl;
  }

  async request(method, path, body = null) {
    const options = {
      method,
      headers: { 'Content-Type': 'application/json' },
    };
    if (body) options.body = JSON.stringify(body);

    const res = await fetch(this.baseUrl + path, options);
    const json = await res.json();

    if (!json.ok) {
      throw new Error(json.error || 'Request failed');
    }
    return json.data;
  }

  getRecords(params) {
    const qs = new URLSearchParams(params).toString();
    return this.request('GET', `/api/records${qs ? '?' + qs : ''}`);
  }
  deleteRecord(id) { return this.request('DELETE', `/api/records/${id}`); }
  restoreRecord(id) { return this.request('POST', `/api/trash/${id}/restore`); }
  // ...
}

const api = new ApiClient();
```

### 轮询机制
```javascript
class PollingService {
  start(interval = 30000) {
    this.timer = setInterval(() => this.poll(), interval);
  }
  stop() {
    clearInterval(this.timer);
  }
  async poll() {
    // 获取最新数据
  }
}
```
