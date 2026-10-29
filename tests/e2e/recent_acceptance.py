"""Global selection order acceptance using independent CLI processes and synthetic data."""

from concurrent.futures import ThreadPoolExecutor
import hashlib
import json
import os
from pathlib import Path
import sqlite3
import subprocess
import tempfile


def accept_recent(binary):
    with tempfile.TemporaryDirectory(prefix="vsc-recent-") as temporary:
        base = Path(temporary).resolve()
        env = {k: v for k, v in os.environ.items() if not k.startswith(("VSC_", "alfred_"))}
        env.update(
            VSC_DATA_DIR=str(base / "data"), VSC_CACHE_DIR=str(base / "cache"),
            VSC_DB_PATH=str(base / "state.vscdb"), VSC_DIRECTORIES="", VSC_NO_BACKGROUND="1",
        )
        editor = base / "fake-editor"
        editor.write_text("#!/bin/sh\nexit 0\n")
        editor.chmod(0o755)
        for name in ("VSCODE", "CURSOR", "TRAE", "ZED"):
            env["VSC_EDITOR_" + name] = str(editor)
        db = sqlite3.connect(env["VSC_DB_PATH"])
        db.execute("CREATE TABLE ItemTable (key TEXT PRIMARY KEY,value TEXT)")
        prefs_path = base / "data/preferences.json"

        def run(*words, success=True):
            result = subprocess.run([binary, *words], env=env, text=True, capture_output=True, timeout=10)
            assert (result.returncode == 0) == success, (words, result.stderr)
            return result

        def history(uris):
            db.execute("INSERT OR REPLACE INTO ItemTable VALUES (?,?)", (
                "recently.opened", json.dumps({"entries": [{"folderUri": uri} for uri in uris]}),
            ))
            db.commit()

        def query(text="hub"):
            feedback = json.loads(run("query", text).stdout)
            assert feedback["skipknowledge"] is True
            return [item for item in feedback["items"] if item.get("uid")]

        def recent():
            return json.loads(prefs_path.read_text())["recent_projects"]

        def identity(uri):
            return hashlib.sha256(uri.encode()).hexdigest()[:32]

        short, chosen, other = (base / name for name in ("github", "example-material-hub", "other"))
        for path in (short, chosen, other):
            path.mkdir()
        uris = [short.as_uri(), chosen.as_uri(), other.as_uri()]
        history(uris)
        initial = query()
        assert [item["title"] for item in initial] == [short.name, chosen.name]
        chosen_id, short_id = identity(chosen.as_uri()), identity(short.as_uri())
        # A new process reads persisted order after each invocation.
        run("open", "--request", initial[1]["arg"])
        assert query()[0]["uid"] == chosen_id
        assert query("")[0]["uid"] == chosen_id
        assert query("material")[0]["uid"] == chosen_id
        assert [item["title"] for item in query("other")] == [other.name]
        assert [item["title"] for item in query("github")] == [short.name]
        run("open", "--request", initial[0]["mods"]["shift"]["arg"])
        assert query()[0]["uid"] == short_id
        choices = json.loads(run("editors", "--target", chosen.as_uri()).stdout)["items"]
        for choice in choices:
            run("open", "--remember", "--request", choice["arg"])
            assert query()[0]["uid"] == chosen_id
            assert recent() == [chosen_id, short_id]
        # Default/explicit CLI routes share the same canonical project identity.
        alias = base / "alias"
        alias.symlink_to(chosen, target_is_directory=True)
        run("open", "--target", alias.as_uri())
        assert recent() == [chosen_id, short_id]
        run("pin", "--target", short.as_uri())
        assert query()[0]["uid"] == short_id
        run("pin", "--target", short.as_uri())
        assert query()[0]["uid"] == chosen_id
        run("hide", "--target", chosen.as_uri())
        assert chosen_id not in [item["uid"] for item in query()]
        run("unhide", "--target", chosen.as_uri())
        assert query()[0]["uid"] == chosen_id

        before = prefs_path.read_bytes()
        run("open", "--dry-run", "--target", short.as_uri())
        assert prefs_path.read_bytes() == before
        editor.write_text("#!/bin/sh\nexit 7\n")
        run("open", "--remember", "--target", short.as_uri(), success=False)
        assert prefs_path.read_bytes() == before
        editor.write_text("#!/bin/sh\nexit 0\n")
        chosen.rmdir()
        run("open", "--target", chosen.as_uri(), success=False)
        assert prefs_path.read_bytes() == before
        assert next(item for item in query() if item["uid"] == chosen_id)["valid"] is False
        history([short.as_uri(), other.as_uri()])
        assert chosen_id not in [item["uid"] for item in query()]
        chosen.mkdir()
        # Direct paths record an ID but never create a source entry or unhide one.
        direct = base / "unlisted"
        direct.mkdir()
        candidate = json.loads(run("query", str(direct)).stdout)["items"][0]
        run("open", "--request", candidate["arg"])
        assert recent()[0] == identity(direct.as_uri())
        assert identity(direct.as_uri()) not in [item["uid"] for item in query("")]
        # Type filtering still applies to globally recent remote selections.
        remote = "vscode-remote://ssh-remote+example/srv/example-hub"
        history(uris + [remote])
        run("open", "--target", remote)
        assert query()[0]["uid"] == identity(remote)
        assert identity(remote) not in [item["uid"] for item in query("d hub")]
        # The selected candidate would be excluded if the old top-50 limit ran first.
        prefixes = []
        for i in range(60):
            path = base / f"hub-{i:03}"
            path.mkdir()
            prefixes.append(path.as_uri())
        history(prefixes + uris)
        run("open", "--target", chosen.as_uri())
        limited = query()
        assert len(limited) == 50 and limited[0]["uid"] == chosen_id
        # Concurrent process writers preserve all selections and other preferences.
        run("pin", "--target", short.as_uri())
        with ThreadPoolExecutor(max_workers=8) as pool:
            list(pool.map(lambda uri: run("open", "--target", uri), prefixes[:30]))
        state = json.loads(prefs_path.read_text())
        assert len(state["recent_projects"]) == len(set(state["recent_projects"]))
        assert {identity(uri) for uri in prefixes[:30]}.issubset(state["recent_projects"])
        assert state["pinned"][short_id] is True
        assert state["editors"][chosen_id] == "cursor"
        # A broken preference file is not silently replaced after launching an IDE.
        prefs_path.write_text('{"version":1,"recent_projects":false}')
        broken = prefs_path.read_bytes()
        result = run("open", "--target", short.as_uri(), "--editor", "vscode", success=False)
        assert "保存项目偏好失败" in result.stderr and prefs_path.read_bytes() == broken
        db.close()
    return [
        "global MRU across CLI processes, empty/different queries, all IDEs, shift and remembered IDE",
        "MRU deduplication, canonical symlinks, remote identity, failure/dry-run/missing target unchanged",
        "pin priority, exact-name/type/hidden/source filtering, ordering before top-50 limit",
        "30 concurrent MRU writers retain selections and preferences; malformed state not overwritten",
    ]
