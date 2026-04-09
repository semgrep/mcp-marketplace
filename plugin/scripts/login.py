#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.12"
# dependencies = [
#     "httpx>=0.28.1",
#     "pyyaml>=6.0.1",
# ]
# ///

# use `uv add --script login.py 'dep'` to add a dependency

"""
Standalone semgrep login script.

Opens a browser for the user to authenticate with semgrep.dev, polls for the
resulting token, validates it, and writes it to ~/.semgrep/settings.yml.
"""

import os
import re
import sys
import tempfile
import time
import uuid
import webbrowser
from pathlib import Path
from typing import Optional

import httpx
import yaml

SEMGREP_URL = os.environ.get("SEMGREP_URL", "https://semgrep.dev")
WAIT_BETWEEN_RETRY_SEC = 6
MAX_RETRIES = 30  # ~3 minutes


def get_settings_path() -> Path:
    xdg = os.environ.get("XDG_CONFIG_HOME")
    if xdg:
        return Path(xdg) / "semgrep" / "settings.yml"
    return Path.home() / ".semgrep" / "settings.yml"

def validate_token(token: str) -> bool:
    try:
        r = httpx.get(
            f"{SEMGREP_URL}/api/agent/deployments/current",
            headers={"Authorization": f"Bearer {token}"},
            timeout=10,
        )
        return r.is_success
    except Exception:
        return False


def read_existing_token(settings_path: Path) -> Optional[str]:
    if not settings_path.exists():
        return None
    try:
        data = yaml.safe_load(settings_path.read_text()) or {}
        token = data.get("api_token")
        if not validate_token(token):
            return None
        return str(token) if token else None
    except Exception:
        return None


def write_token(settings_path: Path, token: str) -> None:
    settings_path.parent.mkdir(parents=True, exist_ok=True)

    if settings_path.exists():
        try:
            data = yaml.safe_load(settings_path.read_text()) or {}
        except Exception:
            data = {}
    else:
        data = {}

    data["api_token"] = token

    fd, tmp = tempfile.mkstemp(dir=settings_path.parent, suffix=".yml", prefix="settings", text=True)
    try:
        with os.fdopen(fd, "w") as f:
            yaml.dump(data, f, default_flow_style=False)
        os.replace(tmp, settings_path)
    except Exception:
        os.unlink(tmp)
        raise


def main() -> None:
    settings_path = get_settings_path()

    existing = read_existing_token(settings_path)
    if existing:
        print(f"Already logged in. Token saved at {settings_path}.")
        print("Run `semgrep logout` first if you want to log in again.")
        sys.exit(0)

    session_id = uuid.uuid4()
    url = f"{SEMGREP_URL}/login?cli-token={session_id}"

    print("Opening browser to log in to semgrep.dev...")
    print(f"  {url}")
    webbrowser.open(url)
    print("\nWaiting for login... (you have ~3 minutes)\n")

    for attempt in range(MAX_RETRIES):
        try:
            r = httpx.post(
                f"{SEMGREP_URL}/api/agent/tokens/requests",
                json={"token_request_key": str(session_id)},
                timeout=10,
            )
        except httpx.RequestError as e:
            print(f"Semgrep login: Network error: {e}", file=sys.stderr)
            sys.exit(2)

        if r.status_code == 200:
            token = r.json().get("token")
            if not token:
                print("Semgrep login: Error: server returned 200 but no token in response.", file=sys.stderr)
                sys.exit(2)

            if len(token) != 64 or not re.match(r"^[0-9a-f]+$", token):
                print("Semgrep login: Error: received token has unexpected format.", file=sys.stderr)
                sys.exit(2)

            print("Token received. Validating...")
            if not validate_token(token):
                print("Semgrep login: Error: token validation failed.", file=sys.stderr)
                sys.exit(2)

            write_token(settings_path, token)
            print(f"Logged in. Token saved to {settings_path}.")
            sys.exit(0)

        elif r.status_code != 404:
            print(f"Semgrep login: Unexpected response from server: {r.status_code}", file=sys.stderr)
            sys.exit(2)

        # 404 = user hasn't completed browser login yet
        print(f"  Waiting... ({attempt + 1}/{MAX_RETRIES})", end="\r")
        time.sleep(WAIT_BETWEEN_RETRY_SEC)

    print("\nSemgrep login: Login timed out. Please try again.", file=sys.stderr)
    sys.exit(2)


if __name__ == "__main__":
    main()
