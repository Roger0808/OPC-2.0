#!/usr/bin/env python3
from pathlib import Path
import shutil
root=Path(__file__).resolve().parent.parent
source=root/'frontend/dist'
if not (source/'index.html').exists(): raise SystemExit('Build opc-client/frontend first')
dest=root/'desktop/assets'
for p in dest.iterdir():
    if p.name=='placeholder.txt': continue
    if p.is_dir(): shutil.rmtree(p)
    else: p.unlink()
shutil.copytree(source,dest,dirs_exist_ok=True)
