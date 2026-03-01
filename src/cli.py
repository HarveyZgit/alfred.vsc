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

# Result codes
RESULT_SUCCESS = "rebuild_index_success"
RESULT_FAIL = "rebuild_index_fail"


# ============================================================
# CLI Commands
# ============================================================

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
