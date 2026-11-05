"""初始化专用凭证目录。切换已有凭证时使用新目录，成功后替换授权清单。"""
import argparse
import hashlib
import json
import os
import secrets
from pathlib import Path

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument("directory", type=Path)
parser.add_argument("--actor", required=True)
parser.add_argument("--name", required=True)
parser.add_argument("--channel", required=True, action="append")
parser.add_argument("--write", action="store_true")
args = parser.parse_args()
os.umask(0o077)
args.directory.mkdir(parents=True, exist_ok=False)
grants = args.directory / "grants"
grants.mkdir()
token = secrets.token_urlsafe(48)
(args.directory / "token").write_text(token + "\n")
record = dict(name=args.name, actorID=args.actor, channels=args.channel,
              write=args.write, sha256=hashlib.sha256(token.encode()).hexdigest())
(grants / "grants.json").write_text(json.dumps([record]) + "\n")
