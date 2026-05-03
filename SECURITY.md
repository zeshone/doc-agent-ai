# Security Policy

## Reporting a Vulnerability

If you discover a security vulnerability in doc-agent-ai, please report it privately.

**Do not open a public issue.**

Email: [zeshone@proton.me](mailto:zeshone@proton.me)

You will receive a response within 48 hours. We take all reports seriously and will work with you to verify and address the issue.

## Supported Versions

| Version | Supported |
|---------|-----------|
| 2.x     | ✅ Active |
| < 2.0   | ❌ End of life |

## Scope

doc-agent-ai is an installer and generator script. The attack surface is minimal — no network services, no user data storage, no dependency tree beyond Node.js standard library.

Areas in scope:
- Code injection via generated artifacts
- Path traversal in the installer
- Supply chain risks in the repository itself
