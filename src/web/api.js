/* api.js — VSC Dashboard API client */

const API = {
  async _req(method, path, body) {
    const opts = {
      method,
      headers: { "Content-Type": "application/json" },
    };
    if (body !== undefined) {
      opts.body = JSON.stringify(body);
    }
    const res = await fetch(path, opts);
    const json = await res.json();
    if (!json.ok) {
      throw new Error(json.error || "Unknown error");
    }
    return json.data;
  },

  getRecords(params = {}) {
    const qs = new URLSearchParams(params).toString();
    return this._req("GET", `/api/records${qs ? "?" + qs : ""}`);
  },

  deleteRecord(id) {
    return this._req("DELETE", `/api/records/${id}`);
  },

  batchDeleteRecords(ids) {
    return this._req("POST", "/api/records/batch-delete", { ids });
  },

  reorderRecords(ids) {
    return this._req("POST", "/api/records/reorder", { ids });
  },

  getTrash() {
    return this._req("GET", "/api/trash");
  },

  restoreRecord(id) {
    return this._req("POST", `/api/trash/${id}/restore`);
  },

  restoreAll() {
    return this._req("POST", "/api/trash/restore-all");
  },

  permanentDelete(id) {
    return this._req("DELETE", `/api/trash/${id}`);
  },

  clearTrash() {
    return this._req("DELETE", "/api/trash");
  },

  rebuild() {
    return this._req("POST", "/api/rebuild");
  },

  getStats() {
    return this._req("GET", "/api/stats");
  },

  getConfig() {
    return this._req("GET", "/api/config");
  },

  patchConfig(data) {
    return this._req("PATCH", "/api/config", data);
  },

  getDirTree() {
    return this._req("GET", "/api/directories/tree");
  },

  addDirectory(path) {
    return this._req("POST", "/api/directories", { path });
  },
};
