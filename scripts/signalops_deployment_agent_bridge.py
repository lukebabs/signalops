#!/usr/bin/env python3
"""Narrow Unix-socket bridge from the gateway to the deployment agent."""

from __future__ import annotations

import http.server
import json
import os
import re
import socketserver
import subprocess
from pathlib import Path

SOCKET_PATH = Path(os.environ.get("SIGNALOPS_DEPLOYMENT_AGENT_SOCKET", "/run/signalops/deployment-agent.sock"))
AGENT_BIN = os.environ.get("SIGNALOPS_DEPLOYMENT_AGENT_BIN", "/usr/local/sbin/signalops-deploy-agent")
ALLOWED_JOB_IDS = {
    "marketops-daily-postclose",
    "marketops-sri-refresh",
    "marketops-sri-holdings-refresh",
    "marketops-intraday",
    "marketops-fmp-continuation",
    "marketops-fmp-annual-financial",
    "marketops-task-retry",
    "marketops-postclose-recovery",
    "marketops-risk-reward",
    "marketops-operations-monitor",
    "signalops-storage-monitor",
    "signalops-retention-governance",
}
ACTION_PATTERN = re.compile(r"^scheduler-run-now:([a-z0-9-]+)$")


def write_json(handler: http.server.BaseHTTPRequestHandler, status: int, body: dict[str, object]) -> None:
    raw = json.dumps(body, separators=(",", ":")).encode()
    handler.send_response(status)
    handler.send_header("Content-Type", "application/json")
    handler.send_header("Content-Length", str(len(raw)))
    handler.end_headers()
    handler.wfile.write(raw)


class Handler(http.server.BaseHTTPRequestHandler):
    server_version = "SignalOpsDeploymentAgentBridge/1.0"

    def log_message(self, _format: str, *_args: object) -> None:
        return

    def do_POST(self) -> None:  # noqa: N802
        if self.path != "/run-now":
            write_json(self, 404, {"status": "rejected", "output": "unknown endpoint"})
            return
        try:
            length = min(int(self.headers.get("Content-Length", "0")), 4096)
            payload = json.loads(self.rfile.read(length) or b"{}")
        except Exception:
            write_json(self, 400, {"status": "rejected", "output": "invalid json"})
            return
        action = str(payload.get("action", "")).strip()
        match = ACTION_PATTERN.fullmatch(action)
        if not match or match.group(1) not in ALLOWED_JOB_IDS:
            write_json(self, 403, {"status": "rejected", "output": "unsupported action"})
            return
        try:
            completed = subprocess.run([AGENT_BIN, action], capture_output=True, text=True, timeout=45, check=False)
        except Exception as exc:
            write_json(self, 503, {"status": "failed", "output": str(exc)})
            return
        output = "\n".join(part.strip() for part in (completed.stdout, completed.stderr) if part.strip())
        if completed.returncode == 0:
            write_json(self, 202, {"status": "accepted", "output": output})
            return
        write_json(self, 503, {"status": "failed", "output": output or f"deployment agent exited {completed.returncode}"})


class Server(socketserver.UnixStreamServer):
    allow_reuse_address = True


def main() -> None:
    SOCKET_PATH.parent.mkdir(mode=0o750, parents=True, exist_ok=True)
    if SOCKET_PATH.exists():
        SOCKET_PATH.unlink()
    with Server(str(SOCKET_PATH), Handler) as server:
        SOCKET_PATH.chmod(0o600)
        server.serve_forever()


if __name__ == "__main__":
    main()
