#!/usr/bin/env python3
"""Generate a mapped binary patch for review; never apply or overwrite client files."""
from pathlib import Path
import argparse, json, re, subprocess

root = Path(__file__).resolve().parent.parent
repo = root.parent
parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument('ref', help='Existing native-client commit or branch to synchronize from')
args = parser.parse_args()
baseline = json.loads((root / 'upstream.json').read_text())
target = subprocess.check_output(['git', 'rev-parse', '--verify', args.ref + '^{commit}'], cwd=repo, text=True).strip()
patch = subprocess.check_output(['git','diff','--binary',baseline['commit'],target,'--','cmd/desktop','frontend'],cwd=repo)
# Rewrite only path metadata; don't alter strings inside changed source lines.
lines=[]
for line in patch.splitlines(keepends=True):
    if line.startswith((b'diff --git ', b'--- ', b'+++ ', b'Binary files ', b'rename from ', b'rename to ', b'copy from ', b'copy to ')):
        for old,new in [(b'cmd/desktop/',b'opc-client/desktop/'),(b'frontend/',b'opc-client/frontend/')]:
            # Each token is mapped only once; generated opc-client paths are never remapped.
            line=re.sub(rb'(?<!opc-client/)' + re.escape(old),new,line)
    lines.append(line)
output=root/'sync-patches'/f"{baseline['commit'][:9]}-{target[:9]}.patch"
output.parent.mkdir(exist_ok=True)
output.write_bytes(b''.join(lines))
print(output)
print('Review this patch, then use git apply --check before applying. Conflicts require manual reconciliation.')
print('After reconciliation, update upstream.json hashes from the chosen native commit; do not simply replace the fork.')
