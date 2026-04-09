#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.12"
# dependencies = [
#     "httpx>=0.28.1",
#     "pyyaml>=6.0.1",
#     "pydantic>=2.10.6",
# ]
# ///

# use `uv add --script cli.py 'dep'` to add a dependency

# uncomment decorator in fragscan/endpoint.py to skip auth

import json
import os
import sys
import time
from typing import Literal

from pathlib import Path

from pydantic import BaseModel
import httpx
import yaml

class PostToolHookResponse(BaseModel):
    # response = {
    #     "decision": "block"|undefined,
    #     "reason": ...,
    #     "hookSpecificOutput": {
    #         "hookEventName": ...,
    #         "additionalContext": ...,
    #     }
    # }
    decision: Literal["block"] | None = None
    reason: str | None = None

# modified from cli/src/semgrep/mcp/hooks/post_tool.py in semgrep-proprietary
def load_file_path_claude() -> str:
    hook_data = json.load(sys.stdin)
    return hook_data["tool_input"]["file_path"]

# coupling: cli/src/semgrep/env.py in semgrep-proprietary
def user_data_folder_default() -> Path:
        config_home = os.getenv("XDG_CONFIG_HOME")
        if config_home is None or not Path(config_home).is_dir():
            parent_dir = Path.home()
        else:
            parent_dir = Path(config_home)
        return parent_dir / ".semgrep"

def user_settings_file_default() -> Path:
        path = os.getenv(
            "SEMGREP_SETTINGS_FILE", str(user_data_folder_default() / "settings.yml")
        )
        return Path(path)

def get_app_token_from_settings() -> str:
        settings_file = user_settings_file_default()
        # print(f"settings_file: {settings_file}")
        if not settings_file.exists():
            return None
        with open(settings_file, 'r') as fd:
            settings = yaml.safe_load(fd)
            app_token = settings.get("api_token")
            if app_token:
                return app_token
            else:
                return None

# copied from fragment scanner repo:
class SemgrepAppToken(httpx.Auth):
    def __init__(self, token):
        self.token = token

    def auth_flow(self, request):
        response = yield request
        if response.status_code == 401:
            request.headers["Authorization"] = f"Bearer {self.token}"
            yield request


def load_files(base, filenames):
    paths = []
    for name in filenames:
        p = Path(name).resolve()
        if p.is_dir():
            for root, dirs, files in p.walk():
                if any(n.startswith(".") for n in root.parts):
                    continue
                for name in files:
                    if name.startswith("."):
                        continue
                    fp = (root / name).relative_to(base)
                    paths.append(fp)
        else:
            p = p.relative_to(base)
            paths.append(p)

    files = {}
    for p in paths:
        try:
            data = p.read_text()
            files[str(p)] = data
            # print("scan: including", p)
        except UnicodeDecodeError:
            pass
    return files


def request_scan(url, args, auth=None, log=print):
    result = None
    while result is None:
        try:
            result = httpx.post(url, json=args, auth=auth, timeout=(5, 60 * 5))
        except httpx.RequestError as exc:
            log(exc)
            log("connection error")
            time.sleep(0.5)
            continue

        if 200 <= result.status_code < 300:
            break
        elif result.status_code == 401:
            break
        else:
            log(result.text)
            log("service error")
            time.sleep(0.5)
            continue
    return result


if __name__ == "__main__":
    local_url = "http://127.0.0.1:8000/api/run"
    # remote_url = f"https://mcp-dev.semgrep.ai/fragment"
    env_url = os.environ.get("SEMGREP_FRAGMENT_URL", None)

    url = env_url or local_url
    scan_rule = None

    config = {}
    files = []
    trace = None
    args = list(sys.argv[1:])

    subcommand = args.pop(0)
    if subcommand != "scan":
        print(f"error: use {sys.argv[0]} scan ...")
        sys.exit(-1)

    while args:
        arg = args.pop(0)
        if arg.startswith("--"):
            arg = arg[2:]
            if arg == "config":
                arg = args.pop(0)
                config["rule"] = Path(arg).read_text()
            else:
                config[arg] = True
        else:
            files.append(arg)

    file_path = load_file_path_claude()
    files.append(file_path)

    if trace:  # check env
        trace = {
            "level": "...",
            "span_id": "...",
            "trace_id": "....",
            "endpoint": "....",
        }

    app_token = os.environ.get("SEMGREP_APP_TOKEN", "")
    auth = None
    # check env for app token, then check settings file
    if not app_token:
        app_token = get_app_token_from_settings()
    
    # print(f"app_token: {app_token}")

    if app_token:
        config["app_token"] = app_token
        auth = SemgrepAppToken(app_token)
    
    if not app_token:
        response = PostToolHookResponse(decision="block", reason="No app token found. You might have to restart your Claude session and activate your Semgrep session in your browser. You should not have to run `semgrep login` manually, a browser window will open at the beginning of the Claude session.")
        print(response.model_dump_json())
        sys.exit(0) # exit 0 here to show json response to user

    scan_files = load_files(Path.cwd(), files)

    scan_args = {
        "name": "scan",
        "files": scan_files,
        "config": config,
        "trace": trace,
    }

    run_args = {"command": scan_args}

    response = request_scan(url, run_args, auth=auth).json()
    result = response.pop("result", None)

    if result and result["json"]:
        findings = result["json"]["results"]
        if findings and len(findings) > 0:
            reason = str(
            [
                {
                    "line": r["start"]["line"],
                    "display_name": r["extra"]["metadata"].get("display-name"),
                    "message": r["extra"]["message"],
                    "severity": r["extra"]["severity"],
                    "cwe": r["extra"]["metadata"].get("cwe"),
                }
                for r in findings
            ]
            )
            response = PostToolHookResponse(decision="block", reason=reason)
            print(response.model_dump_json())
        else:
            response = PostToolHookResponse(decision="allow", reason="No findings")
            print(response.model_dump_json())
    else:
        response = PostToolHookResponse(decision="allow", reason="No results")
        print(response.model_dump_json())

    # err = response.pop("error", None)
    # print("-- response")
    # print(json.dumps(response, indent=4))

    # if err:
    #     print("error:", err)
    # elif result:
    #     if result["json"]:
    #         print("--- JSON:")
    #         print(json.dumps(result["json"], indent=4))
    #     elif result["stdout"]:
    #         print("--- STDOUT:")
    #         print(result["stdout"])
    #     if result["stderr"]:
    #         print("--- STDERR:", file=sys.stderr)
    #         print(result["stderr"], file=sys.stderr)
    #     if result["code"] != 0:
    #         print("error: non zero exit", result["code"])
    #     else:
    #         print("ok: process exited without error")

    #     print(result["cmd"])
    # else:
    #     print("bad response", result)
