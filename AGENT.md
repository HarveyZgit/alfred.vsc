# Development guide

VSC 3 is an Alfred 5 workflow with a Go native CLI. Read README.md and docs/ARCHITECTURE.zh-CN.md for the scope. Use the existing isolated checkout; do not create a worktree unless explicitly requested.

- cmd/vsc: CLI entry point.
- internal/workflow: one internal package, with focused files for query/search/feedback/browse, preferences/storage, migration/source and panel lifecycle/routes/domain handlers. Tests follow domain names; shared fixtures live in test_helpers_test.go.
- internal/workflow/panel: embedded HTML/CSS and native ES modules (app, api, dom, records, settings, migration); no frontend build step or runtime dependencies.
- public/assets: shipped result icons; public contains the workflow template and branding.
- public/info.plist: generated Alfred wiring; edit scripts/generate_workflow.py and regenerate it.
- scripts/build_workflow.py: CGO_ENABLED=0 builds, universal Mach-O creation, workflow packaging. Python is build-only.
- scripts/migrate-v2.sh: native-backed preview/apply migration; source scope filtering, original backups, per-record retry markers. See docs/MIGRATION.zh-CN.md.
- scripts/acceptance.py: real compiled CLI acceptance, SQLite WAL, real Git/worktrees, subprocess IDE adapters and optional benchmark.
- scripts/verify_package.py: package and Mach-O structural checks.

Use Go 1.26+ and Python 3.9+. Run `go test -race ./...`, `go vet ./...`, `go mod verify`; build Linux with `python3 scripts/build_workflow.py --target linux`, then run `python3 scripts/acceptance.py`. Build all targets and verify packages before distributing. Format Go changes with gofmt. Keep module checksums enabled. No npm/pip install, credentials or persistent daemon is needed to build the workflow. Optional source formatters are Ruff 0.11.13 and Prettier 3.5.3; see README for commands. Keep blank lines between Go functions. The embedded management panel runs a loopback-only server on demand; queries never depend on it. Run scripts/panel_acceptance.py against the compiled host binary for panel changes.

Preserve these contracts: only VS Code history plus configured Git roots; no file history; remote entries never probe branches or remote hosts; local visible branches read afresh every query; no synchronous recursive scan in normal query; never write the VS Code DB. Preference writes must retain file locking and atomic replacement. Open editors via explicit argument arrays, never shell-evaluate user paths. Return valid Alfred JSON even for query errors.

Linux tests cannot validate Alfred UI, actual macOS app launches or live remote connections. Report these as pending macOS acceptance, never as passed. Use dev workflow bundle/keywords/data isolation for manual testing. Build outputs are ignored. Do not commit caches, fixtures or benchmark machine paths as runtime configuration.

Release identity is com.harvey.alfredapp.vsc.v3; development is com.harvey.alfredapp.vsc.v3.dev. Never reuse the V2 bundle ID or ship user state/development data overrides in a release package. Release defaults to Cmd+Shift+V connected to SEARCH; development omits this hotkey. Existing V3 installs update in place without another migration.
