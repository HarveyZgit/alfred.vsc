#!/usr/bin/env python3
"""
VSC Alfred Workflow - CLI commands
Handles: rebuild, delete, selected
"""

import argparse
import json
import os
import sqlite3
import sys
from datetime import datetime
from pathlib import Path
from typing import Optional
from urllib.parse import unquote

# ============================================================
# Configuration (same as vsc.py)
# ============================================================

HOME = Path.home()
WORKFLOW_DIR = Path(__file__).parent

# Dev mode support: use separate cache directory when VSC_DEV_MODE is set
DEV_MODE = os.environ.get("VSC_DEV_MODE", "") == "1"
DEV_CACHE_DIR = Path(os.environ.get("VSC_DEV_CACHE_DIR", "/tmp/vsc-dev"))

if DEV_MODE:
    CACHE_FILE = DEV_CACHE_DIR / ".records.cache.json"
    USER_CONFIG_FILE = DEV_CACHE_DIR / ".user.config.json"
    LOGS_DIR = DEV_CACHE_DIR / "logs"
    # Ensure dev cache directory exists
    DEV_CACHE_DIR.mkdir(parents=True, exist_ok=True)
else:
    CACHE_FILE = WORKFLOW_DIR / ".records.cache.json"
    USER_CONFIG_FILE = WORKFLOW_DIR / ".user.config.json"
    LOGS_DIR = WORKFLOW_DIR / "logs"

# Environment variables
ENV_IDE_PATH = os.environ.get("VSC_IDE_PATH", str(HOME / "Library/Application Support/Code"))
ENV_DIRECTORIES = os.environ.get("VSC_DIRECTORIES", "")

# Derived paths
GLOBAL_STORAGE_PATH = Path(ENV_IDE_PATH) / "User/globalStorage"
DATABASE_PATH = GLOBAL_STORAGE_PATH / "state.vscdb"
STORAGE_JSON_PATH = GLOBAL_STORAGE_PATH / "storage.json"

# Result codes
RESULT_SUCCESS = "rebuild_index_success"
RESULT_FAIL = "rebuild_index_fail"


# ============================================================
# Logging
# ============================================================

def get_log_file() -> Path:
    """Get log file path for today."""
    LOGS_DIR.mkdir(exist_ok=True)
    today = datetime.now().strftime("%Y%m%d")
    return LOGS_DIR / f"vsc.{today}.log"


def log(level: str, message: str, context: Optional[dict] = None):
    """Log a message."""
    timestamp = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
    log_line = f"[{timestamp}] [{level}] {message}"
    if context:
        log_line += f" ; [ctx]={json.dumps(context)}"
    
    with open(get_log_file(), "a", encoding="utf-8") as f:
        f.write(log_line + "\n")


def log_info(message: str, context: Optional[dict] = None):
    log("info", message, context)


def log_error(message: str, context: Optional[dict] = None):
    log("error", message, context)


# ============================================================
# Utility Functions (same as vsc.py)
# ============================================================

import hashlib
import re


def is_folder_path(path: str) -> bool:
    clean_path = re.sub(r"^(.*?)://", "", path)
    return Path(clean_path).is_dir() if clean_path and Path(clean_path).exists() else False


def is_remote_path(path: str) -> bool:
    return "remote" in path


def get_icon(path: str) -> dict:
    if is_remote_path(path):
        return {"path": "./assets/remote.png"}
    elif is_folder_path(path):
        return {"path": "./assets/folder.png"}
    else:
        return {"path": "./assets/file.png"}


def get_project_name(path: str) -> str:
    decoded = unquote(path)
    match = re.search(r".*/(.+?)$", decoded)
    return match.group(1) if match else decoded


def generate_record_id(path: str) -> str:
    return hashlib.md5(path.encode()).hexdigest()


# ============================================================
# Data Sources
# ============================================================

def get_records_from_vscode_db(limit: Optional[int] = 100) -> list:
    """Get records from VSCode's SQLite database."""
    if not DATABASE_PATH.exists():
        log_error(f"Database not found: {DATABASE_PATH}")
        return []
    
    try:
        conn = sqlite3.connect(str(DATABASE_PATH))
        cursor = conn.cursor()
        cursor.execute(
            "SELECT value FROM ItemTable WHERE key = 'history.recentlyOpenedPathsList'"
        )
        row = cursor.fetchone()
        conn.close()
        
        if not row:
            return []
        
        data = json.loads(row[0])
        entries = data.get("entries", [])
        
        if limit:
            entries = entries[:limit]
        
        records = []
        for entry in entries:
            if isinstance(entry, str):
                entry = {"fileUri": entry}
            
            path = entry.get("fileUri") or entry.get("folderUri") or \
                   (entry.get("workspace", {}) or {}).get("configPath", "")
            
            if not path:
                continue
                
            path = unquote(path)
            log_info(f"{get_icon(path)['path'].split('/')[-1].split('.')[0]} -- {path}")
            records.append({
                "__vsc_id__": generate_record_id(path),
                "name": get_project_name(path),
                "path": path,
                "icon": get_icon(path),
                "extra": {"from": "getRecordsFromVscodeDB"},
            })
        
        return records
    except Exception as e:
        log_error(f"Failed to read database: {e}")
        return []


def get_records_from_vscode_menu() -> list:
    """Get records from VSCode's storage.json menu data."""
    if not STORAGE_JSON_PATH.exists():
        return []
    
    try:
        with open(STORAGE_JSON_PATH, "r", encoding="utf-8") as f:
            config = json.load(f)
        
        file_menu_items = (
            config.get("lastKnownMenubarData", {})
            .get("menus", {})
            .get("File", {})
            .get("items", [])
        )
        
        recent_folder_config = None
        for item in file_menu_items:
            if item.get("id", "").startswith("submenuitem"):
                submenu = item.get("submenu", {})
                submenu_items = submenu.get("items", [])
                for sub_item in submenu_items:
                    if sub_item.get("id") == "openRecentFolder":
                        recent_folder_config = item
                        break
        
        if not recent_folder_config:
            return []
        
        recent_list = recent_folder_config.get("submenu", {}).get("items", [])
        
        records = []
        for item in recent_list:
            if item.get("id") != "openRecentFolder" or not item.get("uri"):
                continue
            
            uri = item["uri"]
            path = uri.get("external") or ""
            if not path:
                scheme = uri.get("scheme", "")
                uri_path = uri.get("path", "")
                if scheme == "file":
                    path = f"{scheme}://{uri_path}"
                elif scheme == "vscode-remote":
                    authority = uri.get("authority", "")
                    path = f"{scheme}://{authority}{uri_path}"
            
            if not path:
                continue
            
            records.append({
                "__vsc_id__": generate_record_id(path),
                "name": get_project_name(path),
                "path": path,
                "icon": get_icon(path),
                "extra": {"from": "getRecordsFromVscodeMenu"},
            })
        
        return records
    except Exception:
        return []


def find_git_repos(root_path: Path, max_depth: int = 5) -> list:
    """
    Recursively find all directories containing .git.
    Stops searching deeper once a .git is found (won't find nested repos).
    
    Args:
        root_path: The root directory to start searching from
        max_depth: Maximum depth to search (default 5 to avoid going too deep)
    
    Returns:
        List of Path objects for directories containing .git
    """
    git_repos = []
    
    def search(current_path: Path, depth: int):
        if depth > max_depth:
            return
        
        try:
            # Check if this directory contains .git
            git_dir = current_path / ".git"
            if git_dir.exists():
                git_repos.append(current_path)
                # Don't search deeper - found a git repo
                return
            
            # Not a git repo, search subdirectories
            for item in current_path.iterdir():
                if item.is_dir() and not item.name.startswith("."):
                    search(item, depth + 1)
        except PermissionError:
            pass
        except OSError:
            pass
    
    search(root_path, 0)
    return git_repos


def get_records_from_directories() -> list:
    """
    Get records from user-specified directories.
    
    Scanning strategy:
    1. Recursively find all directories containing .git (stops at git boundary)
    2. Also include direct child directories as fallback (for non-git projects)
    
    Naming: Uses relative path from VSC_DIRECTORIES root to distinguish same-named projects.
    """
    if not ENV_DIRECTORIES:
        return []
    
    paths = [p.strip() for p in ENV_DIRECTORIES.split(",") if p.strip()]
    records = []
    seen_paths = set()  # Avoid duplicates
    
    for dir_path in paths:
        if dir_path.startswith("~"):
            dir_path = str(HOME / dir_path[2:])
        
        dir_path_obj = Path(dir_path)
        
        if not dir_path_obj.exists() or not dir_path_obj.is_dir():
            log_error(f"Directory not found or not a directory: {dir_path}")
            continue
        
        try:
            # Strategy 1: Find all git repos recursively
            git_repos = find_git_repos(dir_path_obj)
            for repo_path in git_repos:
                path = f"file://{repo_path}"
                if path not in seen_paths:
                    seen_paths.add(path)
                    # Use relative path from root directory as name
                    relative_path = repo_path.relative_to(dir_path_obj)
                    display_name = str(relative_path)
                    log_info(f"git-repo -- {display_name} -- {path}")
                    records.append({
                        "__vsc_id__": generate_record_id(path),
                        "name": display_name,
                        "path": path,
                        "icon": get_icon(path),
                        "extra": {"from": "getRecordsFromGitRepo"},
                    })
            
            # Strategy 2: Also add direct child directories (fallback for non-git projects)
            for item in dir_path_obj.iterdir():
                if item.is_dir():
                    path = f"file://{item}"
                    if path not in seen_paths:
                        seen_paths.add(path)
                        # Direct children just use their name (one level deep)
                        log_info(f"folder -- {item.name} -- {path}")
                        records.append({
                            "__vsc_id__": generate_record_id(path),
                            "name": item.name,
                            "path": path,
                            "icon": get_icon(path),
                            "extra": {"from": "getRecordsFromSpecifiedDirectory"},
                        })
        except PermissionError:
            log_error(f"Permission denied: {dir_path}")
            continue
    
    return records


# ============================================================
# Cache Management
# ============================================================

def get_cache_default() -> dict:
    return {"records": [], "trash": {}}


def read_cache() -> dict:
    if not CACHE_FILE.exists():
        return get_cache_default()
    
    try:
        with open(CACHE_FILE, "r", encoding="utf-8") as f:
            return json.load(f)
    except Exception:
        return get_cache_default()


def write_cache(data: dict) -> None:
    with open(CACHE_FILE, "w", encoding="utf-8") as f:
        json.dump(data, f, indent=2)


# ============================================================
# CLI Commands
# ============================================================

def cmd_rebuild(drop_all: bool = False):
    """Rebuild the index."""
    log_info("Start Rebuilding Index.")
    log_info(f"Rebuild Mode: {'drop all' if drop_all else 'save trash'}")
    
    try:
        # Get records from user-specified directories only
        # (skip VSCode database as user no longer uses VSCode)
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
    - Update branch cache for this record
    """
    log_info(f"Selected record: {record_path}")
    
    try:
        cache = read_cache()
        records = cache.get("records", [])
        record_id = generate_record_id(record_path)
        
        # Find and remove the selected record
        selected_record = None
        new_records = []
        for record in records:
            if record.get("__vsc_id__") == record_id:
                selected_record = record
            else:
                new_records.append(record)
        
        # Move selected record to the top
        if selected_record:
            new_records.insert(0, selected_record)
            cache["records"] = new_records
            write_cache(cache)
            log_info(f"Record moved to top: {record_path}")
        
    except Exception as e:
        log_error(f"selected error: {e}")


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
    
    args = parser.parse_args()
    
    if args.command == "rebuild":
        cmd_rebuild(drop_all=args.drop_all)
    elif args.command == "delete":
        cmd_delete(args.record)
    elif args.command == "selected":
        cmd_selected(args.record)
    else:
        parser.print_help()


if __name__ == "__main__":
    main()
