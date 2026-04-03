#!/usr/bin/env python3

from __future__ import annotations

import argparse
from pathlib import Path


def load_checksums(checksums_file: Path) -> dict[str, str]:
    checksums: dict[str, str] = {}

    for raw_line in checksums_file.read_text(encoding="utf-8").splitlines():
        line = raw_line.strip()
        if not line:
            continue

        sha256, name = line.split(None, 1)
        checksums[name.strip()] = sha256.strip()

    return checksums


def render_formula(template: str, version: str, checksums: dict[str, str]) -> str:
    required_assets = {
        "__DARWIN_ARM64_SHA256__": f"celador_{version}_darwin_arm64.tar.gz",
        "__LINUX_AMD64_SHA256__": f"celador_{version}_linux_amd64.tar.gz",
    }

    missing_assets = [asset for asset in required_assets.values() if asset not in checksums]
    if missing_assets:
        missing = ", ".join(missing_assets)
        raise SystemExit(f"missing release assets in checksums.txt: {missing}")

    rendered = template.replace("__VERSION__", version)
    for placeholder, asset_name in required_assets.items():
        rendered = rendered.replace(placeholder, checksums[asset_name])

    return rendered


def main() -> None:
    parser = argparse.ArgumentParser(description="Render the Celador Homebrew formula template.")
    parser.add_argument("--template", required=True)
    parser.add_argument("--version", required=True)
    parser.add_argument("--checksums-file", required=True)
    parser.add_argument("--output", required=True)
    args = parser.parse_args()

    template_path = Path(args.template)
    output_path = Path(args.output)
    checksums_file = Path(args.checksums_file)

    rendered = render_formula(
        template=template_path.read_text(encoding="utf-8"),
        version=args.version,
        checksums=load_checksums(checksums_file),
    )
    output_path.write_text(rendered, encoding="utf-8")


if __name__ == "__main__":
    main()
