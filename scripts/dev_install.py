#!/usr/bin/env python3
"""Create an isolated development package; Alfred chooses its preferences path."""

import argparse
from pathlib import Path
import subprocess
import sys

ROOT = Path(__file__).resolve().parent.parent


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--go", default="go")
    parser.add_argument("--skip-build", action="store_true")
    parser.add_argument(
        "--open", action="store_true", help="Ask Alfred to import the package on macOS"
    )
    args = parser.parse_args()
    command = [
        sys.executable,
        str(ROOT / "scripts/build_workflow.py"),
        "--target",
        "macos",
        "--dev",
        "--go",
        args.go,
    ]
    if args.skip_build:
        command.append("--skip-build")
    subprocess.run(command, check=True)
    if args.open:
        if sys.platform != "darwin":
            raise RuntimeError("--open requires macOS")
        subprocess.run(["open", str(ROOT / "build/vsc-dev.alfredworkflow")], check=True)


if __name__ == "__main__":
    try:
        main()
    except (OSError, RuntimeError, subprocess.CalledProcessError) as error:
        print(error, file=sys.stderr)
        sys.exit(1)
