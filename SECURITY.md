# Security Policy

## Disclaimer

This software is provided **"as is"**, without warranty of any kind. See [LICENSE](LICENSE) for full terms.

## Supported Versions

Only the latest release receives security fixes.

| Version | Supported |
|---------|-----------|
| latest  | Yes       |
| older   | No        |

## Reporting a Vulnerability

**Do not open a public GitHub issue for security vulnerabilities.**

Report security issues privately via [GitHub Security Advisories](https://github.com/braunmar/worktree/security/advisories/new).

Include:
- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Suggested fix (optional)

You can expect an acknowledgment within 7 days. If the issue is confirmed, a fix will be released as soon as reasonably possible.

## Scope

This tool runs locally and manages git worktrees and Docker instances. Key areas of concern:

- Command injection via `.worktree.yml` configuration values
- Path traversal in worktree directory operations
- Unintended file overwrites during worktree creation

## Out of Scope

- Issues in third-party dependencies (report to the respective projects)
- Issues requiring physical access to the machine
