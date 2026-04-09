#!/usr/bin/env python3
"""
Build script to package the VSC project into Alfred workflow format.

Alfred Workflow Format (.alfredworkflow):
- A ZIP archive with .alfredworkflow extension
- Contains: info.plist, scripts, assets, icon.png

Usage:
    python scripts/build_workflow.py [--output <path>]
    
Options:
    --output    Output path for .alfredworkflow file (default: ./build/vsc.alfredworkflow)
    --dry-run   Show what would be included without creating the archive
"""

import argparse
import json
import os
import shutil
import sys
import zipfile
from datetime import datetime
from pathlib import Path


# ============================================================
# Configuration
# ============================================================

SCRIPT_DIR = Path(__file__).parent
PROJECT_ROOT = SCRIPT_DIR.parent
SRC_DIR = PROJECT_ROOT / "src"
PUBLIC_DIR = PROJECT_ROOT / "public"
BUILD_DIR = PROJECT_ROOT / "build"

# Files to include in the workflow
WORKFLOW_FILES = {
    # Core scripts (from src/)
    "shared.py": SRC_DIR / "shared.py",
    "vsc.py": SRC_DIR / "vsc.py",
    "cli.py": SRC_DIR / "cli.py",
    "server.py": SRC_DIR / "server.py",

    # Web UI (served by server.py, at workflow root)
    "api.js": SRC_DIR / "web" / "api.js",
    "app.js": SRC_DIR / "web" / "app.js",
    "index.html": SRC_DIR / "web" / "index.html",
    "styles.css": SRC_DIR / "web" / "styles.css",

    # Assets (from src/assets/)
    "assets/folder.png": SRC_DIR / "assets" / "folder.png",
    "assets/file.png": SRC_DIR / "assets" / "file.png",
    "assets/remote.png": SRC_DIR / "assets" / "remote.png",

    # Workflow configuration (from public/)
    "info.plist": PUBLIC_DIR / "info.plist",
    "icon.png": PUBLIC_DIR / "icon.png",
}


# ============================================================
# Build Functions
# ============================================================

def validate_files() -> list:
    """Validate that all required files exist."""
    missing = []
    for dest_name, src_path in WORKFLOW_FILES.items():
        if not src_path.exists():
            missing.append(str(src_path))
    return missing


def get_version() -> str:
    """Get version from package.json."""
    package_json = PROJECT_ROOT / "package.json"
    if package_json.exists():
        with open(package_json, "r", encoding="utf-8") as f:
            data = json.load(f)
            return data.get("version", "0.0.0")
    return "0.0.0"


def build_workflow(output_path: Path, dry_run: bool = False) -> bool:
    """
    Build the Alfred workflow package.
    
    Args:
        output_path: Path for the output .alfredworkflow file
        dry_run: If True, just print what would be included
        
    Returns:
        True if successful, False otherwise
    """
    # Validate files
    missing = validate_files()
    if missing:
        print("❌ Error: Missing required files:")
        for f in missing:
            print(f"   - {f}")
        return False
    
    version = get_version()
    print(f"📦 Building VSC Alfred Workflow v{version}")
    print()
    
    # Show files to include
    print("📋 Files to include:")
    total_size = 0
    for dest_name, src_path in WORKFLOW_FILES.items():
        size = src_path.stat().st_size
        total_size += size
        size_str = f"{size:,} bytes" if size < 1024 else f"{size/1024:.1f} KB"
        print(f"   {dest_name:<30} ({size_str})")
    
    print(f"\n   Total: {total_size:,} bytes ({total_size/1024:.1f} KB)")
    print()
    
    if dry_run:
        print("🔍 Dry run - no file created")
        return True
    
    # Create build directory
    BUILD_DIR.mkdir(parents=True, exist_ok=True)
    
    # Create the workflow archive
    try:
        with zipfile.ZipFile(output_path, 'w', zipfile.ZIP_DEFLATED) as zf:
            for dest_name, src_path in WORKFLOW_FILES.items():
                zf.write(src_path, dest_name)
                
        print(f"✅ Workflow created: {output_path}")
        print(f"   Size: {output_path.stat().st_size:,} bytes")
        return True
        
    except Exception as e:
        print(f"❌ Error creating workflow: {e}")
        return False


def print_workflow_info():
    """Print information about the workflow structure."""
    print("📖 Alfred Workflow Structure")
    print("=" * 50)
    print()
    print("Alfred workflows (.alfredworkflow) are ZIP archives containing:")
    print()
    print("  Required:")
    print("    info.plist     - Workflow configuration (triggers, connections, etc.)")
    print()
    print("  Optional:")
    print("    icon.png       - Workflow icon (shown in Alfred preferences)")
    print("    *.py/*.sh      - Scripts executed by workflow actions")
    print("    assets/        - Icons and other resources")
    print()
    print("  This workflow includes:")
    print("    vsc.py         - Main search script (Script Filter)")
    print("    cli.py         - CLI commands (rebuild, delete, selected)")
    print("    info.plist     - Workflow triggers and connections")
    print("    icon.png       - Workflow icon")
    print("    assets/*.png   - Icons for different path types")
    print()


# ============================================================
# Main
# ============================================================

def main():
    parser = argparse.ArgumentParser(
        description="Build VSC Alfred Workflow package",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
    python scripts/build_workflow.py                    # Build to ./build/vsc.alfredworkflow
    python scripts/build_workflow.py --output ~/Desktop/vsc.alfredworkflow
    python scripts/build_workflow.py --dry-run          # Preview without building
    python scripts/build_workflow.py --info             # Show workflow structure info
        """
    )
    
    parser.add_argument(
        "--output", "-o",
        type=str,
        default=str(BUILD_DIR / "vsc.alfredworkflow"),
        help="Output path for .alfredworkflow file"
    )
    
    parser.add_argument(
        "--dry-run", "-n",
        action="store_true",
        help="Show what would be included without creating the archive"
    )
    
    parser.add_argument(
        "--info",
        action="store_true",
        help="Print information about Alfred workflow structure"
    )
    
    args = parser.parse_args()
    
    if args.info:
        print_workflow_info()
        return 0
    
    output_path = Path(args.output)
    
    # Ensure .alfredworkflow extension
    if output_path.suffix != ".alfredworkflow":
        output_path = output_path.with_suffix(".alfredworkflow")
    
    success = build_workflow(output_path, dry_run=args.dry_run)
    return 0 if success else 1


if __name__ == "__main__":
    sys.exit(main())
