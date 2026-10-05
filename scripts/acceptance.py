#!/usr/bin/env python3
"""Exercise the compiled CLI against real SQLite, Git, subprocesses and typing bursts."""

import argparse
from concurrent.futures import ThreadPoolExecutor
import hashlib
import json
import math
import os
from pathlib import Path
import sqlite3
import statistics
import subprocess
import tempfile
import time
from urllib.parse import unquote, urlsplit


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--binary", type=Path, default=Path("build/vsc-linux-amd64"))
    parser.add_argument("--output", type=Path, default=Path("build/acceptance.json"))
    parser.add_argument("--benchmark", action="store_true")
    args = parser.parse_args()
    binary = str(args.binary.resolve())
    report = {
        "binary_sha256": hashlib.sha256(Path(binary).read_bytes()).hexdigest(),
        "checks": [],
        "benchmarks": [],
    }
    with tempfile.TemporaryDirectory(prefix="vsc-acceptance-") as tmp:
        base = Path(tmp).resolve()
        root = base / "projects"
        root.mkdir()
        env = {k: v for k, v in os.environ.items() if not k.startswith(("VSC_", "alfred_"))}
        env.update(
            VSC_DATA_DIR=str(base / "data"),
            VSC_CACHE_DIR=str(base / "cache"),
            VSC_DB_PATH=str(base / "state.vscdb"),
            VSC_DIRECTORIES=str(root),
            VSC_NO_BACKGROUND="1",
        )

        def run(*words, success=True, environment=None):
            result = subprocess.run(
                [binary, *words], env=environment or env, text=True, capture_output=True, timeout=10
            )
            assert (result.returncode == 0) == success, (words, result.returncode, result.stderr)
            return result

        def query(text=""):
            return json.loads(run("query", text).stdout)

        def records(feedback):
            def canonical(target):
                parsed = urlsplit(target)
                return Path(unquote(parsed.path)).as_uri() if parsed.scheme == "file" else target

            return {
                canonical(json.loads(i["arg"])["target"]): i
                for i in feedback["items"]
                if i.get("uid")
            }

        def git(path, *words):
            return subprocess.run(
                ["git", "-C", str(path), *words], check=True, capture_output=True, text=True
            ).stdout.strip()

        local = root / "项目 space; literal"
        local.mkdir()
        git(local, "init", "-b", "main")
        git(
            local,
            "-c",
            "user.name=Test",
            "-c",
            "user.email=test@example.invalid",
            "commit",
            "--allow-empty",
            "-m",
            "fixture",
        )
        tree = root / "linked"
        git(local, "worktree", "add", "-b", "worktree-branch", str(tree))
        plain = base / "plain"
        plain.mkdir()
        remote = "vscode-remote://ssh-remote+devbox/srv/remote%20project"
        container = "vscode-remote://dev-container+opaque/workspaces/project"
        db = sqlite3.connect(env["VSC_DB_PATH"])
        db.execute("PRAGMA journal_mode=WAL")
        db.execute("CREATE TABLE ItemTable (key TEXT PRIMARY KEY, value TEXT)")

        def history(entries):
            db.execute(
                "INSERT OR REPLACE INTO ItemTable VALUES (?,?)",
                ("recently.opened", json.dumps({"entries": entries})),
            )
            db.commit()

        entries = [{"folderUri": p} for p in (local.as_uri(), plain.as_uri(), remote, container)]
        entries += [
            {"fileUri": local.as_uri() + "/ignored.txt"},
            {"fileUri": remote + "/ignored.txt"},
            {"workspace": {"configPath": plain.as_uri() + "/x.code-workspace"}},
        ]
        history(entries)
        run("index")
        rows = records(query())
        assert set(rows) == {local.as_uri(), plain.as_uri(), tree.as_uri(), remote, container}, (
            list(rows)
        )
        assert (
            "main" in rows[local.as_uri()]["subtitle"]
            and "worktree-branch" in rows[tree.as_uri()]["subtitle"]
        )
        assert all("⎇" not in rows[r]["subtitle"] for r in (remote, container))
        report["checks"].append(
            "source union, exclusions, real Git worktree, remote without branches"
        )
        git(local, "switch", "-c", "fresh-branch")
        assert "fresh-branch" in records(query("literal"))[local.as_uri()]["subtitle"]
        history(entries[1:])
        assert local.as_uri() in records(query())  # still discovered by the root index
        assert list(records(query()))[0] == plain.as_uri()
        report["checks"].append(
            "branch changes and committed WAL history visible on next invocation"
        )
        run("hide", "--target", local.as_uri())
        run("rebuild")
        assert local.as_uri() not in records(query())
        run("restore-all")
        assert local.as_uri() in records(query())
        with ThreadPoolExecutor(max_workers=8) as pool:
            list(
                pool.map(
                    lambda n: run("hide", "--target", f"vscode-remote://ssh-remote+h/srv/{n}"),
                    range(30),
                )
            )
        prefs = json.loads((base / "data/preferences.json").read_text())
        assert len(prefs["hidden"]) == 30
        report["checks"].append(
            "hide, restore and 30 concurrent preference writers without lost updates"
        )
        argv = base / "argv.json"
        fake = base / "fake editor"
        fake.write_text(
            '#!/usr/bin/env python3\nimport json, os, sys\nfrom pathlib import Path\nPath(os.environ["ACCEPT_ARGV"]).write_text(json.dumps(sys.argv[1:]))\n'
        )
        fake.chmod(0o755)
        env["ACCEPT_ARGV"] = str(argv)
        for editor in ("vscode", "cursor", "trae", "zed"):
            env["VSC_EDITOR_" + editor.upper()] = str(fake)
            run(
                "open", "--target", local.as_uri(), "--editor", editor, "--new-window", "--remember"
            )
            flag = "--new" if editor == "zed" else "--new-window"
            assert json.loads(argv.read_text()) == [flag, "--", str(local)]
            run("open", "--target", remote, "--editor", editor)
            want = (
                ["ssh://devbox/srv/remote%20project"]
                if editor == "zed"
                else ["--folder-uri", remote]
            )
            assert json.loads(argv.read_text()) == want
        run("open", "--target", container, "--editor", "zed", success=False)
        fake.write_text("#!/bin/sh\nexit 7\n")
        assert (
            "exit status 7"
            in run("open", "--target", local.as_uri(), "--editor", "vscode", success=False).stderr
        )
        report["checks"].append(
            "four editor adapters, exact argv, Unicode/spaces, remembered editor, failure propagation"
        )
        db.execute("BEGIN EXCLUSIVE")
        db.execute("UPDATE ItemTable SET value='{}'")
        assert len(records(query())) == 5  # committed WAL snapshot, not uncommitted writes
        db.rollback()
        history([])
        assert set(records(query())) == {local.as_uri(), tree.as_uri()}
        report["checks"].append(
            "SQLite read-only committed view, empty source removes stale history"
        )
        assert any(i.get("autocomplete", "").endswith("/") for i in query("/")["items"])
        report["checks"].append("directory browsing")
        (base / "cache/index.cache").unlink()
        background = dict(env, VSC_NO_BACKGROUND="0")
        first = json.loads(run("query", "", environment=background).stdout)
        assert first.get("rerun") == 0.5
        deadline = time.monotonic() + 5
        while not (base / "cache/index.cache").exists() and time.monotonic() < deadline:
            time.sleep(0.02)
        assert (base / "cache/index.cache").exists()
        assert tree.as_uri() in records(query())
        report["checks"].append("real detached background scanner and completed index")
        # A native CLI must run with no interpreters or compiler in PATH.
        isolated = dict(env, PATH="/nonexistent")
        assert json.loads(run("query", "literal", environment=isolated).stdout)["items"][0]["uid"]
        report["checks"].append("query runs without Python, Node, Go, sqlite3 or git on PATH")
        # Exercise the distributed shell entry point and actual native importer.
        legacy = base / "old workflow"
        legacy.mkdir()
        outscope = base / "outside"
        outscope.mkdir()
        git(outscope, "init", "-b", "main")
        source = legacy / ".records.cache.json"
        source.write_text(
            json.dumps(
                {
                    "records": [
                        {"path": "file://" + str(local), "type": "folder"},
                        {"path": outscope.as_uri(), "type": "folder"},
                    ],
                    "trash": {
                        "old": {"path": "file://" + str(local)},
                        "later": {"path": outscope.as_uri()},
                    },
                    "legacy_extra": "retained",
                }
            )
        )
        before = source.read_bytes()
        script = Path(__file__).resolve().parent / "migrate-v2.sh"
        migration_env = dict(env, VSC_BINARY=binary)

        def migrate(apply=False):
            words = ["sh", str(script), "--source", str(legacy), "--data-dir", env["VSC_DATA_DIR"]]
            if apply:
                words.append("--apply")
            result = subprocess.run(
                words, env=migration_env, capture_output=True, text=True, check=True, timeout=10
            )
            return json.loads(result.stdout)

        preview = migrate()
        assert not preview["applied"] and len(preview["unmigrated"]) == 2
        assert not Path(preview["backup"]).exists()
        migrated = migrate(True)
        assert migrated["applied"] and migrated["hidden_imported"] == 1
        assert (
            (Path(migrated["backup"]) / "legacy-records.json").read_bytes()
            == before
            == source.read_bytes()
        )
        assert local.as_uri() not in records(query())
        run("unhide", "--target", local.as_uri())
        assert migrate(True)["already_applied"] and local.as_uri() in records(query())
        history([{"folderUri": outscope.as_uri()}])
        assert not migrate()["already_applied"]
        assert not migrate(True)["unmigrated"]
        assert outscope.as_uri() not in records(query())
        assert local.as_uri() in records(query())
        report["checks"].append(
            "migration shell: preview, scope exclusions and guidance, exact backup, rerun, user unhide preservation and incremental import"
        )

        if args.benchmark:

            def stats(samples):
                ordered = sorted(samples)
                return {
                    "p50_ms": round(statistics.median(samples), 3),
                    "p95_ms": round(ordered[math.ceil(len(samples) * 0.95) - 1], 3),
                    "max_ms": round(max(samples), 3),
                }

            def timed(text):
                start = time.perf_counter()
                result = query(text)
                elapsed = (time.perf_counter() - start) * 1000
                assert len(result["items"]) <= 50
                return elapsed

            for count in (1000, 10000):
                projects = []
                perfroot = base / f"perf{count}"
                for n in range(count):
                    p = perfroot / f"project-{n:05}"
                    (p / ".git").mkdir(parents=True)
                    (p / ".git/HEAD").write_text("ref: refs/heads/perf-main\n")
                    projects.append({"folderUri": p.as_uri()})
                history(projects)
                cold = timed("project")
                for _ in range(3):
                    timed("project")
                samples = [
                    timed(("p", "pr", "pro", "proj", "project", "project-0")[n % 6])
                    for n in range(36)
                ]
                burst = []
                with ThreadPoolExecutor(max_workers=8) as pool:
                    futures = []
                    start = time.perf_counter()
                    for n in range(40):
                        due = start + n * 0.03
                        time.sleep(max(0, due - time.perf_counter()))
                        futures.append(
                            pool.submit(timed, ("p", "pro", "project", "project-0")[n % 4])
                        )
                    burst = [f.result() for f in futures]
                report["benchmarks"].append(
                    {
                        "projects": count,
                        "first_source_import_ms": round(cold, 3),
                        "warm_queries": stats(samples),
                        "typing_30ms_interval": stats(burst),
                        "queries_over_30ms": sum(t > 30 for t in burst),
                    }
                )
        db.close()
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(json.dumps(report, ensure_ascii=False, indent=2) + "\n")
    print(json.dumps(report, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
