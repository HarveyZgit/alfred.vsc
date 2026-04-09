#!/usr/bin/env python3
"""
VSC Alfred Workflow - CLI commands
Handles: rebuild, delete, selected
"""

import argparse
import sys
from datetime import datetime

from shared import (
    generate_record_id,
    get_records_from_directories,
    log_error,
    log_info,
    read_cache,
    write_cache,
)

import os
import signal

# Result codes
RESULT_SUCCESS = "rebuild_index_success"
RESULT_FAIL = "rebuild_index_fail"
PID_FILE = os.path.join(os.path.dirname(os.path.abspath(__file__)), ".server.pid")

def cmd_rebuild(drop_all: bool = False):
    """Rebuild the index."""
    log_info("Start Rebuilding Index.")
    log_info(f"Rebuild Mode: {'drop all' if drop_all else 'save trash'}")

    try:
        # Get records from user-specified directories only
        dir_records = get_records_from_directories()

        # Dedupe by path
        seen_paths = set()
        unique_records = []
        for record in dir_records:
            if record["path"] not in seen_paths:
                seen_paths.add(record["path"])
                unique_records.append(record)

        # Handle trash
        cache = read_cache()
        if drop_all:
            cache["trash"] = {}
        else:
            trash_ids = set(cache.get("trash", {}).keys())
            unique_records = [r for r in unique_records if r["__vsc_id__"] not in trash_ids]

        # Save
        cache["records"] = unique_records
        write_cache(cache)

        log_info(f"Index rebuilt successfully. Total records: {len(unique_records)}")
        print(RESULT_SUCCESS)

    except Exception as e:
        log_error(f"rebuild index error: {e}")
        print(RESULT_FAIL)


def cmd_delete(record_path: str):
    """Delete a record (move to trash)."""
    log_info(f"Deleting record: {record_path}")

    try:
        cache = read_cache()
        record_id = generate_record_id(record_path)

        # Add to trash
        cache.setdefault("trash", {})[record_id] = {
            "path": record_path,
            "deleted_at": datetime.now().isoformat(),
        }

        # Remove from records
        cache["records"] = [r for r in cache.get("records", []) if r["__vsc_id__"] != record_id]

        write_cache(cache)
        log_info(f"Record deleted: {record_path}")

    except Exception as e:
        log_error(f"delete error: {e}")


def cmd_selected(record_path: str):
    """
    Handle record selection:
    - Move selected record to the top of the list (most recently used)
    - If record not in list, add it to the top
    - If cache is empty, auto rebuild index first to load all records
    - Update branch cache for this record
    """
    log_info(f"Selected record: {record_path}")

    try:
        cache = read_cache()
        records = cache.get("records", [])
        record_id = generate_record_id(record_path)

        # 如果缓存是空的，自动重建索引加载所有目录和vscode历史记录
        if not records:
            log_info("Cache is empty, auto rebuilding index...")
            from shared import get_records_from_directories, get_records_from_vscode_menu
            # 加载目录扫描结果
            dir_records = get_records_from_directories()
            # 加载vscode历史记录
            vscode_records = get_records_from_vscode_menu()
            # 合并去重
            seen_paths = set()
            all_records = []
            for record in dir_records + vscode_records:
                if record["path"] not in seen_paths:
                    seen_paths.add(record["path"])
                    all_records.append(record)
            records = all_records
            # 移除回收站里的记录
            trash_ids = set(cache.get("trash", {}).keys())
            records = [r for r in records if r["__vsc_id__"] not in trash_ids]

        # Find and remove the selected record
        selected_record = None
        new_records = []
        for record in records:
            if record.get("__vsc_id__") == record_id:
                selected_record = record
            else:
                new_records.append(record)

        # If record not found, create new one
        if not selected_record:
            from shared import make_record
            selected_record = make_record(record_path, "user_selected")
            log_info(f"Created new record for selected path: {record_path}")

        # Move selected record to the top
        new_records.insert(0, selected_record)
        cache["records"] = new_records
        write_cache(cache)
        log_info(f"Record moved to top: {record_path}")

        # 更新这个新记录的分支缓存
        from shared import update_branches_cache
        update_branches_cache(record_ids=[selected_record["__vsc_id__"]])

    except Exception as e:
        log_error(f"selected error: {e}")


def cmd_serve(port=None, no_open=False):
    """Start the visual management web UI in the background."""
    # Check if already running
    if os.path.exists(PID_FILE):
        try:
            with open(PID_FILE) as f:
                pid = int(f.read().strip())
            os.kill(pid, 0)
            print(f"Server is already running (PID {pid})")
            print(f"URL: http://localhost:{port or 3847}")
            return
        except (ProcessLookupError, ValueError):
            # Stale PID file
            os.remove(PID_FILE)

    # Daemonize
    pid = os.fork()
    if pid == 0:
        # Child — start server
        os.setsid()
        devnull = os.devnull
        os.dup2(open(devnull, "rb").fileno(), 0)
        os.dup2(open(devnull, "ab").fileno(), 1)
        os.dup2(open(devnull, "ab").fileno(), 2)

        from server import run_server
        run_server(port=port, open_browser=(not no_open))
        os._exit(0)

    # Parent
    import time
    time.sleep(1.5)
    with open(PID_FILE, "w") as f:
        f.write(str(pid))
    print(f"🍃 Server started (PID {pid})")
    print(f"   URL: http://localhost:{port or 3847}")


def cmd_stop():
    """Stop the running visual management server."""
    if not os.path.exists(PID_FILE):
        print("Server is not running.")
        return
    try:
        with open(PID_FILE) as f:
            pid = int(f.read().strip())
        os.kill(pid, signal.SIGTERM)
        time.sleep(1)
        os.remove(PID_FILE)
        print("🍃 Server stopped.")
    except ProcessLookupError:
        os.remove(PID_FILE)
        print("Server was not running (stale PID file removed).")
    except Exception as e:
        print(f"Failed to stop server: {e}")


# ============================================================
# Main Entry Point
# ============================================================

def main():
    parser = argparse.ArgumentParser(description="VSC CLI")
    subparsers = parser.add_subparsers(dest="command")

    # rebuild command
    rebuild_parser = subparsers.add_parser("rebuild")
    rebuild_parser.add_argument("--drop-all", action="store_true", help="Clear trash as well")

    # delete command
    delete_parser = subparsers.add_parser("delete")
    delete_parser.add_argument("--record", required=True, help="Record path to delete")

    # selected command
    selected_parser = subparsers.add_parser("selected")
    selected_parser.add_argument("--record", required=True, help="Selected record path")

    # serve command
    serve_parser = subparsers.add_parser("serve")
    serve_parser.add_argument("--port", type=int, default=None, help="Port to listen on")
    serve_parser.add_argument("--no-open", action="store_true", help="Don't open browser automatically")

    # stop command
    subparsers.add_parser("stop")

    args = parser.parse_args()

    if args.command == "rebuild":
        cmd_rebuild(drop_all=args.drop_all)
    elif args.command == "delete":
        cmd_delete(args.record)
    elif args.command == "selected":
        cmd_selected(args.record)
    elif args.command == "serve":
        cmd_serve(port=args.port, no_open=args.no_open)
    elif args.command == "stop":
        cmd_stop()
    else:
        parser.print_help()


if __name__ == "__main__":
    main()
