"""Exercise release guards and failed/retried publication without network access."""

import importlib.util
import json
from pathlib import Path
import plistlib
import shutil
import subprocess
import tempfile
import unittest
from unittest.mock import patch
import zipfile

ROOT = Path(__file__).resolve().parents[2]
SPEC = importlib.util.spec_from_file_location("release", ROOT / "scripts/release.py")
release = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(release)


class ReleaseTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name)
        for name in (
            "package.json",
            "web/package.json",
            "public/info.plist",
            "internal/workflow/config.go",
            "THIRD_PARTY_NOTICES.txt",
            "CHANGELOG.md",
            "scripts/migrate-v2.sh",
        ):
            destination = self.root / name
            destination.parent.mkdir(parents=True, exist_ok=True)
            shutil.copy2(ROOT / name, destination)
        self.version = json.loads((self.root / "package.json").read_text())["version"]
        self.build = self.root / "build"
        self.build.mkdir()
        self.output = self.root / "release"
        self.make_package()

    def make_package(self, **overrides):
        config = {"version": self.version, "bundleid": "com.harvey.alfredapp.vsc.v3", **overrides}
        with zipfile.ZipFile(self.build / "vsc.alfredworkflow", "w") as archive:
            archive.writestr("info.plist", plistlib.dumps(config))
            archive.write(self.root / "scripts/migrate-v2.sh", "scripts/migrate-v2.sh")
        shutil.copy2(self.root / "scripts/migrate-v2.sh", self.build / "migrate-v2.sh")
        (self.build / "SHA256SUMS").write_text(release.checksums(self.build))

    def prepare(self):
        release.prepare(self.root, self.build, self.output, self.version)

    def test_stages_only_production_assets_and_current_notes(self):
        self.prepare()
        self.assertEqual(
            {p.name for p in self.output.iterdir()}, {*release.ASSETS, "RELEASE_NOTES.md"}
        )
        self.assertEqual((self.output / "SHA256SUMS").read_text(), release.checksums(self.output))
        self.assertEqual(
            (self.output / "vsc.alfredworkflow").read_bytes(),
            (self.build / "vsc.alfredworkflow").read_bytes(),
        )
        notes = (self.output / "RELEASE_NOTES.md").read_text()
        self.assertIn(f"VSC {self.version}", notes)
        self.assertNotIn("## v3.0.0", notes)

    def test_rejects_wrong_tags_and_source_versions(self):
        for tag in ("v" + self.version, "0.0.0", self.version + "-rc.1"):
            with self.subTest(tag=tag), self.assertRaises(ValueError):
                release.release_metadata(self.root, tag)
        (self.root / "web/package.json").write_text('{"version":"0.0.0"}')
        with self.assertRaisesRegex(ValueError, "web/package.json"):
            release.release_metadata(self.root)

    def test_requires_changelog(self):
        (self.root / "CHANGELOG.md").write_text("# Changelog\n")
        with self.assertRaisesRegex(ValueError, "CHANGELOG"):
            self.prepare()

    def test_rejects_dev_or_stale_package(self):
        for overrides in ({"bundleid": "com.harvey.alfredapp.vsc.v3.dev"}, {"version": "0.0.0"}):
            with self.subTest(overrides=overrides):
                self.make_package(**overrides)
                with self.assertRaisesRegex(ValueError, "production bundle"):
                    self.prepare()

    def test_rejects_changed_migration_script_or_build_checksum(self):
        (self.build / "SHA256SUMS").write_text("")
        with self.assertRaisesRegex(ValueError, "checksum"):
            self.prepare()
        self.make_package()
        (self.root / "scripts/migrate-v2.sh").write_text("changed")
        with self.assertRaisesRegex(ValueError, "migration script"):
            self.prepare()

    def mock_github(self, existing=None, fail=None, corrupt=False):
        def execute(*args):
            if args[0] == "api":
                return json.dumps([[existing] if existing else []])
            if args[1] == fail:
                raise subprocess.CalledProcessError(1, ["gh", *args])
            if args[1] == "download":
                target = Path(args[args.index("--dir") + 1])
                for name in release.ASSETS:
                    shutil.copy2(self.output / name, target / name)
                if corrupt:
                    (target / "vsc.alfredworkflow").write_bytes(b"corrupted upload")
            return ""

        return patch.object(release, "gh", side_effect=execute)

    def publish(self):
        release.publish(self.root, self.output, self.version, "example/repository")

    def test_create_upload_verify_publish_order(self):
        self.prepare()
        with self.mock_github() as mocked:
            self.publish()
        calls = [call.args for call in mocked.call_args_list]
        self.assertEqual([args[1] for args in calls[1:]], ["create", "upload", "download", "edit"])
        self.assertIn("--draft", calls[1])
        self.assertIn("--verify-tag", calls[1])
        self.assertIn("--draft=false", calls[-1])

    def test_partial_upload_retries_existing_draft(self):
        self.prepare()
        existing = {"tag_name": self.version, "draft": True, "assets": [{"name": "migrate-v2.sh"}]}
        with self.mock_github(existing) as mocked:
            self.publish()
        calls = [call.args for call in mocked.call_args_list]
        self.assertFalse(any(args[:2] == ("release", "create") for args in calls))
        self.assertIn("--clobber", calls[2])
        self.assertIn("--draft=false", calls[-1])

    def test_upload_or_verification_failure_leaves_draft(self):
        self.prepare()
        for failure in ({"fail": "upload"}, {"fail": "download"}, {"corrupt": True}):
            with self.subTest(failure=failure), self.mock_github(**failure) as mocked:
                with self.assertRaises((subprocess.CalledProcessError, ValueError)):
                    self.publish()
                self.assertFalse(
                    any("--draft=false" in call.args for call in mocked.call_args_list)
                )

    def test_published_release_is_never_modified(self):
        self.prepare()
        existing = {"tag_name": self.version, "draft": False, "assets": []}
        with self.mock_github(existing) as mocked:
            with self.assertRaisesRegex(ValueError, "already published"):
                self.publish()
            self.assertEqual(mocked.call_count, 1)

    def test_metadata_api_failure_does_not_create_release(self):
        self.prepare()
        with patch.object(
            release, "gh", side_effect=subprocess.CalledProcessError(1, "gh")
        ) as mocked:
            with self.assertRaises(subprocess.CalledProcessError):
                self.publish()
            self.assertEqual(mocked.call_count, 1)

    def test_existing_draft_with_unexpected_asset_is_not_modified(self):
        self.prepare()
        existing = {
            "tag_name": self.version,
            "draft": True,
            "assets": [{"name": "manual-package.zip"}],
        }
        with self.mock_github(existing) as mocked:
            with self.assertRaisesRegex(ValueError, "unexpected assets"):
                self.publish()
            self.assertEqual(mocked.call_count, 1)

    def test_tampered_staged_assets_never_reach_github(self):
        self.prepare()
        (self.output / "vsc.alfredworkflow").write_bytes(b"changed")
        with patch.object(release, "gh") as mocked:
            with self.assertRaisesRegex(ValueError, "checksum"):
                self.publish()
            mocked.assert_not_called()


if __name__ == "__main__":
    unittest.main()
