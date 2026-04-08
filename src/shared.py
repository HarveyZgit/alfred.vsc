#!/usr/bin/env python3
"""
VSC Alfred Workflow - Shared utilities
Common code used by both vsc.py (search) and cli.py (CLI commands).
"""

import hashlib
import json
import os
import re
import sqlite3
from datetime import datetime
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
ENV_TAB_TAIL_SLASH = os.environ.get("VSC_TAB_TAIL_SLASH", "1") == "1"
ENV_SHRINK_PATH = os.environ.get("VSC_ENABLE_SHRINK_PATH", "1") == "1"

# Derived paths
GLOBAL_STORAGE_PATH = Path(ENV_IDE_PATH) / "User/globalStorage"
DATABASE_PATH = GLOBAL_STORAGE_PATH / "state.vscdb"
STORAGE_JSON_PATH = GLOBAL_STORAGE_PATH / "storage.json"

# Directories to skip during browsing (non-project directories)
SKIP_DIRS = {
    'node_modules', '__pycache__', 'venv', '.venv',
    'dist', 'build', 'vendor', 'target', 'Pods',
    '.git', '.svn', '.hg',
    'miniprogram_npm',
}

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
# Utility Functions
# ============================================================

def remove_path_scheme(path: str) -> str:
    """Remove URI scheme from path."""
    return re.sub(r"^(.*?)://", "", path)


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


def get_git_branch(path: str) -> Optional[str]:
    """
    Get git branch by reading .git/HEAD file directly.
    Handles:
    - Normal git repositories
    - Git worktrees (where .git is a file)
    - Subdirectories of git repositories
    Returns None if not a git repo or on error.
    """
    clean_path = remove_path_scheme(path)
    if not clean_path:
        return None

    path_obj = Path(clean_path)

    # Strategy 1: Direct check (for root or worktree)
    git_head = path_obj / ".git" / "HEAD"
    if git_head.exists():
        return _parse_git_head(git_head)

    # Check if it's a worktree (.git is a file)
    git_file = path_obj / ".git"
    if git_file.is_file():
        # Worktree: .git contains "gitdir: /path/to/actual/.git/worktrees/name"
        try:
            with open(git_file, "r") as f:
                content = f.read().strip()
            if content.startswith("gitdir: "):
                git_dir = Path(content[8:])  # Remove "gitdir: "
                worktree_head = git_dir / "HEAD"
                if worktree_head.exists():
                    return _parse_git_head(worktree_head)
        except Exception:
            pass

    # Strategy 2: Walk up the directory tree to find git root
    current = path_obj
    for _ in range(20):  # Increased depth limit for deep nested projects
        # Check if .git is a directory (normal repo)
        git_dir = current / ".git"
        if git_dir.is_dir():
            git_head = git_dir / "HEAD"
            if git_head.exists():
                return _parse_git_head(git_head)

        # Check if .git is a file (worktree) - for subdirs of worktree
        worktree_git_file = current / ".git"
        if worktree_git_file.is_file():
            try:
                with open(worktree_git_file, "r") as f:
                    worktree_content = f.read().strip()
                if worktree_content.startswith("gitdir: "):
                    git_dir = Path(worktree_content[8:])
                    worktree_head = git_dir / "HEAD"
                    if worktree_head.exists():
                        return _parse_git_head(worktree_head)
            except Exception:
                pass

        current = current.parent
        if current == current.parent:
            break

    return None


def _parse_git_head(git_head: Path) -> Optional[str]:
    """Parse .git/HEAD file and extract branch name."""
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


def get_path_prefixes() -> list:
    """
    Get configured path prefixes from environment variable.
    VSC_PATH_PREFIXES is a JSON array of {path, alias} objects.
    Path supports $HOME placeholder.
    Example: [{"path": "$HOME/Code/Playground", "alias": "~Playground"}]
    """
    env_value = os.environ.get("VSC_PATH_PREFIXES", "")
    if not env_value:
        return []

    try:
        prefixes = json.loads(env_value)
        result = []
        for item in prefixes:
            path = item.get("path", "")
            alias = item.get("alias", "")
            if path and alias:
                # Replace $HOME with actual home path
                if path.startswith("$HOME"):
                    path = str(HOME) + path[5:]
                result.append((path, alias))
        return result
    except Exception:
        return []


def shrink_path(path: str, n: int = 3) -> str:
    """
    Shorten a path by replacing common prefixes and limiting path segments.
    When VSC_ENABLE_SHRINK_PATH=1: full shrinking with segment limit
    When VSC_ENABLE_SHRINK_PATH=0: only replace $HOME with ~
    """
    home = str(HOME)

    # Simple mode: only replace $HOME with ~
    if not ENV_SHRINK_PATH:
        if path == home:
            return "~"
        if path.startswith(home + "/"):
            return "~" + path[len(home):]
        return path

    # Full shrink mode: user config first (higher priority), then defaults
    prefixes = get_path_prefixes()
    prefixes.extend([
        (home, "~"),
    ])

    prefix, rest = "", path
    for match, alias in prefixes:
        if path == match:
            return alias
        if path.startswith(match + "/"):
            prefix = alias
            rest = path[len(match):]
            break

    segs = [s for s in rest.split("/") if s]
    if not segs:
        return prefix or "/"

    if len(segs) <= n:
        return prefix + "/" + "/".join(segs)

    return prefix + "/…/" + "/".join(segs[-n:])


def get_project_name(path: str) -> str:
    """Extract project name from path."""
    decoded = unquote(path)
    match = re.search(r".*/(.+?)$", decoded)
    return match.group(1) if match else decoded


def generate_record_id(path: str) -> str:
    """Generate a simple hash for record ID."""
    return hashlib.md5(path.encode()).hexdigest()


def make_record(path: str, source: str) -> dict:
    """Build a standard record dict."""
    return {
        "__vsc_id__": generate_record_id(path),
        "name": get_project_name(path),
        "path": path,
        "icon": get_icon(path),
        "extra": {"from": source},
    }


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
            records.append(make_record(path, "getRecordsFromVscodeDB"))

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

            records.append(make_record(path, "getRecordsFromVscodeMenu"))

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
                    record = make_record(path, "getRecordsFromGitRepo")
                    record["name"] = display_name
                    records.append(record)

            # Strategy 2: Also add direct child directories (fallback for non-git projects)
            for item in dir_path_obj.iterdir():
                if item.is_dir():
                    path = f"file://{item}"
                    if path not in seen_paths:
                        seen_paths.add(path)
                        log_info(f"folder -- {item.name} -- {path}")
                        record = make_record(path, "getRecordsFromSpecifiedDirectory")
                        record["name"] = item.name
                        records.append(record)
        except PermissionError:
            log_error(f"Permission denied: {dir_path}")
            continue

    return records


def get_vsc_directory_roots() -> list:
    """Parse VSC_DIRECTORIES env var into a list of resolved Path objects."""
    if not ENV_DIRECTORIES:
        return []

    roots = []
    for p in ENV_DIRECTORIES.split(','):
        p = p.strip()
        if not p:
            continue
        if p.startswith('~'):
            p = str(HOME / p[2:])
        path = Path(p)
        if path.exists() and path.is_dir():
            roots.append(path)
    return roots


def scan_subdirectories(
    roots: list,
    segments: list = None,
    search: str = '',
    max_depth: int = 5,
) -> list:
    """
    Scan directories under VSC_DIRECTORIES roots for browsing.

    Args:
        roots: List of root Path objects (from get_vsc_directory_roots())
        segments: Path segments to drill into (e.g., ['alfred', 'vsc'])
        search: Search keyword for fuzzy matching (empty = list all)
        max_depth: Max recursion depth for flat search (only used when
                   segments is empty and search is non-empty)

    Returns:
        List of dicts with 'path' (Path), 'name' (str), 'rel' (str)
    """
    segments = segments or []
    results = []
    seen = set()

    def _is_browsable(d: Path) -> bool:
        """Check if a directory should be included in browse results."""
        return d.is_dir() and not d.name.startswith('.') and d.name not in SKIP_DIRS

    def _add_dir(d: Path, rel_name: str):
        """Add a directory to results if not already seen."""
        resolved = str(d.resolve())
        if resolved not in seen:
            seen.add(resolved)
            results.append({
                'path': d,
                'name': d.name,
                'rel': rel_name,
            })

    def _collect_recursive(base: Path, rel_prefix: str, depth: int):
        """Recursively collect directories up to max_depth."""
        if depth > max_depth:
            return
        try:
            for item in sorted(base.iterdir()):
                if _is_browsable(item):
                    rel = f'{rel_prefix}/{item.name}' if rel_prefix else item.name
                    _add_dir(item, rel)
                    _collect_recursive(item, rel, depth + 1)
        except (PermissionError, OSError):
            pass

    if segments:
        # Drill-down mode: locate the target directory via segments
        # Start from all roots, narrow down segment by segment
        current_dirs = list(roots)
        for seg in segments:
            next_dirs = []
            for parent in current_dirs:
                try:
                    candidate = parent / seg
                    if candidate.exists() and candidate.is_dir():
                        next_dirs.append(candidate)
                except (PermissionError, OSError):
                    pass
            current_dirs = next_dirs
            if not current_dirs:
                return []  # Segment not found

        # List direct children of the target directory(ies)
        for target in current_dirs:
            try:
                for item in sorted(target.iterdir()):
                    if _is_browsable(item):
                        _add_dir(item, item.name)
            except (PermissionError, OSError):
                pass

    elif search:
        # Flat recursive search mode: scan all roots up to max_depth
        for root in roots:
            _collect_recursive(root, '', 1)

    else:
        # Top-level listing: direct children of all roots
        for root in roots:
            try:
                for item in sorted(root.iterdir()):
                    if _is_browsable(item):
                        _add_dir(item, item.name)
            except (PermissionError, OSError):
                pass

    return results


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
