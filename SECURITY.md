# Security Policy

## Supported Versions

| Version | Supported |
|---|---|
| 1.x.x | ✅ |
| < 1.0.0 | ❌ (pre-release) |

## Reporting a Vulnerability

**Do not open a public issue.** Send details to:

📧 info@rubenalvarez.dev

Include:
- Description of the vulnerability
- Steps to reproduce
- Affected version(s)
- Any suggested mitigation

Response within 48 hours. Disclosure coordinated after a fix is released.

## Security Design

- Jart-Stow runs entirely locally. No data leaves the machine.
- SQLite database is stored at `~/.local/share/jart-stow/` with file permissions `0600`.
- The daemon requires `sudo` only for `tmutil` operations; it does not run as root.
- No telemetry, no analytics, no network calls.
