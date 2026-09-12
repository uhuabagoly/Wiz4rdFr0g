"""Actions baseline and five-element selection; never executes installers."""
import json
import os
from pathlib import Path
import subprocess
import time

out = Path("release/actions")
out.mkdir(parents=True, exist_ok=True)
records = []

def run(args, extra_env=None, required=True):
    started = time.time()
    proc = subprocess.run(args, env=dict(os.environ, **(extra_env or {})), text=True,
                          stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
    records.append(dict(command=args, environment=extra_env or {}, exit_code=proc.returncode,
                        output=proc.stdout, duration_seconds=time.time()-started))
    (out / "baseline.json").write_text(json.dumps(records, indent=2), encoding="utf-8")
    print(proc.stdout)
    if required and proc.returncode:
        raise SystemExit(proc.returncode)
    return proc.returncode

if os.environ.get("GITHUB_ACTIONS") != "true":
    raise SystemExit("REMOTE_EXECUTION_REQUIRED: run through GitHub Actions")

run(["go", "test", "./..."])
run(["go", "vet", "./..."])
run(["go", "vet", "./app", "./installer"], {"GOOS": "windows", "GOARCH": "amd64", "CGO_ENABLED": "0"})
run(["go", "run", "./cmd/release-build", "release/build_manifest.json"])
run(["go", "run", "./cmd/campaign-plan", "test/windows-vm"])
run(["go", "build", "-o", "release/actions/release-gate", "./cmd/release-gate"])
empty = out / "empty-results"
empty.mkdir(exist_ok=True)
rc = run(["release/actions/release-gate", str(empty), "release/actions/baseline-gate", "release/build_manifest.json"], required=False)
gate = json.loads((out / "baseline-gate/release_gate.json").read_text())
if rc == 0 or gate["release_ready"] or not gate["missing_physical_evidence"]:
    raise SystemExit("Negative control failed: empty evidence must block the release gate")

plan = json.loads(Path("test/windows-vm/physical_test_plan.json").read_text())
entries = plan["entries"]
if len(entries) != plan["catalog_total"] or sorted(e["index"] for e in entries) != list(range(len(entries))):
    raise SystemExit("Invalid catalog coverage")
selected = []
for name in ["Audacity", "VLC Media Player", "Krita", "Blender", "KeePass 2"]:
    matches = [e for e in entries if e["name"] == name]
    if len(matches) != 1 or matches[0]["execution_disposition"] != "PHYSICAL_REQUIRED" or not matches[0].get("winget_id"):
        raise SystemExit(f"Pilot entry missing, ambiguous or not eligible: {name}")
    selected.append(matches[0]["index"])
(out / "pilot-indexes.json").write_text(json.dumps(selected), encoding="utf-8")
with open(os.environ["GITHUB_OUTPUT"], "a", encoding="utf-8") as output:
    output.write("matrix=" + json.dumps(selected) + "\n")
