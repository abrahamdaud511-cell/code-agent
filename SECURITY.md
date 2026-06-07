# Security Policy

## Supported Versions

| Version | Supported          |
|---------|--------------------|
| 1.x     | :white_check_mark: |
| < 1.0   | :x:                |

## Reporting a Vulnerability

CodeAgent takes security seriously. If you discover a security vulnerability,
please report it by emailing security@codeagent.ai.

Please do NOT create a public GitHub issue for security vulnerabilities.

## Security Features

- **Local API Key Storage**: Keys stored in `~/.local/share/codeagent/auth.json` with 0600 permissions
- **Server Authentication**: HTTP Basic Auth via `CODEAGENT_SERVER_PASSWORD`
- **Permission System**: Granular tool-level access control (allow/deny/ask)
- **No Data Collection**: CodeAgent does not collect usage data or code context
- **Git Integration**: Changes are Git-tracked for full audit trail
- **Secure Defaults**: Destructive operations require explicit approval
