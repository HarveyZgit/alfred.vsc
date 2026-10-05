#!/usr/bin/env python3
"""Exercise the embedded management server and query in real native processes."""

import argparse
import hashlib
import json
import os
from pathlib import Path
import selectors
import sqlite3
import subprocess
import tempfile
import traceback
import urllib.error
import urllib.parse
import urllib.request


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--binary", type=Path, default=Path("build/vsc-linux-amd64"))
    parser.add_argument("--output", type=Path, default=Path("build/panel-acceptance.json"))
    args = parser.parse_args()
    binary = args.binary.resolve()
    checks = []
    with tempfile.TemporaryDirectory(prefix="vsc-panel-") as tmp:
        base = Path(tmp).resolve()
        root = base / "projects"
        repo = root / "panel 项目"
        (repo / ".git").mkdir(parents=True)
        (repo / ".git/HEAD").write_text("ref: refs/heads/main\n")
        missing = base / "missing"
        remote = "vscode-remote://ssh-remote+example.invalid" + urllib.parse.quote(str(repo))
        db = sqlite3.connect(base / "state.vscdb")
        db.execute("CREATE TABLE ItemTable (key TEXT PRIMARY KEY, value TEXT)")
        db.execute(
            "INSERT INTO ItemTable VALUES (?,?)",
            (
                "recently.opened",
                json.dumps(
                    {
                        "entries": [
                            {"folderUri": str(uri)}
                            for uri in (repo.as_uri(), missing.as_uri(), remote)
                        ]
                    }
                ),
            ),
        )
        db.commit()
        db.close()
        env = {k: v for k, v in os.environ.items() if not k.startswith(("VSC_", "alfred_"))}
        env.update(
            VSC_DATA_DIR=str(base / "Workflow Data/com.harvey.alfredapp.vsc.v3"),
            VSC_DB_PATH=str(base / "state.vscdb"),
            VSC_DIRECTORIES=str(base / "wrong"),
            VSC_DEFAULT_EDITOR="vscode",
            VSC_NO_BACKGROUND="1",
        )
        # No Node, Python, Go, shell or editor executable is available to the server.
        env["PATH"] = str(base / "empty-path")
        process = subprocess.Popen(
            [str(binary), "manage", "--serve"],
            env=env,
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
        )
        try:
            with selectors.DefaultSelector() as selector:
                selector.register(process.stdout, selectors.EVENT_READ)
                assert selector.select(5), "server failed to announce startup"
            address = process.stdout.readline().strip()
            parsed = urllib.parse.urlsplit(address)
            assert parsed.hostname == "127.0.0.1" and len(parsed.fragment) == 64
            origin = urllib.parse.urlunsplit((parsed.scheme, parsed.netloc, "", "", ""))
            opener = urllib.request.build_opener(urllib.request.ProxyHandler({}))

            def request(path, body=None, auth=True, extra=None, status=200):
                headers = {"Content-Type": "application/json"}
                if auth:
                    headers["Authorization"] = "Bearer " + parsed.fragment
                headers.update(extra or {})
                req = urllib.request.Request(
                    origin + path, None if body is None else json.dumps(body).encode(), headers
                )
                try:
                    response = opener.open(req, timeout=30)
                except urllib.error.HTTPError as error:
                    response = error
                with response:
                    payload = response.read()
                    assert response.status == status, (path, response.status, payload)
                    return (
                        json.loads(payload)
                        if response.headers.get_content_type() == "application/json"
                        else payload
                    )

            assert b"app.js" in request("/", auth=False)
            assert b'type="module"' in request("/", auth=False)
            for module in ("app", "api", "dom", "records", "settings", "migration"):
                assert request("/" + module + ".js", auth=False)
            assert b"fetch(" in request("/api.js", auth=False)
            assert request("/style.css", auth=False)
            request("/api/records", auth=False, status=401)
            request(
                "/api/settings",
                {"roots": [], "editor": "zed", "editors": {}},
                extra={"Origin": "https://example.invalid"},
                status=403,
            )
            request("/api/records", extra={"Host": "example.invalid"}, status=403)
            checks.append("embedded UI without language runtimes; token, origin and Host guards")
            settings = {"roots": [str(root)], "editor": "cursor", "editors": {}}
            request("/api/settings", settings)
            result = request("/api/settings")
            assert result["roots"] == [str(root)] and result["editor"] == "cursor"
            result = request("/api/rebuild", {})
            assert len(result["projects"]) == 1
            checks.append("panel overrides Alfred variables; explicit Git indexing")
            rows = request("/api/records")["items"]
            row = next(r for r in rows if r["uri"] == repo.as_uri())
            assert row["branch"] == "main"
            assert next(r for r in rows if r["uri"] == missing.as_uri())["status"] == "目录不存在"
            assert "branch" not in next(r for r in rows if r["uri"] == remote)
            (repo / ".git/HEAD").write_text("ref: refs/heads/panel-fresh\n")
            assert (
                next(r for r in request("/api/records")["items"] if r["id"] == row["id"])["branch"]
                == "panel-fresh"
            )
            checks.append(
                "fresh local branch; explicit missing-directory reason; remote without branch"
            )
            request("/api/record", {"id": row["id"], "action": "hidden", "enabled": True})
            assert request("/api/records?filter=hidden")["total"] == 1

            def query():
                return json.loads(
                    subprocess.check_output(
                        [str(binary), "query", "panel"], env=env, text=True, timeout=5
                    )
                )

            assert all(
                json.loads(item["arg"])["target"] != repo.as_uri()
                for item in query()["items"]
                if item.get("uid")
            )
            request("/api/record", {"id": row["id"], "action": "hidden", "enabled": False})
            assert any(
                json.loads(item["arg"])["target"] == repo.as_uri()
                for item in query()["items"]
                if item.get("uid")
            )
            request("/api/record", {"id": row["id"], "action": "pinned", "enabled": True})
            assert request("/api/records?filter=pinned")["total"] == 1
            request("/api/record", {"id": row["id"], "action": "editor", "editor": "zed"})
            assert (
                next(r for r in request("/api/records")["items"] if r["id"] == row["id"])["editor"]
                == "zed"
            )
            checks.append(
                "hide/restore changes actual CLI results; pin and per-project IDE persist"
            )
            source = base / "legacy.json"
            source.write_text(
                json.dumps(
                    {
                        "records": [
                            {"path": str(repo), "type": "folder"},
                            {"path": str(base / "outside"), "type": "folder"},
                        ],
                        "trash": {},
                    }
                )
            )
            preview = request("/api/migrate", {"source": str(source), "apply": False})
            assert (
                len(preview["unmigrated"]) == 1
                and not (Path(env["VSC_DATA_DIR"]) / "migrations").exists()
            )
            applied = request("/api/migrate", {"source": str(source), "apply": True})
            assert applied["applied"] and len(applied["unmigrated"]) == 1
            saved = request("/api/migrations")
            assert len(saved) == 1 and not saved[0]["historical"] and saved[0]["report"]["applied"]
            assert request("/api/migrate", {"source": str(source), "apply": True})[
                "already_applied"
            ]
            checks.append(
                "read-only preview, scoped apply, saved latest result and idempotent retry"
            )
            request("/api/settings/reset", {})
            assert request("/api/settings")["roots"] == [str(base / "wrong")]
            request("/api/close", {})
            assert process.wait(timeout=5) == 0
            checks.append("reset settings and clean server shutdown")
            # An updated package uses the same Alfred data identity, without
            # requiring VSC_DATA_DIR, a dev-mode override or a new migration.
            persistent = Path(env["VSC_DATA_DIR"])
            preserved = {
                name: (persistent / name).read_bytes()
                for name in ("preferences.json", "config.json")
            }
            upgraded_env = dict(env, alfred_workflow_data=str(persistent))
            del upgraded_env["VSC_DATA_DIR"]
            doctor = json.loads(
                subprocess.check_output(
                    [str(binary), "doctor"], env=upgraded_env, text=True, timeout=5
                )
            )
            assert doctor["data_dir"] == str(persistent)
            result = json.loads(
                subprocess.check_output(
                    [str(binary), "query", "panel"], env=upgraded_env, text=True, timeout=5
                )
            )
            item = next(item for item in result["items"] if item.get("uid") == row["id"])
            assert json.loads(item["arg"])["editor"] == "zed"
            assert all((persistent / name).read_bytes() == data for name, data in preserved.items())
            checks.append(
                "release Alfred data directory reuses preferences/config without migration or dev overrides"
            )
            # Verify the short-lived Alfred command starts a detached server and
            # passes its URL to the OS browser opener without shell evaluation.
            bindir = base / "bin"
            bindir.mkdir()
            browser = bindir / ("open" if os.uname().sysname == "Darwin" else "xdg-open")
            browser.write_text("""#!/bin/sh
printf '%s' "$1" > "$VSC_TEST_BROWSER_CAPTURE"
""")
            browser.chmod(0o755)
            launched = dict(
                env, PATH=str(bindir), VSC_TEST_BROWSER_CAPTURE=str(base / "browser-url")
            )
            result = subprocess.run(
                [str(binary), "manage"], env=launched, capture_output=True, text=True, timeout=5
            )
            assert result.returncode == 0, result.stderr
            detached = urllib.parse.urlsplit((base / "browser-url").read_text())
            close_url = urllib.parse.urlunsplit(
                (detached.scheme, detached.netloc, "/api/close", "", "")
            )
            req = urllib.request.Request(
                close_url,
                b"{}",
                {
                    "Authorization": "Bearer " + detached.fragment,
                    "Content-Type": "application/json",
                },
            )
            with opener.open(req, timeout=5) as response:
                assert json.load(response)["ok"]
            checks.append(
                "Alfred launcher exits promptly; detached panel URL reaches fake OS browser opener"
            )
        finally:
            if process.poll() is None:
                process.terminate()
                process.wait(timeout=5)
    report = {"binary_sha256": hashlib.sha256(binary.read_bytes()).hexdigest(), "checks": checks}
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(json.dumps(report, indent=2, ensure_ascii=False) + "\n")
    print(json.dumps(report, indent=2, ensure_ascii=False))


if __name__ == "__main__":
    try:
        main()
    except Exception as error:
        # Keep CI failures actionable in the Checks API, including bare asserts.
        # Do not include request headers or the local server session token.
        if os.environ.get("GITHUB_ACTIONS") == "true":
            frames = traceback.extract_tb(error.__traceback__)
            frame = next(
                (frame for frame in reversed(frames) if frame.filename == __file__), frames[-1]
            )
            message = f"{type(error).__name__}: {error} | {frame.line}"
            message = message.replace("%", "%25").replace("\r", "%0D").replace("\n", "%0A")
            print(f"::error file=scripts/panel_acceptance.py,line={frame.lineno}::{message}")
        raise
