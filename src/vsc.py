#!/usr/bin/env python3
"""
VSC Alfred Workflow - Search recent VSCode projects
"""

import json
import sys
from typing import Optional

from shared import (
    PATH_TYPE_ALL,
    PATH_TYPE_ALIASES,
    PATH_TYPE_FILE,
    PATH_TYPE_FOLDER,
    PATH_TYPE_REMOTE,
    get_cached_records,
    get_git_branch,
    get_records_from_vscode_menu,
    get_vsc_directory_roots,
    ENV_TAB_TAIL_SLASH,
    is_file_path,
    is_folder_path,
    is_remote_path,
    read_cache,
    remove_path_scheme,
    scan_subdirectories,
    write_cache,
)

# ============================================================
# Branch Cache
# ============================================================

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
# Directory Browse Mode
# ============================================================

def parse_browse_query(query: str) -> tuple:
    """
    Parse a browse-mode query (after removing leading '/').

    Returns (segments, search):
      '/'           -> ([], '')          list top-level
      '/alf'        -> ([], 'alf')       flat recursive search
      '/alfred/'    -> (['alfred'], '')  drill into alfred
      '/alfred/vs'  -> (['alfred'], 'vs') drill + search
      '/a/b/c/'     -> (['a','b','c'], '') deep drill
    """
    # Remove the leading '/'
    raw = query[1:]

    if not raw:
        return ([], '')

    parts = raw.split('/')

    # If query ended with '/', last part is empty string -> drill-down mode
    # e.g. 'alfred/' -> parts = ['alfred', ''] -> segments=['alfred'], search=''
    if query.endswith('/'):
        segments = [p for p in parts if p]  # filter empty strings
        return (segments, '')

    # Otherwise last part is the search keyword
    # e.g. 'alfred/vs' -> parts = ['alfred', 'vs'] -> segments=['alfred'], search='vs'
    search = parts[-1]
    segments = [p for p in parts[:-1] if p]
    return (segments, search)


def format_browse_results(dir_entries: list, segments: list = None) -> dict:
    """
    Format browse results for Alfred output.

    Args:
        dir_entries: List of dicts with 'path' (Path), 'name' (str), 'rel' (str)
        segments: Current path segments for building autocomplete paths

    Returns:
        Alfred Script Filter JSON dict
    """
    segments = segments or []
    prefix = '/' + '/'.join(segments) + '/' if segments else '/'

    items = []
    for entry in dir_entries:
        dir_path = entry['path']
        display_name = entry['rel']  # relative name for display
        full_path = str(dir_path)
        file_uri = f'file://{full_path}'

        # Tab autocomplete: drill into this directory
        suffix = '/' if ENV_TAB_TAIL_SLASH else ''
        autocomplete = f'{prefix}{entry["rel"]}{suffix}'
        items.append({
            'title': display_name,
            'subtitle': full_path,
            'arg': file_uri,
            'autocomplete': autocomplete,
            'icon': {'path': './assets/folder.png'},
        })

    return {'items': items}


def browse_directories(query: str) -> dict:
    """
    Handle directory browse mode.
    Called when query starts with '/'.

    Returns Alfred Script Filter JSON output.
    """
    roots = get_vsc_directory_roots()
    if not roots:
        return {'items': []}

    segments, search = parse_browse_query(query)

    # Scan directories based on mode
    dir_entries = scan_subdirectories(
        roots=roots,
        segments=segments,
        search=search,
    )

    # If there's a search keyword, apply fuzzy match filtering
    if search:
        scored = []
        for entry in dir_entries:
            match_target = entry['rel']  # match against relative path
            is_match, score = fuzzy_match(search, match_target)
            if is_match:
                scored.append((entry, score))
        scored.sort(key=lambda x: -x[1])
        dir_entries = [e[0] for e in scored]

    # If drill-down yields no results, show a hint
    if not dir_entries and segments and not search:
        current_path = '/'.join(segments)
        # Autocomplete pops back up one level
        parent_autocomplete = '/' + '/'.join(segments[:-1]) + '/' if len(segments) > 1 else '/'
        return {'items': [{
            'title': '没有更多子目录了',
            'subtitle': current_path,
            'arg': '',
            'autocomplete': parent_autocomplete,
            'icon': {'path': './assets/folder.png'},
            'valid': False,
        }]}

    return format_browse_results(dir_entries, segments)


# ============================================================
# Main Entry Point
# ============================================================

def main():
    """Main entry point for search."""
    query = sys.argv[1] if len(sys.argv) > 1 else ""

    # Directory browse mode: query starts with '/'
    if query.lstrip().startswith('/'):
        output = browse_directories(query.lstrip())
        print(json.dumps(output))
        return

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
