/* app.js — VSC Dashboard main application */

(function () {
  "use strict";

  // ============================================================
  // State
  // ============================================================
  let currentView = "dashboard";
  let records = [];
  let trash = [];
  let config = {};
  let dirTree = [];
  let selectedIds = new Set();
  let viewMode = "card"; // "card" | "table"
  let searchTimer = null;
  let draggedId = null; // drag-and-drop state

  // ============================================================
  // Init
  // ============================================================
  async function init() {
    setupEventListeners();
    setupNavigation();
    await loadDashboard();
    updateServerInfo();
    // Poll stats every 30s to keep trash badge fresh
    setInterval(refreshStats, 30000);
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
    selectedIds.clear();
    updateBatchActions();
    if (view === "dashboard") loadDashboard();
    else if (view === "records") loadRecords();
    else if (view === "trash") loadTrash();
    else if (view === "dirbrowser") loadDirTree();
    else if (view === "settings") loadSettings();
  }

  // ============================================================
  // Event Listeners
  // ============================================================
  function setupEventListeners() {
    // Rebuild button
    document.getElementById("btn-rebuild").addEventListener("click", handleRebuild);
    // Open button
    document.getElementById("btn-open").addEventListener("click", handleOpenRecord);
    // Global search
    document.getElementById("global-search").addEventListener("input", handleGlobalSearch);

    // Records view controls
    document.getElementById("sort-select").addEventListener("change", loadRecords);
    document.getElementById("type-select").addEventListener("change", loadRecords);
    document.getElementById("toggle-card").addEventListener("click", () => setViewMode("card"));
    document.getElementById("toggle-table").addEventListener("click", () => setViewMode("table"));
    document.getElementById("check-all").addEventListener("change", handleCheckAll);
    document.getElementById("btn-batch-delete").addEventListener("click", handleBatchDelete);

    // Trash
    document.getElementById("btn-restore-all").addEventListener("click", handleRestoreAll);
    document.getElementById("btn-clear-trash").addEventListener("click", handleClearTrash);

    // Dir browser
    document.getElementById("btn-add-dir").addEventListener("click", showAddDirModal);

    // Settings
    document.getElementById("btn-save-config").addEventListener("click", handleSaveConfig);
    document.getElementById("btn-rebuild-settings").addEventListener("click", handleRebuild);
    document.getElementById("btn-export-data")?.addEventListener("click", handleExportData);
    document.getElementById("btn-import-data")?.addEventListener("click", () => document.getElementById("import-file-input").click());
    document.getElementById("import-file-input")?.addEventListener("change", (e) => {
      if (e.target.files[0]) handleImportData(e.target.files[0]);
    });
  }

  function setViewMode(mode) {
    viewMode = mode;
    document.getElementById("toggle-card").classList.toggle("active", mode === "card");
    document.getElementById("toggle-table").classList.toggle("active", mode === "table");
    document.getElementById("records-cards").style.display = mode === "card" ? "" : "none";
    document.getElementById("records-table-wrap").style.display = mode === "table" ? "" : "none";
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
    setTimeout(() => {
      el.style.opacity = "0";
      el.style.transition = "opacity .3s";
      setTimeout(() => el.remove(), 300);
    }, 3000);
  }

  // ============================================================
  // Dashboard
  // ============================================================
  async function loadDashboard() {
    try {
      const [stats, data] = await Promise.all([
        API.getStats(),
        API.getRecords({ limit: 5 }),
      ]);
      renderStats(stats);
      renderRecent(data.records);
    } catch (e) {
      toast("加载总览失败: " + e.message, "error");
    }
  }

  function renderStats(stats) {
    document.getElementById("stat-records").textContent = stats.total_records ?? 0;
    document.getElementById("stat-trash").textContent = stats.total_trash ?? 0;
    document.getElementById("stat-dirs").textContent = stats.watched_directories ?? 0;
    document.getElementById("stat-git").textContent = stats.git_repos ?? 0;

    // Update trash badge
    const badge = document.getElementById("trash-badge");
    if (stats.total_trash > 0) {
      badge.style.display = "";
      badge.textContent = stats.total_trash;
    } else {
      badge.style.display = "none";
    }
  }

  function renderRecent(recs) {
    const el = document.getElementById("recent-list");
    if (!recs || recs.length === 0) {
      el.innerHTML = '<div class="empty-state">暂无最近项目</div>';
      return;
    }
    el.innerHTML = recs.map((r) => buildRecentItem(r)).join("");
    el.querySelectorAll(".recent-item").forEach((item) => {
      item.addEventListener("click", () => openRecord(item.dataset.path));
    });
  }

  function buildRecentItem(r) {
    const icon = getTypeIcon(r.path);
    const branch = r.branch
      ? `<span class="branch-badge">⎇ ${escHtml(r.branch)}</span>`
      : `<span class="branch-badge none">—</span>`;
    return `
      <div class="recent-item" data-path="${escAttr(r.path)}">
        <span class="recent-icon">${icon}</span>
        <div class="recent-info">
          <div class="recent-name">${escHtml(r.name || r.path)}</div>
          <div class="recent-meta">
            ${branch}
            <span>${escHtml(r.display_path || r.path)}</span>
          </div>
        </div>
      </div>
    `;
  }

  async function refreshStats() {
    try {
      const stats = await API.getStats();
      renderStats(stats);
    } catch (_) {}
  }

  // ============================================================
  // Records
  // ============================================================
  async function loadRecords() {
    const sort = document.getElementById("sort-select")?.value || "mru";
    const type = document.getElementById("type-select")?.value || "all";
    const q = document.getElementById("global-search")?.value || "";

    try {
      const data = await API.getRecords({ sort, type, q });
      records = data.records;
      renderRecords();
    } catch (e) {
      toast("加载项目失败: " + e.message, "error");
    }
  }

  function renderRecords() {
    if (viewMode === "card") {
      renderRecordCards();
    } else {
      renderRecordTable();
    }
  }

  function renderRecordCards() {
    const el = document.getElementById("records-cards");
    if (!records.length) {
      el.innerHTML = '<div class="empty-state">暂无项目</div>';
      return;
    }
    el.innerHTML = records.map((r) => buildRecordCard(r)).join("");
    el.querySelectorAll(".record-card").forEach((card) => {
      card.addEventListener("click", (e) => {
        if (e.target.closest(".record-check") || e.target.closest(".record-actions")) return;
        openRecord(card.dataset.path);
      });
      card.querySelector(".record-check")?.addEventListener("change", (e) => {
        toggleSelect(card.dataset.id, e.target.checked);
      });
      card.querySelector(".btn-delete-record")?.addEventListener("click", () => handleDeleteRecord(card.dataset.id));

      // Drag-and-drop
      card.addEventListener("dragstart", (e) => {
        draggedId = card.dataset.id;
        card.classList.add("dragging");
        e.dataTransfer.effectAllowed = "move";
      });
      card.addEventListener("dragend", () => {
        card.classList.remove("dragging");
        draggedId = null;
        el.querySelectorAll(".drag-over").forEach((c) => c.classList.remove("drag-over"));
      });
      card.addEventListener("dragover", (e) => {
        e.preventDefault();
        e.dataTransfer.dropEffect = "move";
        if (card.dataset.id !== draggedId) card.classList.add("drag-over");
      });
      card.addEventListener("dragleave", () => {
        card.classList.remove("drag-over");
      });
      card.addEventListener("drop", (e) => {
        e.preventDefault();
        card.classList.remove("drag-over");
        const targetId = card.dataset.id;
        if (!draggedId || draggedId === targetId) return;
        const fromIdx = records.findIndex((r) => r.__vsc_id__ === draggedId);
        const toIdx = records.findIndex((r) => r.__vsc_id__ === targetId);
        if (fromIdx < 0 || toIdx < 0) return;
        const [moved] = records.splice(fromIdx, 1);
        records.splice(toIdx, 0, moved);
        API.reorderRecords(records.map((r) => r.__vsc_id__)).catch((err) => toast("排序失败: " + err.message, "error"));
        renderRecordCards();
      });
    });
  }

  function buildRecordCard(r) {
    const icon = getTypeIcon(r.path);
    const checked = selectedIds.has(r.__vsc_id__) ? "checked" : "";
    const branch = r.branch
      ? `<span class="branch-badge">⎇ ${escHtml(r.branch)}</span>`
      : `<span class="branch-badge none">—</span>`;
    return `
      <div class="record-card" data-id="${escAttr(r.__vsc_id__)}" data-path="${escAttr(r.path)}">
        <input type="checkbox" class="record-check" ${checked} />
        <div class="record-card-header">
          <span class="record-type-icon">${icon}</span>
        </div>
        <div class="record-name" title="${escAttr(r.name)}">${escHtml(r.name)}</div>
        <div class="record-path" title="${escAttr(r.display_path || r.path)}">${escHtml(r.display_path || r.path)}</div>
        <div class="record-footer">
          ${branch}
          <div class="record-actions">
            <button class="btn-icon btn-delete-record" title="删除">🗑️</button>
          </div>
        </div>
      </div>
    `;
  }

  function renderRecordTable() {
    const el = document.getElementById("records-tbody");
    if (!records.length) {
      el.innerHTML = '<tr><td colspan="5" class="empty-state">暂无项目</td></tr>';
      return;
    }
    el.innerHTML = records.map((r) => buildTableRow(r)).join("");
    el.querySelectorAll("tr").forEach((row) => {
      const id = row.dataset.id;
      row.querySelector(".row-check")?.addEventListener("change", (e) => {
        toggleSelect(id, e.target.checked);
      });
      row.querySelector(".btn-delete-record")?.addEventListener("click", () => handleDeleteRecord(id));

      // Drag-and-drop
      row.setAttribute("draggable", "true");
      row.addEventListener("dragstart", (e) => {
        draggedId = id;
        row.classList.add("dragging");
        e.dataTransfer.effectAllowed = "move";
      });
      row.addEventListener("dragend", () => {
        row.classList.remove("dragging");
        draggedId = null;
        el.querySelectorAll(".drag-over").forEach((r) => r.classList.remove("drag-over"));
      });
      row.addEventListener("dragover", (e) => {
        e.preventDefault();
        e.dataTransfer.dropEffect = "move";
        if (id !== draggedId) row.classList.add("drag-over");
      });
      row.addEventListener("dragleave", () => {
        row.classList.remove("drag-over");
      });
      row.addEventListener("drop", (e) => {
        e.preventDefault();
        row.classList.remove("drag-over");
        if (!draggedId || draggedId === id) return;
        const fromIdx = records.findIndex((r) => r.__vsc_id__ === draggedId);
        const toIdx = records.findIndex((r) => r.__vsc_id__ === id);
        if (fromIdx < 0 || toIdx < 0) return;
        const [moved] = records.splice(fromIdx, 1);
        records.splice(toIdx, 0, moved);
        API.reorderRecords(records.map((r) => r.__vsc_id__)).catch((err) => toast("排序失败: " + err.message, "error"));
        renderRecordTable();
      });
    });
  }

  function buildTableRow(r) {
    const icon = getTypeIcon(r.path);
    const checked = selectedIds.has(r.__vsc_id__) ? "checked" : "";
    const branch = r.branch ? `<span class="branch-badge">⎇ ${escHtml(r.branch)}</span>` : `<span class="branch-badge none">—</span>`;
    return `
      <tr data-id="${escAttr(r.__vsc_id__)}">
        <td><input type="checkbox" class="row-check" ${checked} /></td>
        <td>
          <div style="display:flex;align-items:center;gap:8px">
            <span>${icon}</span>
            <span class="table-name" title="${escAttr(r.name)}">${escHtml(r.name)}</span>
          </div>
        </td>
        <td>${branch}</td>
        <td><span class="table-path" title="${escAttr(r.display_path || r.path)}">${escHtml(r.display_path || r.path)}</span></td>
        <td>
          <button class="btn btn-sm btn-danger btn-delete-record">🗑️ 删除</button>
        </td>
      </tr>
    `;
  }

  function toggleSelect(id, checked) {
    if (checked) selectedIds.add(id);
    else selectedIds.delete(id);

    // Update card check state
    const card = document.querySelector(`.record-card[data-id="${id}"]`);
    if (card) {
      card.classList.toggle("selected", checked);
      card.querySelector(".record-check").checked = checked;
    }
    // Update table row check
    const row = document.querySelector(`tr[data-id="${id}"]`);
    if (row) row.querySelector(".row-check").checked = checked;

    updateBatchActions();
  }

  function updateBatchActions() {
    const count = selectedIds.size;
    const el = document.getElementById("batch-actions");
    const countEl = document.getElementById("batch-count");
    if (count > 0) {
      el.style.display = "flex";
      countEl.textContent = `已选择 ${count} 项`;
    } else {
      el.style.display = "none";
    }
  }

  function handleCheckAll() {
    const checked = document.getElementById("check-all").checked;
    if (checked) {
      records.forEach((r) => selectedIds.add(r.__vsc_id__));
    } else {
      selectedIds.clear();
    }
    renderRecords();
    updateBatchActions();
  }

  async function handleBatchDelete() {
    if (selectedIds.size === 0) return;
    if (!confirm(`确定删除选中的 ${selectedIds.size} 个项目？`)) return;
    try {
      await API.batchDeleteRecords([...selectedIds]);
      toast(`已删除 ${selectedIds.size} 个项目`, "success");
      selectedIds.clear();
      updateBatchActions();
      await loadRecords();
      await refreshStats();
    } catch (e) {
      toast("删除失败: " + e.message, "error");
    }
  }

  async function handleDeleteRecord(id) {
    try {
      await API.deleteRecord(id);
      toast("已删除", "success");
      await loadRecords();
      await refreshStats();
    } catch (e) {
      toast("删除失败: " + e.message, "error");
    }
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
      el.innerHTML = '<div class="empty-state">回收站为空</div>';
      return;
    }
    el.innerHTML = trash.map((t) => buildTrashItem(t)).join("");
    el.querySelectorAll(".btn-restore").forEach((btn) => {
      btn.addEventListener("click", () => handleRestore(btn.dataset.id));
    });
    el.querySelectorAll(".btn-perm-delete").forEach((btn) => {
      btn.addEventListener("click", () => handlePermanentDelete(btn.dataset.id));
    });
  }

  function buildTrashItem(t) {
    const date = formatDate(t.deleted_at);
    return `
      <div class="trash-item">
        <div class="trash-info">
          <div class="trash-name">${escHtml(t.name || t.path)}</div>
          <div class="trash-meta">${escHtml(t.path)} · ${date}</div>
        </div>
        <div class="trash-actions">
          <button class="btn btn-sm btn-success btn-restore" data-id="${escAttr(t.id)}">↩️ 恢复</button>
          <button class="btn btn-sm btn-danger btn-perm-delete" data-id="${escAttr(t.id)}">✕ 永久删除</button>
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
    if (!confirm("确定永久删除？此操作不可恢复。")) return;
    try {
      await API.permanentDelete(id);
      toast("已永久删除", "success");
      await loadTrash();
      await refreshStats();
    } catch (e) {
      toast("删除失败: " + e.message, "error");
    }
  }

  async function handleRestoreAll() {
    if (!confirm("确定恢复全部？")) return;
    try {
      await API.restoreAll();
      toast("已全部恢复", "success");
      await loadTrash();
      await refreshStats();
    } catch (e) {
      toast("恢复失败: " + e.message, "error");
    }
  }

  async function handleClearTrash() {
    if (!confirm("确定清空回收站？此操作不可恢复。")) return;
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
  // Directory Browser
  // ============================================================
  async function loadDirTree() {
    try {
      const data = await API.getDirTree();
      dirTree = data.roots || [];
      renderDirTree();
    } catch (e) {
      toast("加载目录失败: " + e.message, "error");
    }
  }

  function renderDirTree() {
    const el = document.getElementById("dir-tree");
    if (!dirTree.length) {
      el.innerHTML = '<div class="empty-state">未配置项目目录 (VSC_DIRECTORIES)</div>';
      return;
    }
    el.innerHTML = '<div class="tree-root" id="tree-root"></div>';
    const root = document.getElementById("tree-root");
    dirTree.forEach((r) => root.appendChild(buildTreeNode(r)));
  }

  function buildTreeNode(node) {
    const children = node.children && node.children.length > 0;
    const item = document.createElement("div");
    item.className = "tree-node";

    const itemEl = document.createElement("div");
    itemEl.className = "tree-item";
    itemEl.dataset.path = node.path;

    const expand = children
      ? `<span class="tree-expand" data-open="false">▶</span>`
      : `<span class="tree-expand" style="visibility:hidden">▶</span>`;

    itemEl.innerHTML = `
      ${expand}
      <span class="tree-icon">📁</span>
      <span class="tree-name" title="${escAttr(node.display_path)}">${escHtml(node.name)}</span>
      <span class="tree-add" title="添加为项目">+添加</span>
    `;

    itemEl.querySelector(".tree-expand")?.addEventListener("click", (e) => {
      e.stopPropagation();
      const el = e.currentTarget;
      const isOpen = el.dataset.open === "true";
      el.dataset.open = (!isOpen).toString();
      el.textContent = isOpen ? "▶" : "▼";
      const child = item.querySelector(".tree-children");
      if (child) child.style.display = isOpen ? "none" : "block";
    });

    itemEl.addEventListener("dblclick", () => openRecord(node.path));

    itemEl.querySelector(".tree-add")?.addEventListener("click", (e) => {
      e.stopPropagation();
      handleAddDirectory(node.path);
    });

    if (children) {
      const childContainer = document.createElement("div");
      childContainer.className = "tree-children";
      childContainer.style.display = "none";
      node.children.forEach((c) => childContainer.appendChild(buildTreeNode(c)));
      item.appendChild(childContainer);
    }

    item.appendChild(itemEl);
    return item;
  }

  async function handleAddDirectory(path) {
    try {
      await API.addDirectory(path);
      toast("目录已添加: " + path, "success");
      await loadDirTree();
    } catch (e) {
      toast("添加失败: " + e.message, "error");
    }
  }

  function showAddDirModal() {
    const overlay = document.createElement("div");
    overlay.className = "modal-overlay";
    overlay.innerHTML = `
      <div class="modal">
        <div class="modal-title">添加目录</div>
        <input type="text" class="modal-input" id="modal-dir-input" placeholder="输入目录路径，如 ~/Projects" />
        <div class="modal-actions">
          <button class="btn btn-secondary" id="modal-cancel">取消</button>
          <button class="btn btn-primary" id="modal-confirm">添加</button>
        </div>
      </div>
    `;
    document.body.appendChild(overlay);

    overlay.querySelector("#modal-cancel").addEventListener("click", () => overlay.remove());
    overlay.querySelector("#modal-confirm").addEventListener("click", () => {
      const val = document.getElementById("modal-dir-input").value.trim();
      if (val) {
        handleAddDirectory(val);
        overlay.remove();
      }
    });
    overlay.addEventListener("click", (e) => {
      if (e.target === overlay) overlay.remove();
    });
    setTimeout(() => document.getElementById("modal-dir-input").focus(), 100);
  }

  // ============================================================
  // Settings
  // ============================================================
  async function loadSettings() {
    try {
      config = await API.getConfig();
      renderSettings();
    } catch (e) {
      toast("加载设置失败: " + e.message, "error");
    }
  }

  // ---- Settings ----
  let settingsDirs = []; // live list editable in settings UI

  function renderSettings() {
    settingsDirs = (config.VSC_DIRECTORIES || "").split(",").filter(Boolean);
    renderSettingsDirs();
    document.getElementById("setting-open-default").value = config.VSC_OPEN_DEFAULT || "";
    document.getElementById("setting-open-cmd").value = config.VSC_OPEN_WITH_CMD || "";
    document.getElementById("setting-cache-file").textContent = config.cache_file || "";
  }

  function renderSettingsDirs() {
    const el = document.getElementById("settings-dirs");
    if (!settingsDirs.length) {
      el.innerHTML = '<div class="empty-state">未配置项目目录</div>';
      return;
    }
    el.innerHTML = `
      ${settingsDirs.map((d, i) => `
        <div class="dir-list-item settings-dir-item" data-index="${i}">
          <code>${escHtml(d)}</code>
          <button class="btn-remove-dir" data-index="${i}" title="移除">✕</button>
        </div>
      `).join("")}
      <div class="dir-add-row">
        <input type="text" id="setting-dir-input" class="setting-input" placeholder="输入目录路径，如 ~/Projects" />
        <button class="btn btn-secondary" id="btn-add-settings-dir">+ 添加</button>
      </div>
    `;
    el.querySelectorAll(".btn-remove-dir").forEach((btn) => {
      btn.addEventListener("click", () => {
        const idx = parseInt(btn.dataset.index, 10);
        settingsDirs.splice(idx, 1);
        config.VSC_DIRECTORIES = settingsDirs.join(",");
        renderSettingsDirs();
      });
    });
    document.getElementById("btn-add-settings-dir")?.addEventListener("click", () => {
      const input = document.getElementById("setting-dir-input");
      const val = input.value.trim();
      if (val && !settingsDirs.includes(val)) {
        settingsDirs.push(val);
        config.VSC_DIRECTORIES = settingsDirs.join(",");
        renderSettingsDirs();
      }
    });
  }

  async function handleSaveConfig() {
    try {
      await API.patchConfig({ VSC_DIRECTORIES: config.VSC_DIRECTORIES });
      toast("配置已保存", "success");
    } catch (e) {
      toast("保存失败: " + e.message, "error");
    }
  }

  // Export data
  async function handleExportData() {
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

  // Import data
  async function handleImportData(file) {
    try {
      const text = await file.text();
      const bundle = JSON.parse(text);
      if (!bundle.records) throw new Error("Invalid backup file");
      // Show confirmation
      const count = (bundle.records || []).length;
      if (!confirm(`确定导入 ${count} 条记录？此操作将覆盖现有索引。`)) return;
      // Rebuild with imported records
      const result = await API.rebuild();
      toast(`已导入 ${count} 条记录`, "success");
      await loadRecords();
      await refreshStats();
    } catch (e) {
      toast("导入失败: " + e.message, "error");
    }
  }

  // ============================================================
  // Global Search
  // ============================================================
  function handleGlobalSearch() {
    if (searchTimer) clearTimeout(searchTimer);
    searchTimer = setTimeout(() => {
      if (currentView === "records" || currentView === "dashboard") {
        loadRecords();
      }
    }, 300);
  }

  // ============================================================
  // Rebuild
  // ============================================================
  async function handleRebuild() {
    const btn = document.getElementById("btn-rebuild");
    btn.textContent = "🔄 重建中...";
    btn.disabled = true;
    try {
      const result = await API.rebuild();
      toast("索引已重建: " + result.total + " 条记录", "success");
      await loadRecords();
      await loadDashboard();
      await refreshStats();
    } catch (e) {
      toast("重建失败: " + e.message, "error");
    } finally {
      btn.textContent = "🔄 重建";
      btn.disabled = false;
    }
  }

  // ============================================================
  // Open record
  // ============================================================
  function openRecord(path) {
    // Use VSCode URL scheme to open directly from browser
    // vsc://file/open?path=%2FUsers%2Fuser%2Fproject
    const encoded = encodeURIComponent(path.replace(/^file:\/\//, ""));
    window.location.href = `vsc://file/open?path=${encoded}`;
  }

  async function handleOpenRecord() {
    // Open first selected or first record
    let path;
    if (selectedIds.size > 0) {
      const id = [...selectedIds][0];
      const rec = records.find((r) => r.__vsc_id__ === id);
      path = rec?.path;
    } else {
      path = records[0]?.path;
    }
    if (path) {
      openRecord(path);
    } else {
      toast("没有可打开的项目", "warning");
    }
  }

  // ============================================================
  // Helpers
  // ============================================================
  function getTypeIcon(path) {
    if (!path) return "📁";
    if (path.startsWith("http") || path.includes("://")) return "🌐";
    // We'll check path existence in a smarter way
    // For display purposes, folders end with / or are dirs
    if (path.startsWith("file://")) {
      // can't check fs from browser, use heuristic
    }
    return "📁";
  }

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

  function updateServerInfo() {
    const url = `${window.location.host}`;
    document.getElementById("server-url").textContent = url;
  }

  // ============================================================
  // Start
  // ============================================================
  init();
})();
