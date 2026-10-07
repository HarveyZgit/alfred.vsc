# End-to-end acceptance

These tests exercise the compiled native binary and embedded panel. Build and
packaging tools live in `scripts/`; end-to-end test fixtures and browser automation
belong here.

Run from the repository root after building the Linux binary:

```sh
python3 tests/e2e/acceptance.py
python3 tests/e2e/panel_acceptance.py
CHROMIUM_PATH=/path/to/chromium node tests/e2e/panel_browser_acceptance.cjs build/vsc-linux-amd64
```

The browser test requires development-only Playwright and Chromium. See
[validation instructions](../../docs/VALIDATION.zh-CN.md) for dependency setup,
reports and platform limitations.
