#!/usr/bin/env python3
"""Verify that this fork did not modify the tracked native sources it copied."""
from pathlib import Path
import hashlib, json, subprocess

root = Path(__file__).resolve().parent.parent
repo = root.parent
manifest = json.loads((root / 'upstream.json').read_text())
changed = []
for item in manifest['files']:
    path = repo / item['source']
    if not path.exists() or hashlib.sha256(path.read_bytes()).hexdigest() != item['sha256']:
        changed.append(item['source'])
if changed:
    print('Native sources differ from the recorded copy baseline:')
    print('\n'.join(changed))
    raise SystemExit(1)
print(f"Verified {len(manifest['files'])} original source files unchanged (baseline {manifest['commit'][:9]})")
