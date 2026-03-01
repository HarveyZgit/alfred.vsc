# AGENT.md

## Project Overview

**VSC** is an [Alfred](https://www.alfredapp.com/) workflow for quickly searching and opening recent VS Code (or any IDE) projects. It reads VS Code's internal database and storage files, combines them with user-configured project directories, and presents results via Alfred's Script Filter protocol.

- **Author:** Harvey Z
- **License:** MIT
- **Current Version:** 2.0.0 (Python rewrite; previously Node.js)
- **Repository:** <https://github.com/HarveyZgit/alfred.vsc>

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Language | **Python 3** (no external dependencies) |
| Platform | macOS + Alfred 5 |
| Data source | VS Code SQLite DB (`state.vscdb`) + `storage.json` |
| Packaging | ZIP archive → `.alfredworkflow` |
| Configuration | Alfred workflow environment variables |

## Project Structure

```
├── src/
│   ├── shared.py           # Shared utilities, config, data sources, cache
│   ├── vsc.py              # Search engine (Alfred Script Filter)
│   ├── cli.py              # CLI commands: rebuild / delete / selected
│   └── assets/             # Icons (folder.png, file.png, remote.png)
├── public/
│   ├── info.plist           # Alfred workflow configuration (triggers, connections)
│   ├── icon.png             # Workflow icon
│   ├── logo.svg             # Production logo
│   └── logo-dev.svg         # Dev-mode logo
├── scripts/
│   ├── build_workflow.py    # Package project → build/vsc.alfredworkflow
│   └── dev_install.py       # Install isolated dev workflow into Alfred
├── build/                   # Build output (gitignored)
├── package.json             # Version metadata
├── CHANGELOG.md
├── README.md
└── .editorconfig
```

## Key Modules

### `src/shared.py` — Shared Utilities

Common module imported by both `vsc.py` and `cli.py`. Contains:

- **Configuration** — all path constants, env vars, dev-mode support
- **Logging** — `log_info()`, `log_error()` with daily log rotation
- **Utilities** — `remove_path_scheme`, `is_folder_path`, `is_remote_path`, `get_icon`, `get_project_name`, `generate_record_id`, `get_git_branch`
- **Record builder** — `make_record(path, source)` for consistent record construction
- **Data sources** — `get_records_from_vscode_db()`, `get_records_from_vscode_menu()`, `find_git_repos()`, `get_records_from_directories()`
- **Cache management** — `read_cache()`, `write_cache()`, `get_cached_records()`

### `src/vsc.py` — Search Engine

Entry point for the `vsc` keyword in Alfred. Imports from `shared.py`. Contains only search-specific logic:

- **Search:** `fuzzy_match()` scores candidates by consecutive matches, word-boundary bonuses, and text length
- **Path type filtering:** `parse_input()`, `matches_path_type()` with aliases like `d`, `f`, `r`
- **Branch cache:** `update_branches_cache()`, `get_cached_branch()` — updated lazily on empty-query invocations
- **Alfred output:** `format_for_alfred()` — Script Filter JSON format

### `src/cli.py` — CLI Commands

Entry point for the `vscli` keyword. Imports from `shared.py`. Subcommands:

| Command | Function | Description |
|---------|----------|-------------|
| `rebuild` | `cmd_rebuild()` | Re-merge all data sources and rebuild the cache |
| `delete` | `cmd_delete()` | Soft-delete a record (move to trash list) |
| `selected` | `cmd_selected()` | Promote a record to MRU position and refresh its branch |

### `scripts/build_workflow.py`

Packages `src/`, `public/info.plist`, `public/icon.png`, and assets into a ZIP archive with `.alfredworkflow` extension.

```bash
python3 scripts/build_workflow.py              # → build/vsc.alfredworkflow
python3 scripts/build_workflow.py --dry-run    # preview only
```

### `scripts/dev_install.py`

Installs an isolated development copy of the workflow into Alfred:

- **Bundle ID:** `com.harvey.alfredapp.vsc.dev`
- **Keywords:** `vscd` / `vscdli`
- **Cache dir:** `/tmp/vsc-dev/`
- **Symlinks** source files so edits take effect immediately
- **No hotkeys** to avoid conflicts with the production version

```bash
python3 scripts/dev_install.py            # install
python3 scripts/dev_install.py --status   # check status
python3 scripts/dev_install.py --remove   # uninstall
```

## Environment Variables

Configured in Alfred's workflow settings panel:

| Variable | Purpose | Example |
|----------|---------|---------|
| `VSC_DIRECTORIES` | Comma-separated project root directories to scan | `~/Code,~/Projects` |
| `VSC_OPEN_DEFAULT` | Binary path for Enter key action | `/usr/local/bin/code` |
| `VSC_OPEN_WITH_CMD` | Binary path for Cmd+Enter action | `.../Cursor` |
| `VSC_IDE_PATH` | Path to VS Code config directory | `~/.config/Code` |
| `VSC_DEV_MODE` | Set `1` to enable dev-mode cache isolation | `1` |
| `VSC_DEV_CACHE_DIR` | Override cache directory in dev mode | `/tmp/vsc-dev` |

## Coding Conventions

- **Python 3**, stdlib only — no pip dependencies
- **EditorConfig:** 2-space indent, LF line endings, UTF-8, trailing whitespace trimmed
- Single quotes for Python strings; double quotes for HTML/JSON/CSS
- Section separators use `# ============` comment blocks
- Functions include docstrings describing purpose
- Alfred output follows the [Script Filter JSON Format](https://www.alfredapp.com/help/workflows/inputs/script-filter/json/)

## Build & Development

```bash
# Build production workflow
python3 scripts/build_workflow.py

# Install dev version (isolated from production)
python3 scripts/dev_install.py

# Check dev status
python3 scripts/dev_install.py --status
```

There is no test suite. Verification is done manually through Alfred by invoking the `vsc` / `vscd` keywords.

## Important Notes

- All shared logic lives in `shared.py`. Both `vsc.py` and `cli.py` import from it.
- The workflow reads VS Code's **internal** SQLite database and JSON storage; the schema may change across VS Code versions.
- `info.plist` is an Apple XML plist that defines Alfred triggers, script connections, and UI elements. Edit with care.
- The `.alfredworkflow` file in `build/` is gitignored.
