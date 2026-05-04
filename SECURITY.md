# Security Policy

## Reporting a Vulnerability

If you discover a security vulnerability in doc-agent-ai, please open a public issue.

We review all issues promptly and take security reports seriously.

## Supported Versions

| Version | Supported |
|---------|-----------|
| 3.x     | ✅ Active |
| 2.x     | ✅ Critical fixes only |
| < 2.0   | ❌ End of life |

## Scope

doc-agent-ai is an installer and generator distributed as a single Go binary. The attack surface is minimal — no network services, no user data storage, no runtime dependencies.

Areas in scope:
- Code injection via generated artifacts
- Path traversal in the installer or uninstaller
- Supply chain risks in the repository or release artifacts
