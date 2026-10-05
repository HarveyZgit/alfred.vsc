#!/usr/bin/env python3
"""Build self-contained binaries and a universal macOS Alfred workflow."""

import argparse
import hashlib
import json
import os
from pathlib import Path
import plistlib
import shutil
import struct
import subprocess
import sys
import zipfile

ROOT = Path(__file__).resolve().parent.parent
BUILD = ROOT / "build"
TARGETS = {
    "linux-amd64": ("linux", "amd64"),
    "darwin-arm64": ("darwin", "arm64"),
    "darwin-amd64": ("darwin", "amd64"),
}


def universal(inputs, destination):
    """Join unchanged Mach-O slices using Apple's public fat binary format."""
    slices, offset = ([], 1 << 14)
    for name, cpu, subtype in [("darwin-arm64", 0x0100000C, 0), ("darwin-amd64", 0x01000007, 3)]:
        data = inputs[name].read_bytes()
        magic, actual_cpu, actual_subtype = struct.unpack_from("<III", data)
        if magic != 0xFEEDFACF or actual_cpu != cpu:
            raise ValueError(f"{inputs[name]} is not the expected Mach-O architecture")
        slices.append((cpu, actual_subtype, offset, data))
        offset = (offset + len(data) + (1 << 14) - 1) & ~((1 << 14) - 1)
    with destination.open("wb") as out:
        out.write(struct.pack(">II", 0xCAFEBABE, len(slices)))
        for cpu, subtype, position, data in slices:
            out.write(struct.pack(">IIIII", cpu, subtype, position, len(data), 14))
        for _, _, position, data in slices:
            out.seek(position)
            out.write(data)
    destination.chmod(0o755)


def package(binary, output, development=False):
    config = plistlib.loads((ROOT / "public/info.plist").read_bytes())
    expected = json.loads((ROOT / "package.json").read_text())["version"]
    if config["version"] != expected:
        raise ValueError(
            "package.json and info.plist versions differ; run scripts/generate_workflow.py"
        )
    if development:
        config["bundleid"] += ".dev"
        config["name"] += " (dev)"
        # Avoid competing with the release hotkey during development.
        config["objects"] = [obj for obj in config["objects"] if obj["uid"] != "HOTKEY"]
        config["connections"].pop("HOTKEY", None)
        config["uidata"].pop("HOTKEY", None)
        config["variables"].update(VSC_DEV_MODE="1", VSC_DEV_CACHE_DIR="/tmp/vsc-dev-v3")
        for obj in config["objects"]:
            c = obj.get("config", {})
            if c.get("keyword") in ("vsc", "vscli"):
                c["keyword"] = {"vsc": "vscd", "vscli": "vscdli"}[c["keyword"]]
    output.parent.mkdir(parents=True, exist_ok=True)
    files = {
        "vsc": binary,
        "icon.png": ROOT / "public/icon.png",
        "README.md": ROOT / "README.md",
        "LICENSE": ROOT / "LICENSE",
        "THIRD_PARTY_NOTICES.txt": ROOT / "THIRD_PARTY_NOTICES.txt",
    }
    files["scripts/migrate-v2.sh"] = ROOT / "scripts/migrate-v2.sh"
    files.update(
        {
            f"docs/{name}.zh-CN.md": ROOT / f"docs/{name}.zh-CN.md"
            for name in ("ARCHITECTURE", "VALIDATION", "MACOS-ACCEPTANCE", "MIGRATION")
        }
    )
    files.update(
        {f"assets/{name}.png": ROOT / f"public/assets/{name}.png" for name in ("folder", "remote")}
    )
    with zipfile.ZipFile(output, "w", zipfile.ZIP_DEFLATED) as archive:
        archive.writestr("info.plist", plistlib.dumps(config, sort_keys=False))
        for name, path in files.items():
            info = zipfile.ZipInfo(name)
            info.create_system = 3
            info.external_attr = (
                0o100755 if name in ("vsc", "scripts/migrate-v2.sh") else 0o100644
            ) << 16
            info.compress_type = zipfile.ZIP_DEFLATED
            archive.writestr(info, path.read_bytes())
    with zipfile.ZipFile(output) as archive:
        if archive.testzip() is not None:
            raise ValueError("Archive checksum failed")
    print(output)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--go", default=os.environ.get("VSC_GO", "go"))
    parser.add_argument("--target", choices=("all", "linux", "macos"), default="all")
    parser.add_argument(
        "--skip-build", action="store_true", help="Package existing build/vsc-<os>-<arch> files"
    )
    parser.add_argument("--dev", action="store_true")
    parser.add_argument("--output", type=Path)
    parser.add_argument("--dry-run", action="store_true")
    args = parser.parse_args()
    names = (
        list(TARGETS)
        if args.target == "all"
        else [
            name
            for name in TARGETS
            if name.startswith("linux" if args.target == "linux" else "darwin")
        ]
    )
    if args.dry_run:
        print(
            json.dumps(
                {"targets": names, "runtime_dependencies": [], "workflow": args.target != "linux"},
                indent=2,
            )
        )
        return
    BUILD.mkdir(exist_ok=True)
    if not args.skip_build:
        tool = shutil.which(args.go) if not Path(args.go).is_absolute() else args.go
        if not tool:
            raise ValueError("Go is required to build; end users do not need it")
        version = subprocess.run(
            [tool, "version"], capture_output=True, text=True, check=True
        ).stdout
        if not version.startswith("go version go"):
            raise ValueError("--go must point to the Go compiler")
        pnpm = shutil.which("pnpm")
        if not pnpm:
            raise ValueError(
                "Node.js and pnpm are required to build the panel; end users do not need them"
            )
        subprocess.run([pnpm, "--dir", "web", "install", "--frozen-lockfile"], cwd=ROOT, check=True)
        subprocess.run([pnpm, "--dir", "web", "run", "build"], cwd=ROOT, check=True)
        for name in names:
            system, arch = TARGETS[name]
            environment = dict(os.environ, CGO_ENABLED="0", GOOS=system, GOARCH=arch)
            subprocess.run(
                [
                    tool,
                    "build",
                    "-mod=readonly",
                    "-trimpath",
                    "-ldflags=-s -w",
                    "-o",
                    str(BUILD / f"vsc-{name}"),
                    "./cmd/vsc",
                ],
                cwd=ROOT,
                env=environment,
                check=True,
            )
    for name in names:
        (BUILD / f"vsc-{name}").chmod(0o755)
    if args.target != "linux":
        universal(
            {name: BUILD / f"vsc-{name}" for name in names if name.startswith("darwin")},
            BUILD / "vsc",
        )
        output = args.output or BUILD / (
            "vsc-dev.alfredworkflow" if args.dev else "vsc.alfredworkflow"
        )
        package(BUILD / "vsc", output, args.dev)
    outputs = [BUILD / f"vsc-{name}" for name in names]
    if args.target != "linux":
        outputs += [BUILD / "vsc", output]
    outputs = sorted(
        set(
            outputs
            + list(BUILD.glob("*.alfredworkflow"))
            + list(BUILD.glob("vsc-*"))
            + ([BUILD / "vsc"] if (BUILD / "vsc").exists() else [])
        )
    )
    (BUILD / "SHA256SUMS").write_text(
        "".join((f"{hashlib.sha256(p.read_bytes()).hexdigest()}  {p.name}\n" for p in outputs))
    )
    print("Checksums:", BUILD / "SHA256SUMS")


if __name__ == "__main__":
    try:
        main()
    except (OSError, ValueError, subprocess.CalledProcessError) as error:
        print(f"Build failed: {error}", file=sys.stderr)
        sys.exit(1)
