#!/usr/bin/env python3
"""
Dev Install Script - Install development version of VSC workflow to Alfred.

This creates an isolated development workflow that:
- Uses a different bundleid (com.harvey.alfredapp.vsc.dev)
- Uses different keywords (vscd, vscdli)
- Uses a separate cache directory (/tmp/vsc-dev/)
- Symlinks to source code for instant updates

Usage:
    python scripts/dev_install.py          # Install dev workflow
    python scripts/dev_install.py --status # Check installation status
    python scripts/dev_install.py --remove # Remove dev workflow
"""

import argparse
import os
import plistlib
import shutil
import sys
from pathlib import Path


# ============================================================
# Configuration
# ============================================================

SCRIPT_DIR = Path(__file__).parent
PROJECT_ROOT = SCRIPT_DIR.parent
SRC_DIR = PROJECT_ROOT / "src"
PUBLIC_DIR = PROJECT_ROOT / "public"

# Alfred workflow directory
ALFRED_WORKFLOWS_DIR = Path.home() / "Library/Application Support/Alfred/Alfred.alfredpreferences/workflows"

# Dev workflow settings
DEV_WORKFLOW_DIR_NAME = "user.workflow.vsc-dev"
DEV_BUNDLE_ID = "com.harvey.alfredapp.vsc.dev"
DEV_WORKFLOW_NAME = "vsc (dev)"

# Keyword replacements: original -> dev
KEYWORD_MAP = {
    "vsc": "vscd",
    "vscli": "vscdli",
}

# Environment variable for dev mode
DEV_ENV_VARS = {
    "VSC_DEV_MODE": "1",
    "VSC_DEV_CACHE_DIR": "/tmp/vsc-dev",
}


# ============================================================
# Plist Modification
# ============================================================

def modify_plist_for_dev(plist_path: Path) -> dict:
    """
    Read and modify info.plist for dev mode.
    
    Changes:
    - bundleid -> com.harvey.alfredapp.vsc.dev
    - name -> vsc (dev)
    - keywords: vsc -> vscd, vscli -> vscdli
    - Add dev environment variables
    - Remove hotkey triggers (dev version uses keywords only)
    """
    with open(plist_path, "rb") as f:
        plist = plistlib.load(f)
    
    # Change bundle ID and name
    plist["bundleid"] = DEV_BUNDLE_ID
    plist["name"] = DEV_WORKFLOW_NAME
    
    # Add dev environment variables
    if "variables" not in plist:
        plist["variables"] = {}
    plist["variables"].update(DEV_ENV_VARS)
    
    # Find hotkey trigger UIDs to remove
    hotkey_uids = set()
    for obj in plist.get("objects", []):
        if obj.get("type") == "alfred.workflow.trigger.hotkey":
            hotkey_uids.add(obj.get("uid"))
    
    # Remove hotkey triggers from objects
    plist["objects"] = [
        obj for obj in plist.get("objects", [])
        if obj.get("type") != "alfred.workflow.trigger.hotkey"
    ]
    
    # Remove connections from hotkey triggers
    connections = plist.get("connections", {})
    for uid in hotkey_uids:
        if uid in connections:
            del connections[uid]
    
    # Remove hotkey positions from uidata
    uidata = plist.get("uidata", {})
    for uid in hotkey_uids:
        if uid in uidata:
            del uidata[uid]
    
    # Modify keywords in objects
    for obj in plist.get("objects", []):
        config = obj.get("config", {})
        if "keyword" in config:
            old_keyword = config["keyword"]
            if old_keyword in KEYWORD_MAP:
                config["keyword"] = KEYWORD_MAP[old_keyword]
    
    return plist


def write_plist(plist: dict, path: Path):
    """Write plist to file."""
    with open(path, "wb") as f:
        plistlib.dump(plist, f)


# ============================================================
# Installation Functions
# ============================================================

def get_dev_workflow_path() -> Path:
    """Get path to dev workflow directory."""
    return ALFRED_WORKFLOWS_DIR / DEV_WORKFLOW_DIR_NAME


def is_dev_installed() -> bool:
    """Check if dev workflow is installed."""
    dev_path = get_dev_workflow_path()
    return dev_path.exists()


def install_dev_workflow():
    """Install development workflow."""
    print("🔧 Installing VSC dev workflow...")
    print()
    
    # Check Alfred workflows directory exists
    if not ALFRED_WORKFLOWS_DIR.exists():
        print(f"❌ Alfred workflows directory not found:")
        print(f"   {ALFRED_WORKFLOWS_DIR}")
        print("   Is Alfred installed?")
        return False
    
    dev_workflow_path = get_dev_workflow_path()
    
    # Remove existing dev workflow if present
    if dev_workflow_path.exists():
        print(f"   Removing existing dev workflow...")
        if dev_workflow_path.is_symlink():
            dev_workflow_path.unlink()
        else:
            shutil.rmtree(dev_workflow_path)
    
    # Create dev workflow directory
    dev_workflow_path.mkdir(parents=True)
    print(f"   Created: {dev_workflow_path}")
    
    # Generate modified info.plist
    print("   Generating dev info.plist...")
    source_plist = PUBLIC_DIR / "info.plist"
    dev_plist = modify_plist_for_dev(source_plist)
    write_plist(dev_plist, dev_workflow_path / "info.plist")
    
    # Copy dev icon
    print("   Copying dev icon...")
    dev_icon_src = PUBLIC_DIR / "icon.png"
    if dev_icon_src.exists():
        shutil.copy(dev_icon_src, dev_workflow_path / "icon.png")
        print(f"      icon.png copied")
    else:
        print(f"      ⚠️  icon.png not found")
    
    # Create symlinks to source files
    print("   Creating symlinks to source files...")
    
    # Symlink Python scripts
    for script in ["vsc.py", "cli.py", "shared.py", "server.py"]:
        src = SRC_DIR / script
        dst = dev_workflow_path / script
        dst.symlink_to(src)
        print(f"      {script} -> {src}")

    # Symlink web directory (for server static files)
    web_src = SRC_DIR / "web"
    web_dst = dev_workflow_path / "web"
    web_dst.symlink_to(web_src)
    print(f"      web/ -> {web_src}")

    # Symlink assets directory
    assets_src = SRC_DIR / "assets"
    assets_dst = dev_workflow_path / "assets"
    assets_dst.symlink_to(assets_src)
    print(f"      assets/ -> {assets_src}")
    
    # Create dev cache directory
    cache_dir = Path(DEV_ENV_VARS["VSC_DEV_CACHE_DIR"])
    cache_dir.mkdir(parents=True, exist_ok=True)
    print(f"   Created cache dir: {cache_dir}")
    
    print()
    print("✅ Dev workflow installed successfully!")
    print()
    print("📋 Usage:")
    print(f"   Keyword:     {KEYWORD_MAP['vsc']} (search projects)")
    print(f"   CLI:         {KEYWORD_MAP['vscli']} (manage commands)")
    print(f"   Cache:       {DEV_ENV_VARS['VSC_DEV_CACHE_DIR']}/")
    print()
    print("💡 Tips:")
    print("   - Edit source files, changes take effect immediately")
    print("   - Run 'vscdli rebuild index' to rebuild dev cache")
    print("   - Dev and prod workflows are completely isolated")
    
    return True


def show_status():
    """Show installation status."""
    dev_path = get_dev_workflow_path()
    
    print("📊 VSC Dev Workflow Status")
    print("=" * 50)
    print()
    
    if not dev_path.exists():
        print("❌ Not installed")
        print()
        print("Run: python scripts/dev_install.py")
        return
    
    print(f"✅ Installed")
    print(f"   Path: {dev_path}")
    print()
    
    # Check symlinks
    print("   Symlinks:")
    for name in ["vsc.py", "cli.py", "shared.py", "server.py", "web", "assets"]:
        link = dev_path / name
        if link.is_symlink():
            target = link.resolve()
            exists = "✅" if target.exists() else "❌ (broken)"
            print(f"      {name} -> {target} {exists}")
        elif link.exists():
            print(f"      {name} (regular file)")
        else:
            print(f"      {name} ❌ missing")
    
    # Check cache
    cache_dir = Path(DEV_ENV_VARS["VSC_DEV_CACHE_DIR"])
    if cache_dir.exists():
        cache_file = cache_dir / ".records.cache.json"
        if cache_file.exists():
            size = cache_file.stat().st_size
            print(f"\n   Cache: {cache_file} ({size} bytes)")
        else:
            print(f"\n   Cache: {cache_dir}/ (empty)")
    
    print()
    print(f"   Keywords: {KEYWORD_MAP['vsc']}, {KEYWORD_MAP['vscli']}")


def remove_dev_workflow():
    """Remove development workflow."""
    dev_path = get_dev_workflow_path()
    
    if not dev_path.exists():
        print("ℹ️  Dev workflow is not installed")
        return True
    
    print("🗑️  Removing dev workflow...")
    
    if dev_path.is_symlink():
        dev_path.unlink()
    else:
        shutil.rmtree(dev_path)
    
    # Optionally clean cache
    cache_dir = Path(DEV_ENV_VARS["VSC_DEV_CACHE_DIR"])
    if cache_dir.exists():
        shutil.rmtree(cache_dir)
        print(f"   Cleaned cache: {cache_dir}")
    
    print("✅ Dev workflow removed")
    return True


# ============================================================
# Main
# ============================================================

def main():
    parser = argparse.ArgumentParser(
        description="Install/manage VSC dev workflow for Alfred",
        formatter_class=argparse.RawDescriptionHelpFormatter,
    )
    
    parser.add_argument(
        "--status", "-s",
        action="store_true",
        help="Show installation status"
    )
    
    parser.add_argument(
        "--remove", "-r",
        action="store_true",
        help="Remove dev workflow"
    )
    
    args = parser.parse_args()
    
    if args.status:
        show_status()
    elif args.remove:
        remove_dev_workflow()
    else:
        install_dev_workflow()


if __name__ == "__main__":
    main()
