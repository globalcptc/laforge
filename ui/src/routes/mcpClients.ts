// How to connect each AI client to LaForge's MCP server, shown in the new-key
// dialog. Steps may use `backticks` for inline code. Formats checked against
// each client's docs, 2026-11.

export interface McpSnippet {
  label: string
  text: string
}

export interface McpClient {
  id: string
  name: string
  // A caveat shown above the steps (a workaround, or why it can't connect).
  note?: string
  steps: string[]
  snippets: McpSnippet[]
}

const json = (v: unknown) => JSON.stringify(v, null, 2)

export function mcpClients(url: string, token: string): McpClient[] {
  const bearer = `Bearer ${token}`
  return [
    {
      id: 'claude-code',
      name: 'Claude Code',
      steps: [
        'Run the command in a terminal. Add `--scope user` to use LaForge in every project, not just this folder.',
        'Or, to share the server setup with a project, put the config in `.mcp.json` at its root (keep the key out of git).',
        'Run `/mcp` in Claude Code to check `laforge` is connected.',
      ],
      snippets: [
        { label: 'Command', text: `claude mcp add --transport http laforge ${url} --header "Authorization: ${bearer}"` },
        { label: '.mcp.json', text: json({ mcpServers: { laforge: { type: 'http', url, headers: { Authorization: bearer } } } }) },
      ],
    },
    {
      id: 'claude-desktop',
      name: 'Claude Desktop',
      note: "Claude Desktop's Connectors only accept servers with OAuth sign-in, so this goes through the mcp-remote bridge, which needs Node.js installed.",
      steps: [
        'Open Settings → Developer → Edit Config. That opens `claude_desktop_config.json`.',
        'Add `laforge` under `mcpServers` (merge it in if the file already has servers).',
        'Fully quit Claude Desktop and reopen it. LaForge shows up in the tools menu.',
      ],
      snippets: [
        {
          label: 'claude_desktop_config.json',
          text: json({
            mcpServers: {
              laforge: {
                command: 'npx',
                args: ['-y', 'mcp-remote', url, '--header', 'Authorization:${LAFORGE_AUTH}'],
                env: { LAFORGE_AUTH: bearer },
              },
            },
          }),
        },
      ],
    },
    {
      id: 'cursor',
      name: 'Cursor',
      steps: [
        'Open `~/.cursor/mcp.json` for every project, or `.cursor/mcp.json` in one project.',
        'Add `laforge` under `mcpServers`.',
        'Check Settings → MCP shows `laforge` with a green dot.',
      ],
      snippets: [{ label: 'mcp.json', text: json({ mcpServers: { laforge: { url, headers: { Authorization: bearer } } } }) }],
    },
    {
      id: 'vscode',
      name: 'VS Code (Copilot)',
      steps: [
        'Run `MCP: Open User Configuration` from the Command Palette, or create `.vscode/mcp.json` in a project.',
        'Add `laforge` under `servers`.',
        'Start it from the file (the Start link above the server) and use it in Copilot Chat in Agent mode.',
      ],
      snippets: [{ label: 'mcp.json', text: json({ servers: { laforge: { type: 'http', url, headers: { Authorization: bearer } } } }) }],
    },
    {
      id: 'windsurf',
      name: 'Windsurf',
      note: 'On Teams and Enterprise plans an admin has to allow remote MCP servers first.',
      steps: [
        'Open `~/.codeium/windsurf/mcp_config.json` (Cascade → MCP → Manage → View raw config).',
        'Add `laforge` under `mcpServers`. Windsurf uses `serverUrl`, not `url`.',
        'Refresh the MCP list in Cascade.',
      ],
      snippets: [{ label: 'mcp_config.json', text: json({ mcpServers: { laforge: { serverUrl: url, headers: { Authorization: bearer } } } }) }],
    },
    {
      id: 'gemini',
      name: 'Gemini CLI',
      steps: [
        'Open `~/.gemini/settings.json` for every project, or `.gemini/settings.json` in one project.',
        'Add `laforge` under `mcpServers`.',
        'Restart Gemini CLI and run `/mcp` to check it connected.',
      ],
      snippets: [{ label: 'settings.json', text: json({ mcpServers: { laforge: { httpUrl: url, headers: { Authorization: bearer } } } }) }],
    },
    {
      id: 'codex',
      name: 'OpenAI Codex',
      note: 'Codex reads the key from an environment variable rather than its config file. The variable has to be set before Codex starts.',
      steps: [
        'Add the export to your shell profile (`~/.zshrc`, `~/.bashrc`) and open a new terminal.',
        'Add the server to `~/.codex/config.toml`. The CLI and the IDE extension share this file.',
        'Run `codex mcp list` to check it.',
      ],
      snippets: [
        { label: 'Shell Profile', text: `export LAFORGE_API_KEY=${token}` },
        { label: 'config.toml', text: `[mcp_servers.laforge]\nurl = "${url}"\nbearer_token_env_var = "LAFORGE_API_KEY"` },
      ],
    },
    {
      id: 'openai-api',
      name: 'OpenAI API',
      note: "OpenAI's servers call LaForge directly, so the key is sent to OpenAI with every request. Prefer a short-lived key limited to the repositories you need.",
      steps: [
        'Add the tool to the `tools` list of a Responses API request.',
        'Send the headers on every request: OpenAI does not store them with the response.',
        'Keep `require_approval` on `always` if your app can power hosts.',
      ],
      snippets: [
        {
          label: 'Tool',
          text: json({ type: 'mcp', server_label: 'laforge', server_url: url, headers: { Authorization: bearer }, require_approval: 'always' }),
        },
      ],
    },
    {
      id: 'chatgpt',
      name: 'ChatGPT',
      note: "Not yet. ChatGPT supports custom MCP servers, but only with OAuth sign-in or no sign-in at all: it can't send an API key. LaForge's MCP server will work here once it supports OAuth. Until then, use another client.",
      steps: [
        'Once OAuth is supported: go to chatgpt.com/plugins, select +, then Add custom MCP server.',
        'Enter the server URL, choose OAuth, and sign in with your LaForge account.',
      ],
      snippets: [{ label: 'Server URL', text: url }],
    },
    {
      id: 'claude-ai',
      name: 'Claude.ai',
      note: "Not yet. Claude.ai's custom connectors also sign in with OAuth and can't send an API key. Claude Desktop works today through the bridge (pick Claude Desktop above).",
      steps: [],
      snippets: [],
    },
    {
      id: 'other',
      name: 'Other',
      steps: [
        'Any MCP client that supports the Streamable HTTP transport and custom headers works.',
        'Point it at the URL and send the key as a bearer token in the `Authorization` header.',
      ],
      snippets: [
        { label: 'Server URL', text: url },
        { label: 'Header', text: `Authorization: ${bearer}` },
      ],
    },
  ]
}
