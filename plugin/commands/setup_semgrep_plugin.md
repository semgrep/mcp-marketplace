---
description: setup Claude Code for the Semgrep plugin (semgrep-plugin)
---

You are setting up Semgrep for Claude Code. Do the following in order and confirm each step:

1) Install Semgrep:
- Check if Semgrep is already installed by running: `which semgrep`
- If not installed, run: `brew install semgrep`

2) Authenticate Semgrep:
- Run: `semgrep login --force`

3) Install Semgrep Pro:
- Run: `semgrep install-semgrep-pro || true`

4) Report back:
- Confirm Semgrep login/install status by running `semgrep --pro --version`
- Also check the Semgrep version in `${CLAUDE_PLUGIN_ROOT}/semgrep-version` is lower
than or equal to the version installed.

5) Tell the user that they are all set for using the Semgrep Plugin!
