/* app.js — VSC Dashboard main application */

(function () {
  "use strict";

  // ============================================================
  // State
  // ============================================================
  let currentView = "records";
  let records = [];
  let trash = [];
  let config = {};
  let searchTimer = null;

  // ============================================================
  // Init
  // ============================================================
  async function init() {
    initTheme();
    setupNavigation();
    setupEventListeners();
    loadRecords();
    setInterval(refreshStats, 30000);
  }

  // ============================================================
  // Theme
  // ============================================================
  function initTheme() {
    const saved = localStorage.getItem("vsc-theme") || "dark";
    applyTheme(saved);
    updateThemeSwitch(saved);
  }

  function updateThemeSwitch(theme) {
    const thumb = document.querySelector(".switch-thumb");
    if (theme === "dark") {
      thumb.style.transform = "translateX(20px)";
    } else {
      thumb.style.transform = "translateX(0)";
    }
  }

  function applyTheme(theme) {
    document.documentElement.setAttribute("data-theme", theme);
    localStorage.setItem("vsc-theme", theme);
  }

  // ============================================================
  // Navigation
  // ============================================================
  function setupNavigation() {
    document.querySelectorAll(".nav-item").forEach((el) => {
      el.addEventListener("click", () => {
        const view = el.dataset.view;
        if (!view) return;
        switchView(view);
      });
    });
  }

  function switchView(view) {
    currentView = view;
    document.querySelectorAll(".nav-item").forEach((el) => {
      el.classList.toggle("active", el.dataset.view === view);
    });
    document.querySelectorAll(".view").forEach((el) => {
      el.classList.toggle("active", el.id === `view-${view}`);
    });
    if (view === "records") loadRecords();
    else if (view === "trash") loadTrash();
    else if (view === "settings") loadSettings();
  }

  // ============================================================
  // Event Listeners
  // ============================================================
  function setupEventListeners() {
    // Topbar
    document.getElementById("search-input").addEventListener("input", handleSearch);

    // Theme switch
    const track = document.getElementById("switch-track");
    track.addEventListener("click", () => {
      const current = document.documentElement.getAttribute("data-theme") || "dark";
      const next = current === "dark" ? "light" : "dark";
      applyTheme(next);
      updateThemeSwitch(next);
    });

    // Records view
    document.getElementById("sort-select").addEventListener("change", loadRecords);
    document.getElementById("toggle-card").addEventListener("click", () => setViewMode("card"));
    document.getElementById("toggle-table").addEventListener("click", () => setViewMode("table"));

    // Trash
    document.getElementById("btn-clear-trash").addEventListener("click", handleClearTrash);

    // Settings - directories
    document.getElementById("btn-add-dir").addEventListener("click", handleAddDir);

    // Settings - data management
    document.getElementById("btn-export").addEventListener("click", handleExport);
    document.getElementById("btn-import").addEventListener("click", handleImport);
    document.getElementById("btn-rebuild-settings").addEventListener("click", handleRebuild);

    // Dialog - handled in showDialog via onclick
  }

  // ============================================================
  // View Mode
  // ============================================================
  let viewMode = "card";

  function setViewMode(mode) {
    viewMode = mode;
    document.getElementById("toggle-card").classList.toggle("active", mode === "card");
    document.getElementById("toggle-table").classList.toggle("active", mode === "table");
    const el = document.getElementById("records-list");
    el.className = mode === "card" ? "records-list card-view" : "records-list table-view";
    renderRecords();
  }

  // ============================================================
  // Toast
  // ============================================================
  function toast(msg, type = "default") {
    const container = document.getElementById("toast-container");
    const el = document.createElement("div");
    el.className = `toast ${type}`;
    el.textContent = msg;
    container.appendChild(el);
    requestAnimationFrame(() => {
      el.style.opacity = "1";
    });
    setTimeout(() => {
      el.style.opacity = "0";
      el.style.transition = "opacity .3s";
      setTimeout(() => el.remove(), 300);
    }, 3000);
  }

  // ============================================================
  // Dialog
  // ============================================================
  function showDialog(title, body, confirmText = "确定", cancelText = "取消") {
    return new Promise((resolve) => {
      document.getElementById("dialog-title").textContent = title;
      document.getElementById("dialog-body").textContent = body;
      document.getElementById("dialog-confirm").textContent = confirmText;
      document.getElementById("dialog-cancel").textContent = cancelText;

      const overlay = document.getElementById("dialog-overlay");
      const confirmBtn = document.getElementById("dialog-confirm");
      const cancelBtn = document.getElementById("dialog-cancel");

      // Remove old handlers by replacing buttons
      const newConfirm = confirmBtn.cloneNode(true);
      const newCancel = cancelBtn.cloneNode(true);
      confirmBtn.parentNode.replaceChild(newConfirm, confirmBtn);
      cancelBtn.parentNode.replaceChild(newCancel, cancelBtn);

      newConfirm.onclick = (e) => {
        e.stopPropagation();
        overlay.classList.add("hidden");
        newConfirm.onclick = null;
        newCancel.onclick = null;
        resolve(true);
      };

      newCancel.onclick = (e) => {
        e.stopPropagation();
        overlay.classList.add("hidden");
        newConfirm.onclick = null;
        newCancel.onclick = null;
        resolve(false);
      };

      overlay.onclick = (e) => {
        if (e.target === overlay) {
          overlay.classList.add("hidden");
          newConfirm.onclick = null;
          newCancel.onclick = null;
          resolve(false);
        }
      };

      overlay.classList.remove("hidden");
    });
  }

  async function confirmDialog(title, body) {
    return showDialog(title, body, "确定", "取消");
  }

  // ============================================================
  // Records (main page with stats)
  // ============================================================
  async function loadRecords() {
    const sort = document.getElementById("sort-select")?.value || "recent";
    const q = document.getElementById("search-input")?.value || "";

    try {
      const [stats, data] = await Promise.all([
        API.getStats(),
        API.getRecords({ sort, q }),
      ]);
      renderStats(stats);
      records = data.records || [];
      renderRecords();
    } catch (e) {
      toast("加载项目失败: " + e.message, "error");
    }
  }

  function renderStats(stats) {
    document.getElementById("stat-projects").textContent = stats.total_records ?? 0;
    document.getElementById("stat-trash").textContent = stats.total_trash ?? 0;
    document.getElementById("stat-git").textContent = stats.git_repos ?? 0;
  }

  async function refreshStats() {
    try {
      const stats = await API.getStats();
      renderStats(stats);
    } catch (_) {}
  }

  function renderRecords() {
    const el = document.getElementById("records-list");
    if (!records.length) {
      el.innerHTML = '<div class="empty-state"><div class="empty-icon">📂</div><div class="empty-title">暂无项目</div><div class="empty-description">重建索引以加载项目</div></div>';
      return;
    }

    if (viewMode === "card") {
      el.innerHTML = records.map((r) => buildRecordCard(r)).join("");
    } else {
      el.innerHTML = `<table class="records-table"><thead><tr><th>名称</th><th>路径</th><th>操作</th></tr></thead><tbody>${records.map((r) => buildTableRow(r)).join("")}</tbody></table>`;
    }

    // Bind events
    el.querySelectorAll(".project-card").forEach((item) => {
      item.addEventListener("click", (e) => {
        if (e.target.closest(".delete-btn")) return;
        openRecord(item.dataset.path);
      });
      item.querySelector(".delete-btn")?.addEventListener("click", (e) => {
        e.stopPropagation();
        handleDeleteRecord(item.dataset.id);
      });
    });

    // Also bind table rows
    el.querySelectorAll("tr[data-id]").forEach((row) => {
      row.addEventListener("click", (e) => {
        if (e.target.closest(".delete-btn")) return;
        openRecord(row.dataset.path);
      });
      row.querySelector(".delete-btn")?.addEventListener("click", (e) => {
        e.stopPropagation();
        handleDeleteRecord(row.dataset.id);
      });
    });
  }

  function buildRecordCard(r) {
    const branchTag = r.branch ? `<span class="branch">⎇ ${escHtml(r.branch)}</span>` : "";
    return `
      <div class="project-card" data-id="${escAttr(r.__vsc_id__)}" data-path="${escAttr(r.path)}">
        <div class="card-header">
          <span class="status-dot"></span>
          ${branchTag}
        </div>
        <div class="name" title="${escAttr(r.name || r.path)}">${escHtml(r.name || r.path)}</div>
        <div class="path" title="${escAttr(r.display_path || r.path)}">${escHtml(r.display_path || r.path)}</div>
        <button class="delete-btn" title="删除">🗑️</button>
      </div>
    `;
  }

  function buildTableRow(r) {
    return `
      <tr data-id="${escAttr(r.__vsc_id__)}" data-path="${escAttr(r.path)}">
        <td>
          <div style="display:flex;align-items:center;gap:8px">
            <span>📁</span>
            <span class="name" title="${escAttr(r.name || r.path)}">${escHtml(r.name || r.path)}</span>
          </div>
        </td>
        <td><span class="path" title="${escAttr(r.display_path || r.path)}">${escHtml(r.display_path || r.path)}</span></td>
        <td><button class="delete-btn" title="删除">🗑️</button></td>
      </tr>
    `;
  }

  async function handleDeleteRecord(id) {
    const confirmed = await confirmDialog("确认删除", "确定删除此项目？可在回收站恢复。");
    if (!confirmed) return;
    try {
      await API.deleteRecord(id);
      toast("已删除", "success");
      await loadRecords();
    } catch (e) {
      toast("删除失败: " + e.message, "error");
    }
  }

  async function handleOpenRecord() {
    if (!records.length) {
      toast("没有可打开的项目", "warning");
      return;
    }
    openRecord(records[0].path);
  }

  // ============================================================
  // Trash
  // ============================================================
  async function loadTrash() {
    try {
      const data = await API.getTrash();
      trash = data.trash || [];
      renderTrash();
    } catch (e) {
      toast("加载回收站失败: " + e.message, "error");
    }
  }

  function renderTrash() {
    const el = document.getElementById("trash-list");
    if (!trash.length) {
      el.innerHTML = '<div class="empty-state"><div class="empty-icon">🗑️</div><div class="empty-title">回收站为空</div><div class="empty-description">删除的项目会在此显示</div></div>';
      return;
    }
    el.innerHTML = trash.map((t) => buildTrashItem(t)).join("");
    el.querySelectorAll(".trash-item").forEach((item) => {
      const id = item.dataset.id;
      item.querySelector(".btn-restore")?.addEventListener("click", () => handleRestore(id));
      item.querySelector(".btn-perm-delete")?.addEventListener("click", () => handlePermanentDelete(id));
    });
  }

  function buildTrashItem(t) {
    const date = formatDate(t.deleted_at);
    return `
      <div class="trash-item" data-id="${escAttr(t.id)}">
        <div class="info">
          <div class="name">${escHtml(t.name || t.path)}</div>
          <div class="meta">${escHtml(t.path)} · ${date}</div>
        </div>
        <div class="actions">
          <button class="btn btn-sm btn-success btn-restore" title="恢复">↩️ 恢复</button>
          <button class="btn btn-sm btn-danger btn-perm-delete" title="永久删除">✕</button>
        </div>
      </div>
    `;
  }

  async function handleRestore(id) {
    try {
      await API.restoreRecord(id);
      toast("已恢复", "success");
      await loadTrash();
      await refreshStats();
    } catch (e) {
      toast("恢复失败: " + e.message, "error");
    }
  }

  async function handlePermanentDelete(id) {
    const confirmed = await confirmDialog("永久删除", "确定永久删除？此操作不可恢复。");
    if (!confirmed) return;
    try {
      await API.permanentDelete(id);
      toast("已永久删除", "success");
      await loadTrash();
      await refreshStats();
    } catch (e) {
      toast("删除失败: " + e.message, "error");
    }
  }

  async function handleClearTrash() {
    if (!trash.length) return;
    const confirmed = await confirmDialog("清空回收站", `确定清空回收站？将永久删除 ${trash.length} 个项目。`);
    if (!confirmed) return;
    try {
      await API.clearTrash();
      toast("回收站已清空", "success");
      await loadTrash();
      await refreshStats();
    } catch (e) {
      toast("清空失败: " + e.message, "error");
    }
  }

  // ============================================================
  // Settings
  // ============================================================
  async function loadSettings() {
    try {
      config = await API.getConfig();
      renderSettingsDirs();
    } catch (e) {
      toast("加载设置失败: " + e.message, "error");
    }
  }

  let settingsDirs = [];

  function renderSettingsDirs() {
    settingsDirs = (config.VSC_DIRECTORIES || "").split(",").filter(Boolean);
    const el = document.getElementById("dir-list");
    if (!settingsDirs.length) {
      el.innerHTML = '<div class="empty-state"><div class="empty-icon">📁</div><div class="empty-title">未配置项目目录</div><div class="empty-description">点击上方"添加目录"添加</div></div>';
      return;
    }
    el.innerHTML = settingsDirs.map((d, i) => `
      <div class="dir-list-item" data-index="${i}">
        <span class="path" title="${escAttr(d)}">${escHtml(d)}</span>
        <button class="remove-btn" title="移除">✕</button>
      </div>
    `).join("");

    el.querySelectorAll(".remove-btn").forEach((btn) => {
      btn.addEventListener("click", () => {
        const idx = parseInt(btn.closest(".dir-list-item").dataset.index, 10);
        settingsDirs.splice(idx, 1);
        saveSettingsDirs();
      });
    });
  }

  async function saveSettingsDirs() {
    const newDirs = settingsDirs.join(",");
    try {
      await API.patchConfig({ VSC_DIRECTORIES: newDirs });
      config.VSC_DIRECTORIES = newDirs;
      renderSettingsDirs();
      toast("目录已更新", "success");
    } catch (e) {
      toast("保存失败: " + e.message, "error");
      loadSettings();
    }
  }

  async function handleAddDir() {
    const path = prompt("输入目录路径（如 ~/Projects ）:");
    if (!path) return;
    const trimmed = path.trim();
    if (!trimmed) return;
    try {
      await API.addDirectory(trimmed);
      toast("目录已添加: " + trimmed, "success");
      await loadSettings();
    } catch (e) {
      toast("添加失败: " + e.message, "error");
    }
  }

  async function handleExport() {
    try {
      const [recordsData, trashData, configData] = await Promise.all([
        API.getRecords(),
        API.getTrash(),
        API.getConfig(),
      ]);
      const bundle = {
        exported_at: new Date().toISOString(),
        records: recordsData.records || [],
        trash: trashData.trash || [],
        config: configData,
      };
      const blob = new Blob([JSON.stringify(bundle, null, 2)], { type: "application/json" });
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      const date = new Date().toISOString().slice(0, 10);
      a.download = `vsc-backup-${date}.json`;
      a.click();
      URL.revokeObjectURL(url);
      toast("数据已导出", "success");
    } catch (e) {
      toast("导出失败: " + e.message, "error");
    }
  }

  async function handleImport() {
    const input = document.createElement("input");
    input.type = "file";
    input.accept = ".json";
    input.onchange = async (e) => {
      const file = e.target.files[0];
      if (!file) return;
      try {
        const text = await file.text();
        const bundle = JSON.parse(text);
        if (!bundle.records) throw new Error("无效的备份文件");
        const count = (bundle.records || []).length;
        const confirmed = await confirmDialog("导入数据", `将导入 ${count} 条记录，确定继续？`);
        if (!confirmed) return;
        await API.rebuild();
        toast(`已导入 ${count} 条记录`, "success");
        await loadRecords();
        await refreshStats();
      } catch (e) {
        toast("导入失败: " + e.message, "error");
      }
    };
    input.click();
  }

  // ============================================================
  // Search
  // ============================================================
  function handleSearch() {
    if (searchTimer) clearTimeout(searchTimer);
    searchTimer = setTimeout(() => {
      if (currentView === "records") {
        loadRecords();
      }
    }, 300);
  }

  // ============================================================
  // Rebuild
  // ============================================================
  async function handleRebuild() {
    const btn = document.getElementById("btn-rebuild");
    const origText = btn.textContent;
    btn.textContent = "🔄 重建中...";
    btn.disabled = true;
    try {
      const result = await API.rebuild();
      toast("索引已重建: " + result.total + " 条记录", "success");
      await loadRecords();
      await refreshStats();
    } catch (e) {
      toast("重建失败: " + e.message, "error");
    } finally {
      btn.textContent = origText;
      btn.disabled = false;
    }
  }

  // ============================================================
  // Open Record
  // ============================================================
  function openRecord(path) {
    const encoded = encodeURIComponent(path.replace(/^file:\/\//, ""));
    window.location.href = `vsc://file/open?path=${encoded}`;
  }

  // ============================================================
  // Helpers
  // ============================================================
  function formatDate(iso) {
    if (!iso) return "—";
    try {
      const d = new Date(iso);
      const now = new Date();
      const diff = now - d;
      const mins = Math.floor(diff / 60000);
      if (mins < 1) return "刚刚";
      if (mins < 60) return `${mins}分钟前`;
      const hours = Math.floor(mins / 60);
      if (hours < 24) return `${hours}小时前`;
      const days = Math.floor(hours / 24);
      if (days < 7) return `${days}天前`;
      return d.toLocaleDateString("zh-CN");
    } catch {
      return iso;
    }
  }

  function escHtml(s) {
    if (!s) return "";
    return String(s)
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;");
  }

  function escAttr(s) {
    if (!s) return "";
    return String(s).replace(/"/g, "&quot;").replace(/'/g, "&#39;");
  }

  // ============================================================
  // Start
  // ============================================================
  init();
})();
