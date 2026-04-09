#!/usr/bin/env python3
"""
VSC HTTP Server — Visual Management Interface
Serves a local web UI for managing VSC records.
"""

import html
import json
import os
import re
import socketserver
import sys
import threading
from http.server import HTTPServer, SimpleHTTPRequestHandler
from pathlib import Path
from urllib.parse import urlparse, parse_qs

# Add parent dir to path so we can import shared
sys.path.insert(0, str(Path(__file__).parent))
from shared import (
    CACHE_FILE,
    USER_CONFIG_FILE,
    ENV_DIRECTORIES,
    ENV_IDE_PATH,
    HOME,
    PATH_TYPE_FILE,
    PATH_TYPE_FOLDER,
    PATH_TYPE_REMOTE,
    get_vsc_directory_roots,
    get_git_branch,
    log_info,
    log_error,
    read_cache,
    write_cache,
    get_cache_default,
    generate_record_id,
    make_record,
    remove_path_scheme,
    scan_subdirectories,
)

# ============================================================
# Port Management
# ============================================================

PORT_DEFAULT = 3847
PORT_MIN = 3847
PORT_MAX = 65535

_lock_file = CACHE_FILE.parent / ".serve.port"


def get_saved_port():
    """Read previously saved port from lock file."""
    try:
        if _lock_file.exists():
            return int(_lock_file.read_text().strip())
    except Exception:
        pass
    return None


def save_port(port):
    """Save the used port to lock file."""
    try:
        _lock_file.write_text(str(port))
    except Exception:
        pass


def find_available_port(start=PORT_DEFAULT):
    """Find an available port starting from start."""
    import socket
    for port in range(start, PORT_MAX + 1):
        try:
            s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
            s.bind(("0.0.0.0", port))
            s.close()
            return port
        except OSError:
            continue
    return None


# ============================================================
# API Helpers
# ============================================================

def json_response(handler, data, status=200):
    """Send a JSON response."""
    handler.send_response(status)
    handler.send_header("Content-Type", "application/json")
    handler.send_header("Access-Control-Allow-Origin", "*")
    handler.send_header("Access-Control-Allow-Methods", "GET, POST, DELETE, PATCH, OPTIONS")
    handler.send_header("Access-Control-Allow-Headers", "Content-Type")
    handler.end_headers()
    handler.wfile.write(json.dumps(data, ensure_ascii=False).encode("utf-8"))


def ok(data):
    return {"ok": True, "data": data}


def err(msg):
    return {"ok": False, "error": msg}


def get_record_by_id(record_id):
    """Find a record by its __vsc_id__."""
    cache = read_cache()
    for r in cache.get("records", []):
        if r.get("__vsc_id__") == record_id:
            return r
    return None


def shrink_path(path):
    """Replace $HOME with ~."""
    home = str(HOME)
    if path.startswith(home):
        return path.replace(home, "~", 1)
    return path


# ============================================================
# Route Handlers
# ============================================================

def handle_api_records(handler):
    """GET /api/records — list all records with optional filtering."""
    parsed = urlparse(handler.path)
    params = parse_qs(parsed.query)

    cache = read_cache()
    records = cache.get("records", [])
    branches = cache.get("branches", {})

    # Filter by type
    rec_type = params.get("type", ["all"])[0]
    if rec_type and rec_type != "all":
        filtered = []
        for r in records:
            p = remove_path_scheme(r.get("path", ""))
            if rec_type == PATH_TYPE_FOLDER and Path(p).is_dir():
                filtered.append(r)
            elif rec_type == PATH_TYPE_FILE and Path(p).is_file():
                filtered.append(r)
            elif rec_type == PATH_TYPE_REMOTE and "://" in r.get("path", ""):
                filtered.append(r)
        records = filtered

    # Search filter
    q = params.get("q", [""])[0].lower()
    if q:
        records = [r for r in records if q in r.get("name", "").lower() or q in r.get("path", "").lower()]

    # Sort
    sort = params.get("sort", ["mru"])[0]
    if sort == "name":
        records.sort(key=lambda r: r.get("name", "").lower())
    elif sort == "path":
        records.sort(key=lambda r: r.get("path", ""))
    # default: mru — keep as-is (already sorted by mru in cache)

    # Enrich with branch info — eagerly look up missing branches
    for r in records:
        rid = r.get("__vsc_id__", "")
        branch = branches.get(rid)
        if not branch:
            # Try to get branch directly from the path
            path = r.get("path", "")
            branch = get_git_branch(path) if path else None
            if branch:
                branches[rid] = branch
        r["branch"] = branch or ""
        r["display_path"] = shrink_path(remove_path_scheme(r.get("path", "")))

    return ok({
        "records": records,
        "total": len(records),
    }), 200


def handle_delete_record(handler, record_id):
    """DELETE /api/records/<id> — soft delete (move to trash)."""
    from datetime import datetime
    cache = read_cache()

    record = get_record_by_id(record_id)
    if not record:
        return err("Record not found"), 404

    # Move to trash
    cache.setdefault("trash", {})[record_id] = {
        "path": record["path"],
        "deleted_at": datetime.now().isoformat(),
        "name": record.get("name", ""),
    }

    # Remove from records
    cache["records"] = [r for r in cache.get("records", []) if r.get("__vsc_id__") != record_id]

    write_cache(cache)
    return ok({"deleted": record_id}), 200


def handle_batch_delete(handler):
    """POST /api/records/batch-delete — delete multiple records."""
    from datetime import datetime
    length = int(handler.headers.get("Content-Length", 0))
    body = handler.rfile.read(length).decode("utf-8")
    try:
        data = json.loads(body)
    except Exception:
        return err("Invalid JSON"), 400

    ids = data.get("ids", [])
    if not ids:
        return err("No ids provided"), 400

    cache = read_cache()
    removed = []
    for record_id in ids:
        for r in cache.get("records", []):
            if r.get("__vsc_id__") == record_id:
                cache.setdefault("trash", {})[record_id] = {
                    "path": r["path"],
                    "deleted_at": datetime.now().isoformat(),
                    "name": r.get("name", ""),
                }
                removed.append(record_id)
                break

    cache["records"] = [r for r in cache.get("records", []) if r.get("__vsc_id__") not in removed]
    write_cache(cache)
    return ok({"deleted": removed}), 200


def handle_reorder_records(handler):
    """POST /api/records/reorder — reorder records by passing ordered ids."""
    length = int(handler.headers.get("Content-Length", 0))
    body = handler.rfile.read(length).decode("utf-8")
    try:
        data = json.loads(body)
    except Exception:
        return err("Invalid JSON"), 400

    ids = data.get("ids", [])
    if not ids:
        return err("No ids provided"), 400

    cache = read_cache()
    all_records = cache.get("records", [])
    id_to_record = {r.get("__vsc_id__"): r for r in all_records}

    # Reorder
    ordered = [id_to_record[i] for i in ids if i in id_to_record]
    # Append any records not in ids (shouldn't happen but safety)
    remaining = [r for r in all_records if r.get("__vsc_id__") not in ids]
    cache["records"] = ordered + remaining
    write_cache(cache)
    return ok({"reordered": len(ordered)}), 200


def handle_get_trash(handler):
    """GET /api/trash — list trash items."""
    cache = read_cache()
    trash = cache.get("trash", {})
    items = [{"id": k, **v} for k, v in trash.items()]
    # Sort by deleted_at descending
    items.sort(key=lambda x: x.get("deleted_at", ""), reverse=True)
    return ok({"trash": items, "total": len(items)}), 200


def handle_restore_record(handler, record_id):
    """POST /api/trash/<id>/restore — restore from trash."""
    cache = read_cache()
    trash = cache.get("trash", {})
    if record_id not in trash:
        return err("Trash item not found"), 404

    item = trash.pop(record_id)
    # Reconstruct record
    record = make_record(item["path"], "restore")
    record["__vsc_id__"] = record_id
    if item.get("name"):
        record["name"] = item["name"]

    # Insert at top of records
    cache.setdefault("records", []).insert(0, record)
    write_cache(cache)
    return ok({"restored": record_id}), 200


def handle_permanent_delete(handler, record_id):
    """DELETE /api/trash/<id> — permanently delete a trash item."""
    cache = read_cache()
    if record_id in cache.get("trash", {}):
        cache["trash"].pop(record_id)
        write_cache(cache)
        return ok({"permanent_deleted": record_id}), 200
    return err("Trash item not found"), 404


def handle_clear_trash(handler):
    """DELETE /api/trash — clear all trash."""
    cache = read_cache()
    count = len(cache.get("trash", {}))
    cache["trash"] = {}
    write_cache(cache)
    return ok({"cleared": count}), 200


def handle_restore_all(handler):
    """POST /api/trash/restore-all — restore all trash items."""
    cache = read_cache()
    trash = cache.get("trash", {})
    restored = []
    for record_id, item in list(trash.items()):
        record = make_record(item["path"], "restore")
        record["__vsc_id__"] = record_id
        if item.get("name"):
            record["name"] = item["name"]
        cache.setdefault("records", []).insert(0, record)
        restored.append(record_id)
    cache["trash"] = {}
    write_cache(cache)
    return ok({"restored": restored}), 200


def handle_rebuild(handler):
    """POST /api/rebuild — rebuild the record index."""
    from shared import get_records_from_directories, get_records_from_vscode_menu
    from datetime import datetime

    log_info("Web UI: rebuild triggered")

    try:
        dir_records = get_records_from_directories()
        vscode_records = get_records_from_vscode_menu()

        seen_paths = set()
        unique_records = []
        for record in dir_records + vscode_records:
            if record["path"] not in seen_paths:
                seen_paths.add(record["path"])
                unique_records.append(record)

        cache = read_cache()
        trash_ids = set(cache.get("trash", {}).keys())
        unique_records = [r for r in unique_records if r.get("__vsc_id__") not in trash_ids]

        cache["records"] = unique_records
        write_cache(cache)

        return ok({
            "total": len(unique_records),
            "message": f"Rebuilt index: {len(unique_records)} records"
        }), 200
    except Exception as e:
        log_error(f"Rebuild error: {e}")
        return err(str(e)), 500


def handle_get_stats(handler):
    """GET /api/stats — return dashboard statistics."""
    cache = read_cache()
    records = cache.get("records", [])
    trash = cache.get("trash", {})
    branches = cache.get("branches", {})

    # Count git repos by checking .git directory/file
    def is_git_repo(path):
        p = Path(remove_path_scheme(path)) if path else None
        if not p:
            return False
        return (p / ".git").is_dir() or (p / ".git").is_file()

    git_count = sum(1 for r in records if is_git_repo(r.get("path", "")))

    # Source breakdown
    sources = {}
    for r in records:
        src = r.get("source", "unknown")
        sources[src] = sources.get(src, 0) + 1

    # Directory distribution
    dir_count = len(get_vsc_directory_roots())

    return ok({
        "total_records": len(records),
        "total_trash": len(trash),
        "watched_directories": dir_count,
        "git_repos": git_count,
        "sources": sources,
    }), 200


def handle_get_config(handler):
    """GET /api/config — return current config."""
    return ok({
        "VSC_DIRECTORIES": ENV_DIRECTORIES,
        "VSC_OPEN_DEFAULT": os.environ.get("VSC_OPEN_DEFAULT", "/usr/local/bin/code"),
        "VSC_OPEN_WITH_CMD": os.environ.get("VSC_OPEN_WITH_CMD", ""),
        "VSC_IDE_PATH": ENV_IDE_PATH,
        "cache_file": str(CACHE_FILE),
    }), 200


def handle_patch_config(handler):
    """PATCH /api/config — update config (env vars for this session)."""
    length = int(handler.headers.get("Content-Length", 0))
    body = handler.rfile.read(length).decode("utf-8")
    try:
        data = json.loads(body)
    except Exception:
        return err("Invalid JSON"), 400

    # We can only persist VSC_DIRECTORIES to the config file
    dirs = data.get("VSC_DIRECTORIES", "")
    if dirs is not None:
        os.environ["VSC_DIRECTORIES"] = dirs
        # Also save to user config file
        try:
            config = {}
            if USER_CONFIG_FILE.exists():
                config = json.loads(USER_CONFIG_FILE.read_text())
            config["VSC_DIRECTORIES"] = dirs
            USER_CONFIG_FILE.parent.mkdir(parents=True, exist_ok=True)
            USER_CONFIG_FILE.write_text(json.dumps(config, indent=2))
        except Exception as e:
            return err(f"Failed to persist config: {e}"), 500

    return ok({"message": "Config updated (restart server to take full effect)"}), 200


def handle_get_dir_tree(handler):
    """GET /api/directories/tree — return directory tree for VSC_DIRECTORIES."""
    roots = get_vsc_directory_roots()
    from shared import SKIP_DIRS

    def build_tree(path_obj, depth=0, max_depth=3):
        if depth > max_depth:
            return None
        try:
            items = []
            for item in sorted(path_obj.iterdir()):
                if item.name.startswith(".") or item.name in SKIP_DIRS:
                    continue
                if item.is_dir():
                    children = build_tree(item, depth + 1, max_depth)
                    items.append({
                        "name": item.name,
                        "path": str(item),
                        "display_path": shrink_path(str(item)),
                        "children": children,
                    })
            return items
        except PermissionError:
            return []

    trees = []
    for root in roots:
        trees.append({
            "name": root.name,
            "path": str(root),
            "display_path": shrink_path(str(root)),
            "children": build_tree(root),
        })

    return ok({"roots": trees}), 200


def handle_add_directory(handler):
    """POST /api/directories — add a path to VSC_DIRECTORIES."""
    length = int(handler.headers.get("Content-Length", 0))
    body = handler.rfile.read(length).decode("utf-8")
    try:
        data = json.loads(body)
    except Exception:
        return err("Invalid JSON"), 400

    new_path = data.get("path", "").strip()
    if not new_path:
        return err("No path provided"), 400

    # Resolve relative paths
    if new_path.startswith("~/"):
        new_path = str(HOME / new_path[2:])

    if not Path(new_path).exists():
        return err("Path does not exist"), 400

    # Append to VSC_DIRECTORIES
    current = ENV_DIRECTORIES
    paths = [p.strip() for p in current.split(",") if p.strip()] if current else []
    if new_path not in paths:
        paths.append(new_path)
    new_dirs = ",".join(paths)

    os.environ["VSC_DIRECTORIES"] = new_dirs
    try:
        config = {}
        if USER_CONFIG_FILE.exists():
            config = json.loads(USER_CONFIG_FILE.read_text())
        config["VSC_DIRECTORIES"] = new_dirs
        USER_CONFIG_FILE.parent.mkdir(parents=True, exist_ok=True)
        USER_CONFIG_FILE.write_text(json.dumps(config, indent=2))
    except Exception as e:
        return err(f"Failed to persist: {e}"), 500

    return ok({"VSC_DIRECTORIES": new_dirs}), 200


# ============================================================
# Request Handler
# ============================================================

WEB_DIR = Path(__file__).parent
INDEX_FILE = WEB_DIR / "index.html"


class Handler(SimpleHTTPRequestHandler):
    """HTTP request handler with API routing."""

    def do_OPTIONS(self):
        self.send_response(200)
        self.send_header("Access-Control-Allow-Origin", "*")
        self.send_header("Access-Control-Allow-Methods", "GET, POST, DELETE, PATCH, OPTIONS")
        self.send_header("Access-Control-Allow-Headers", "Content-Type")
        self.end_headers()

    def do_GET(self):
        parsed = urlparse(self.path)
        path = parsed.path

        # API routes
        if path == "/api/records" or path.startswith("/api/records?"):
            data, status = handle_api_records(self)
            return json_response(self, data, status)

        if path.startswith("/api/records/"):
            # /api/records/<id> — just return record info for now
            record_id = path.split("/")[3]
            r = get_record_by_id(record_id)
            if r:
                data, status = ok(r), 200
            else:
                data, status = err("Not found"), 404
            return json_response(self, data, status)

        if path == "/api/trash":
            data, status = handle_get_trash(self)
            return json_response(self, data, status)

        if path == "/api/rebuild":
            data, status = handle_rebuild(self)
            return json_response(self, data, status)

        if path == "/api/stats":
            data, status = handle_get_stats(self)
            return json_response(self, data, status)

        if path == "/api/config":
            data, status = handle_get_config(self)
            return json_response(self, data, status)

        if path == "/api/directories/tree":
            data, status = handle_get_dir_tree(self)
            return json_response(self, data, status)

        # Serve static files
        if path == "/" or path == "/index.html":
            file_path = INDEX_FILE
        else:
            file_path = WEB_DIR / path.lstrip("/")

        if file_path.exists() and file_path.is_file():
            ext = file_path.suffix.lower()
            mime_types = {
                ".html": "text/html",
                ".css": "text/css",
                ".js": "application/javascript",
                ".json": "application/json",
                ".png": "image/png",
                ".svg": "image/svg+xml",
                ".ico": "image/x-icon",
            }
            mime = mime_types.get(ext, "application/octet-stream")
            self.send_response(200)
            self.send_header("Content-Type", mime)
            self.send_header("Access-Control-Allow-Origin", "*")
            self.end_headers()
            self.wfile.write(file_path.read_bytes())
        else:
            self.send_response(404)
            self.send_header("Content-Type", "text/plain")
            self.end_headers()
            self.wfile.write(b"Not found")

    def do_POST(self):
        parsed = urlparse(self.path)
        path = parsed.path

        if path == "/api/records/reorder":
            data, status = handle_reorder_records(self)
            return json_response(self, data, status)

        if path == "/api/records/batch-delete":
            data, status = handle_batch_delete(self)
            return json_response(self, data, status)

        if path == "/api/rebuild":
            data, status = handle_rebuild(self)
            return json_response(self, data, status)

        if path == "/api/trash/restore-all":
            data, status = handle_restore_all(self)
            return json_response(self, data, status)

        if path == "/api/directories":
            data, status = handle_add_directory(self)
            return json_response(self, data, status)

        if path.startswith("/api/trash/") and "/restore" in path:
            # /api/trash/<id>/restore
            parts = path.split("/")
            if len(parts) >= 5:
                record_id = parts[3]
                data, status = handle_restore_record(self, record_id)
                return json_response(self, data, status)

        # Default: 404
        json_response(self, err("Unknown endpoint"), 404)

    def do_DELETE(self):
        parsed = urlparse(self.path)
        path = parsed.path

        if path.startswith("/api/records/"):
            record_id = path.split("/")[3]
            data, status = handle_delete_record(self, record_id)
            return json_response(self, data, status)

        if path == "/api/trash":
            data, status = handle_clear_trash(self)
            return json_response(self, data, status)

        if path.startswith("/api/trash/"):
            parts = path.split("/")
            if len(parts) >= 4:
                record_id = parts[3]
                data, status = handle_permanent_delete(self, record_id)
                return json_response(self, data, status)

        json_response(self, err("Unknown endpoint"), 404)

    def do_PATCH(self):
        parsed = urlparse(self.path)
        path = parsed.path

        if path == "/api/config":
            data, status = handle_patch_config(self)
            return json_response(self, data, status)

        json_response(self, err("Unknown endpoint"), 404)

    def log_message(self, format, *args):
        # Suppress default stderr logging, use our own logger
        pass


# ============================================================
# Server Runner
# ============================================================

def run_server(port=None, open_browser=True):
    """Start the HTTP server."""
    if port is None:
        port = get_saved_port() or PORT_DEFAULT

    port = find_available_port(port)
    if port is None:
        print("ERROR: No available port found in range.")
        sys.exit(1)

    save_port(port)

    addr = ("0.0.0.0", port)
    server = HTTPServer(addr, Handler)

    url = f"http://localhost:{port}"
    print(f"\n🍃 VSC Dashboard running at {url}")
    print(f"   Cache: {CACHE_FILE}")
    print(f"   Press Ctrl+C to stop\n")

    if open_browser:
        import subprocess
        subprocess.run(["open", url], check=False)

    try:
        server.serve_forever()
    except KeyboardInterrupt:
        print("\n\n👋 Server stopped.")
        server.shutdown()
        sys.exit(0)


# ============================================================
# CLI Entry
# ============================================================

if __name__ == "__main__":
    import argparse

    parser = argparse.ArgumentParser(description="VSC HTTP Server")
    parser.add_argument("--port", type=int, default=None, help="Port to listen on")
    parser.add_argument("--no-open", action="store_true", help="Don't open browser automatically")
    args = parser.parse_args()

    run_server(port=args.port, open_browser=not args.no_open)
