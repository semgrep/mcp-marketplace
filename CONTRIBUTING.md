## Local development
- If you are in the directory of this repo, you can add the marketplace to your Claude Code instance by simply running
```
/plugin marketplace add ./
```

## Versioning
- Update the plugin version (in `plugin/.claude-plugin/plugin.json`) whenever the plugin changes
- Update `semgrep-version` if the change made to the plugin requires a newer version of `semgrep`
- `main` should always work on all versions of `semgrep` greater than the version stored in `semgrep-version`

## Testing
- In addition to the tests in CI, we should manually test that the plugin still works in Claude Code