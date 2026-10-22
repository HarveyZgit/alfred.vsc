#!/usr/bin/env python3
"""Validate, stage and publish stable releases from an already-tested build."""

import argparse
import hashlib
import json
from pathlib import Path
import plistlib
import re
import shutil
import subprocess
import tempfile
import zipfile

ROOT = Path(__file__).resolve().parent.parent
ASSETS = ("vsc.alfredworkflow", "migrate-v2.sh", "SHA256SUMS")


def release_metadata(root, tag=None):
    version = json.loads((root / "package.json").read_text())["version"]
    if not re.fullmatch(r"(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)", version):
        raise ValueError("Only stable X.Y.Z versions are supported")
    if tag is not None and tag != version:
        raise ValueError(f"Tag {tag!r} must exactly match package version {version!r}")
    versions = {
        "web/package.json": json.loads((root / "web/package.json").read_text())["version"],
        "public/info.plist": plistlib.loads((root / "public/info.plist").read_bytes())["version"],
    }
    match = re.search(
        r'^const Version = "([^"]+)"$', (root / "internal/workflow/config.go").read_text(), re.M
    )
    versions["internal/workflow/config.go"] = match.group(1) if match else None
    for name, actual in versions.items():
        if actual != version:
            raise ValueError(f"{name}: expected {version}, got {actual}")
    notice = (root / "THIRD_PARTY_NOTICES.txt").read_text().splitlines()
    if not notice or notice[0] != f"Third-party notices for VSC {version}":
        raise ValueError("THIRD_PARTY_NOTICES.txt version does not match")
    changelog = (root / "CHANGELOG.md").read_text()
    match = re.search(rf"^## v{re.escape(version)}\n(.*?)(?=^## |\Z)", changelog, re.M | re.S)
    if not match or not match.group(1).strip():
        raise ValueError(f"Missing non-empty CHANGELOG section for v{version}")
    return version, match.group(1).strip()


def checksums(directory):
    return "".join(
        f"{hashlib.sha256((directory / name).read_bytes()).hexdigest()}  {name}\n"
        for name in ASSETS[:2]
    )


def prepare(root, build, output, tag=None):
    version, changes = release_metadata(root, tag)
    package = build / "vsc.alfredworkflow"
    with zipfile.ZipFile(package) as archive:
        if archive.testzip() is not None:
            raise ValueError("Workflow archive is corrupt")
        config = plistlib.loads(archive.read("info.plist"))
        if config["version"] != version or config["bundleid"] != "com.harvey.alfredapp.vsc.v3":
            raise ValueError("Workflow must be the production bundle with the release version")
        if archive.read("scripts/migrate-v2.sh") != (root / "scripts/migrate-v2.sh").read_bytes():
            raise ValueError("Packaged migration script differs from release source")
    source_sums = (build / "SHA256SUMS").read_text().splitlines()
    expected = f"{hashlib.sha256(package.read_bytes()).hexdigest()}  vsc.alfredworkflow"
    if expected not in source_sums:
        raise ValueError("Workflow checksum differs from the tested build")
    output.mkdir(parents=True, exist_ok=True)
    shutil.copy2(package, output / ASSETS[0])
    shutil.copy2(root / "scripts/migrate-v2.sh", output / ASSETS[1])
    (output / "SHA256SUMS").write_text(checksums(output))
    notes = f"""VSC {version}

{changes}

下载 `vsc.alfredworkflow` 并导入 Alfred，支持 Apple Silicon / Intel Mac，无需安装语言运行时。
已有 V3 正式版直接更新，配置保留，无需再次迁移。`migrate-v2.sh` 仅供 V2 用户迁移。
`SHA256SUMS` 可用于校验这两个附件。

发布前已通过 Linux / macOS 自动验收；实际 Alfred 界面及 IDE 启动需在 Mac 上验证。
本包未做 Developer ID 公证，安装和 Gatekeeper 处理见
[安装指南](https://github.com/HarveyZgit/alfred.vsc/blob/{version}/docs/INSTALLATION.zh-CN.md)，
迁移见[迁移指南](https://github.com/HarveyZgit/alfred.vsc/blob/{version}/docs/MIGRATION.zh-CN.md)。
"""
    (output / "RELEASE_NOTES.md").write_text(notes)
    return version


def gh(*args):
    try:
        return subprocess.run(["gh", *args], check=True, capture_output=True, text=True).stdout
    except subprocess.CalledProcessError as error:
        raise RuntimeError(f"GitHub CLI failed: {error.stderr.strip()}") from error


def publish(root, directory, tag, repository):
    release_metadata(root, tag)
    if (directory / "SHA256SUMS").read_text() != checksums(directory):
        raise ValueError("Release asset checksum mismatch")
    # API failures must stop publication, never be mistaken for a missing release.
    pages = json.loads(gh("api", "--paginate", "--slurp", f"repos/{repository}/releases"))
    existing = next(
        (release for page in pages for release in page if release["tag_name"] == tag), None
    )
    if existing and not existing["draft"]:
        raise ValueError(f"Release {tag} is already published; refusing to replace assets")
    if existing and any(asset["name"] not in ASSETS for asset in existing["assets"]):
        raise ValueError("Existing draft has unexpected assets; review it before retrying")
    common = ("--repo", repository)
    notes = ("--title", f"VSC {tag}", "--notes-file", str(directory / "RELEASE_NOTES.md"))
    if existing:
        gh("release", "edit", tag, *common, *notes)
    else:
        gh("release", "create", tag, *common, "--verify-tag", "--draft", *notes)
    gh("release", "upload", tag, *common, "--clobber", *(str(directory / name) for name in ASSETS))
    # Verify the uploaded bytes before making the release public, including on retry.
    with tempfile.TemporaryDirectory(prefix="vsc-release-verify-") as temporary:
        gh("release", "download", tag, *common, "--dir", temporary)
        downloaded = Path(temporary)
        for name in ASSETS:
            if (downloaded / name).read_bytes() != (directory / name).read_bytes():
                raise ValueError(f"Uploaded {name} differs from the tested release asset")
    gh("release", "edit", tag, *common, "--draft=false", "--prerelease=false", "--latest")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("command", choices=("check", "prepare", "publish"))
    parser.add_argument("--tag")
    parser.add_argument("--build", type=Path, default=ROOT / "build")
    parser.add_argument("--directory", type=Path, default=ROOT / "build/release")
    parser.add_argument("--repo", default="HarveyZgit/alfred.vsc")
    args = parser.parse_args()
    if args.command == "check":
        print(release_metadata(ROOT, args.tag)[0])
    elif args.command == "prepare":
        print(prepare(ROOT, args.build, args.directory, args.tag))
    else:
        if not args.tag:
            parser.error("publish requires --tag")
        publish(ROOT, args.directory, args.tag, args.repo)


if __name__ == "__main__":
    main()
