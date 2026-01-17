#!/usr/bin/env python3
"""
VSC Alfred Workflow - Search recent VSCode projects
Python rewrite of the original Node.js implementation
"""

import json
import os
import re
import sqlite3
import sys
from pathlib import Path
from typing import Optional
from urllib.parse import unquote

# ============================================================
# Configuration
# ============================================================

HOME = Path.home()
WORKFLOW_DIR = Path(__file__).parent

# Dev mode support: use separate cache directory when VSC_DEV_MODE is set
DEV_MODE = os.environ.get("VSC_DEV_MODE", "") == "1"
DEV_CACHE_DIR = Path(os.environ.get("VSC_DEV_CACHE_DIR", "/tmp/vsc-dev"))

if DEV_MODE:
    CACHE_FILE = DEV_CACHE_DIR / ".records.cache.json"
    USER_CONFIG_FILE = DEV_CACHE_DIR / ".user.config.json"
    # Ensure dev cache directory exists
    DEV_CACHE_DIR.mkdir(parents=True, exist_ok=True)
else:
    CACHE_FILE = WORKFLOW_DIR / ".records.cache.json"
    USER_CONFIG_FILE = WORKFLOW_DIR / ".user.config.json"

# Environment variables
ENV_IDE_PATH = os.environ.get("VSC_IDE_PATH", str(HOME / "Library/Application Support/Code"))
ENV_DIRECTORIES = os.environ.get("VSC_DIRECTORIES", "")

# Derived paths
GLOBAL_STORAGE_PATH = Path(ENV_IDE_PATH) / "User/globalStorage"
DATABASE_PATH = GLOBAL_STORAGE_PATH / "state.vscdb"
STORAGE_JSON_PATH = GLOBAL_STORAGE_PATH / "storage.json"

# ============================================================
# Path Type Definitions
# ============================================================

PATH_TYPE_ALL = "all"
PATH_TYPE_FILE = "file"
PATH_TYPE_FOLDER = "folder"
PATH_TYPE_REMOTE = "remote"

PATH_TYPE_ALIASES = {
    PATH_TYPE_FILE: ["file", "fi", "f"],
    PATH_TYPE_FOLDER: ["folder", "fo", "ff", "dir", "d"],
    PATH_TYPE_REMOTE: ["remote", "re", "r"],
}


# ============================================================
# Utility Functions
# ============================================================

def is_folder_path(path: str) -> bool:
    """Check if path is a folder (after removing scheme)."""
    clean_path = remove_path_scheme(path)
    return Path(clean_path).is_dir() if clean_path and Path(clean_path).exists() else False


def is_remote_path(path: str) -> bool:
    """Check if path is a remote path."""
    return "remote" in path


def is_file_path(path: str) -> bool:
    """Check if path is a file (not folder and not remote)."""
    return not is_folder_path(path) and not is_remote_path(path)


def remove_path_scheme(path: str) -> str:
    """Remove URI scheme from path."""
    return re.sub(r"^(.*?)://", "", path)


def get_git_branch(path: str) -> Optional[str]:
    """
    Get git branch by reading .git/HEAD file directly.
    This is much faster than executing `git branch` command.
    Returns None if not a git repo or on error.
    """
    # Remove file:// scheme
    clean_path = remove_path_scheme(path)
    if not clean_path:
        return None
    
    git_head = Path(clean_path) / ".git" / "HEAD"
    
    if not git_head.exists():
        return None
    
    try:
        with open(git_head, "r") as f:
            content = f.read().strip()
        
        # Format: "ref: refs/heads/branch-name"
        if content.startswith("ref: refs/heads/"):
            return content[16:]  # len("ref: refs/heads/") == 16
        
        # Detached HEAD (commit hash)
        if len(content) == 40:
            return content[:7]  # Short hash
        
        return None
    except Exception:
        return None


def get_icon(path: str) -> dict:
    """Get icon based on path type."""
    if is_remote_path(path):
        return {"path": "./assets/remote.png"}
    elif is_folder_path(path):
        return {"path": "./assets/folder.png"}
    else:
        return {"path": "./assets/file.png"}


def get_project_name(path: str) -> str:
    """Extract project name from path."""
    decoded = unquote(path)
    match = re.search(r".*/(.+?)$", decoded)
    return match.group(1) if match else decoded


def generate_record_id(path: str) -> str:
    """Generate a simple hash for record ID."""
    import hashlib
    return hashlib.md5(path.encode()).hexdigest()


# ============================================================
# Data Sources
# ============================================================

def get_records_from_vscode_db(limit: Optional[int] = 100) -> list:
    """Get records from VSCode's SQLite database."""
    if not DATABASE_PATH.exists():
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
            records.append({
                "__vsc_id__": generate_record_id(path),
                "name": get_project_name(path),
                "path": path,
                "icon": get_icon(path),
                "extra": {"from": "getRecordsFromVscodeDB"},
            })
        
        return records
    except Exception as e:
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
        
        # Find recent folder config
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


def get_records_from_directories() -> list:
    """Get records from user-specified directories."""
    if not ENV_DIRECTORIES:
        return []
    
    paths = [p.strip() for p in ENV_DIRECTORIES.split(",") if p.strip()]
    records = []
    
    for dir_path in paths:
        # Expand ~ to home directory
        if dir_path.startswith("~"):
            dir_path = str(HOME / dir_path[2:])
        
        dir_path = Path(dir_path)
        
        if not dir_path.exists() or not dir_path.is_dir():
            continue
        
        try:
            for item in dir_path.iterdir():
                if item.is_dir():
                    path = f"file://{item}"
                    records.append({
                        "__vsc_id__": generate_record_id(path),
                        "name": item.name,
                        "path": path,
                        "icon": get_icon(path),
                        "extra": {"from": "getRecordsFromSpecifiedDirectory"},
                    })
        except PermissionError:
            continue
    
    return records


# ============================================================
# Cache Management
# ============================================================

def get_cache_default() -> dict:
    """Get default cache structure."""
    return {"records": [], "trash": {}}


def read_cache() -> dict:
    """Read cache from file."""
    if not CACHE_FILE.exists():
        return get_cache_default()
    
    try:
        with open(CACHE_FILE, "r", encoding="utf-8") as f:
            return json.load(f)
    except Exception:
        return get_cache_default()


def write_cache(data: dict) -> None:
    """Write cache to file."""
    with open(CACHE_FILE, "w", encoding="utf-8") as f:
        json.dump(data, f, indent=2)


def get_cached_records() -> list:
    """Get cached records."""
    cache = read_cache()
    return cache.get("records", [])


def update_branches_cache() -> None:
    """
    Update git branch info for all cached records.
    Called only on initial search (empty query) for performance.
    """
    cache = read_cache()
    records = cache.get("records", [])
    
    branches = {}
    for record in records:
        path = record.get("path", "")
        branch = get_git_branch(path)
        if branch:
            branches[record.get("__vsc_id__", "")] = branch
    
    cache["branches"] = branches
    write_cache(cache)


def get_cached_branch(record_id: str) -> Optional[str]:
    """Get cached branch for a record."""
    cache = read_cache()
    return cache.get("branches", {}).get(record_id)


# ============================================================
# Search & Filter
# ============================================================

def parse_input(query: str) -> dict:
    """Parse search input."""
    params = query.split()
    
    if not params:
        return {
            "original": query,
            "path_type": PATH_TYPE_ALL,
            "search_key": "",
        }
    
    # Check if first param is a path type
    first_param = params[0].lower()
    path_type = PATH_TYPE_ALL
    
    for ptype, aliases in PATH_TYPE_ALIASES.items():
        if first_param in aliases:
            path_type = ptype
            params = params[1:]  # Remove path type from search
            break
    
    return {
        "original": query,
        "path_type": path_type,
        "search_key": " ".join(params),
    }


def matches_path_type(record: dict, path_type: str) -> bool:
    """Check if record matches path type filter."""
    if path_type == PATH_TYPE_ALL:
        return True
    
    path = record.get("path", "")
    
    if path_type == PATH_TYPE_REMOTE:
        return is_remote_path(path)
    elif path_type == PATH_TYPE_FOLDER:
        return is_folder_path(path)
    elif path_type == PATH_TYPE_FILE:
        return is_file_path(path)
    
    return True


def fuzzy_match(pattern: str, text: str) -> tuple[bool, int]:
    """
    Fuzzy match pattern against text.
    Returns (is_match, score).
    
    Score is higher for:
    - Consecutive matches
    - Matches at word boundaries
    - Shorter text (more specific match)
    """
    pattern = pattern.lower()
    text = text.lower()
    
    # Empty pattern matches everything
    if not pattern:
        return (True, 0)
    
    # Exact substring match gets highest score
    if pattern in text:
        # Bonus for exact match at start
        if text.startswith(pattern):
            return (True, 1000 + (100 - len(text)))
        return (True, 500 + (100 - len(text)))
    
    # Fuzzy match: all characters must appear in order
    pattern_idx = 0
    score = 0
    prev_match_idx = -1
    
    for i, char in enumerate(text):
        if pattern_idx < len(pattern) and char == pattern[pattern_idx]:
            # Consecutive match bonus
            if prev_match_idx == i - 1:
                score += 10
            else:
                score += 1
            
            # Word boundary bonus (after -, _, /, or start)
            if i == 0 or text[i-1] in '-_/ ':
                score += 5
            
            prev_match_idx = i
            pattern_idx += 1
    
    # All pattern characters must be found
    if pattern_idx == len(pattern):
        # Penalty for longer text
        score = score + (100 - min(len(text), 100))
        return (True, score)
    
    return (False, 0)


def filter_records(records: list, input_info: dict) -> list:
    """Filter records based on input with fuzzy matching."""
    path_type = input_info["path_type"]
    search_key = input_info["search_key"]
    
    results = []
    for record in records:
        # Check path type
        if not matches_path_type(record, path_type):
            continue
        
        # Fuzzy match on name
        name = record.get("name", "")
        is_match, score = fuzzy_match(search_key, name)
        
        if is_match:
            results.append((record, score))
    
    # Sort by score (highest first), preserving original order for equal scores
    results.sort(key=lambda x: -x[1])
    
    return [r[0] for r in results]


# ============================================================
# Alfred Output
# ============================================================

def format_for_alfred(records: list) -> dict:
    """Format records for Alfred output."""
    items = []
    for record in records:
        path = record.get("path", "")
        clean_path = remove_path_scheme(path)
        record_id = record.get("__vsc_id__", "")
        
        # Use cached branch info (updated on initial search)
        branch = get_cached_branch(record_id)
        
        # Format subtitle: ⎇ branch | path or just path
        if branch:
            subtitle = f"⎇ {branch} | {clean_path}"
        else:
            subtitle = clean_path
        
        items.append({
            "title": record.get("name", "Unknown"),
            "subtitle": subtitle,
            "arg": path,
            "icon": record.get("icon", {"path": "./assets/folder.png"}),
        })
    
    return {"items": items}


# ============================================================
# Main Entry Point
# ============================================================

def main():
    """Main entry point for search."""
    query = sys.argv[1] if len(sys.argv) > 1 else ""
    
    # Parse input
    input_info = parse_input(query)
    
    # Get records from cache
    records = get_cached_records()
    
    # Fallback to menu if no cached records
    if not records:
        records = get_records_from_vscode_menu()
    
    # On initial search (empty query), update branch cache
    # This ensures branches are fresh when user first opens vsc
    if not query.strip():
        update_branches_cache()
    
    # Filter records
    filtered = filter_records(records, input_info)
    
    # Output for Alfred
    output = format_for_alfred(filtered)
    print(json.dumps(output))


if __name__ == "__main__":
    main()
