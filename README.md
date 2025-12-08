# Semgrep MCP Marketplace

This repo is where the Semgrep [Plugin Marketplace](https://code.claude.com/docs/en/plugin-marketplaces) (`semgrep`) and the Semgrep [Plugin](https://code.claude.com/docs/en/plugins) (`semgrep-plugin@semgrep`) live.

To use the Semgrep plugin:
1. Start a Claude Code instance by running:
    ```
    claude
    ```
1. Add the Semgrep marketplace by running the following command in Claude:
    ```
    /plugin marketplace add semgrep/mcp-marketplace
    ```
1. Install the plugin from the marketplace:
    ```
    /plugin install semgrep-plugin@semgrep
    ```
1. If it is installed, see if you can run the `/semgrep-plugin:setup_semgrep_plugin` command. If you cannot run the command, try enabling the plugin:
    ```
    /plugin enable semgrep-plugin@semgrep
    ```