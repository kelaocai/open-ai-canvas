"""Deploy images received over SSH. Run beside .env and compose.yaml with a commit SHA."""
import fcntl
import json
import os
from pathlib import Path
import re
import subprocess
import sys
import urllib.request

os.umask(0o077)
root = Path(__file__).resolve().parent
os.chdir(root)
lock = open("deploy.lock", "w")
try:
    fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
except BlockingIOError:
    raise SystemExit(0)
if Path("deployment-in-progress").exists():
    raise SystemExit("Previous deployment needs inspection; automatic retry paused")
if len(sys.argv) != 2:
    raise SystemExit("Usage: deploy-local.py COMMIT_SHA")
sha = sys.argv[1]
if not re.fullmatch("[0-9a-f]{40}", sha):
    raise ValueError("Invalid commit")
if Path("deployed-sha").exists() and Path("deployed-sha").read_text().strip() == sha:
    raise SystemExit(0)
docker = "/usr/local/bin/docker"
def run(*args, **kwargs):
    return subprocess.run(args, check=True, **kwargs)
images = {}
for variable, image in [("CANVAS_BACKEND_IMAGE", "backend"), ("CANVAS_WEB_IMAGE", "web"), ("CANVAS_AGENT_IMAGE", "yingce-agent")]:
    tag = f"rightmovie-{image}:sha-{sha}"
    info = json.loads(subprocess.check_output([docker, "image", "inspect", tag]))[0]
    if info["Architecture"] != "arm64":
        raise ValueError("Mac mini requires ARM64 images")
    images[variable] = tag
candidate = Path("candidate.env")
candidate.write_text("".join(k + "=" + v + "\n" for k, v in images.items()))
compose = [docker, "compose", "--env-file", ".env", "--env-file", str(candidate), "-f", "compose.yaml"]
run(*compose, "config", "--quiet")
Path("deployment-in-progress").write_text(sha + "\n")
# Stop waits for graceful worker drain before taking a consistent database backup.
run(*compose, "stop", "web", "backend")
backup = Path("backups") / (sha + ".dump")
backup.parent.mkdir(exist_ok=True)
with backup.open("wb") as out:
    run(docker, "exec", "-u", "postgres", "postgres-local", "pg_dump", "-Fc", "rightmovie", stdout=out)
run("/usr/bin/tar", "-czf", str(backup.with_suffix(".data.tgz")), "-C", "../data", ".")
if Path("release.env").exists():
    Path("previous-release.env").write_bytes(Path("release.env").read_bytes())
# Keep the failed candidate and backup for recovery. Schema rollback needs an explicit restore.
run(*compose, "run", "--rm", "migrate")
run(*compose, "up", "-d", "--no-build", "--wait", "--wait-timeout", "180", "redis", "yingce-agent", "backend", "web")
with urllib.request.urlopen("http://127.0.0.1:18880/api/health/ready", timeout=15) as response:
    health = json.load(response)["data"]
if not health["ready"] or health["build"]["commit"] != sha:
    raise RuntimeError("Release health verification failed")
candidate.replace("release.env")
Path("deployed-sha").write_text(sha + "\n")
Path("deployment-in-progress").unlink()
print("Deployed", sha, flush=True)
