# Security Policy

The Matrix (`neo`, `morp`, `oracle`, `trinity`, and `matrixapp`) provides local developer tooling and scaffolding for autonomous AI coding agents. Because agent configurations and Oracle knowledge exports handle local source trees, developer prompts, and system prompts, security and confidentiality are foundational to the project.

## Reporting a Vulnerability

**Please do not report security vulnerabilities through public GitHub issues, discussions, or pull requests.**

To report a vulnerability:

1. **Preferred method:** Submit a private advisory through [GitHub Security Advisories](https://github.com/0merUfuk/the-matrix/security/advisories/new).
2. **Alternative method:** If GitHub Security Advisories is unavailable, contact the project maintainer privately via the security contact on [0merUfuk's GitHub profile](https://github.com/0merUfuk).

### What to Include

Please provide the minimum synthetic information necessary to reproduce the issue:
- Affected component (`neo`, `morp`, `oracle`, `trinity`, or `matrixapp`)
- The Matrix version or Git commit hash
- Operating system and Go runtime version
- Step-by-step reproduction instructions with synthetic fixtures (never include real credentials, private API keys, or proprietary codebase paths)
- Observed behavior vs. expected secure behavior
- Suggested remediation or patch, if available

### Response Timelines and SLA

- **Initial Acknowledgement:** Within 48 hours of report receipt.
- **Triage and Impact Assessment:** Within 5 business days.
- **Patch Development & Release:** Critical vulnerabilities are prioritized for resolution within 14 days on the `main` branch.
- **Coordinated Disclosure:** We adhere to responsible coordinated disclosure, typically publishing details 30 days after a fix has been released or in coordination with the reporter.

## Supported Versions

Security updates are provided exclusively for the latest release line on the `main` branch. Older tagged releases do not receive backported security patches; users are strongly encouraged to upgrade to the latest stable release.

| Version | Supported |
| --- | --- |
| `v1.9.x` / `main` | :white_check_mark: Yes |
| `< v1.9.0` | :x: No (Upgrade required) |

## Security Principles and Architecture

### 1. Localhost Isolation
The local dashboard server (`matrixapp`) binds strictly to loopback (`127.0.0.1`). It does not bind to wildcard (`0.0.0.0`) or LAN interfaces, preventing remote network access on local networks.

### 2. Zero Telemetry and Outbound Secrecy
The CLI tools execute entirely locally. They do not send telemetry, user analytics, execution traces, or source files to remote servers or third-party endpoints.

### 3. File System Integrity
Oracle exports (`oracle export --format antigravity`) write directly into the designated local directory (`.agents/rules/`). File operations enforce strict path boundary checks to prevent path traversal outside the targeted repository or rule directory.

### 4. Supply Chain and Vulnerability Management
- All Go dependencies are tracked and audited using `govulncheck` and static analysis (`gosec`, `golangci-lint`) in CI.
- Automated dependency scanning is managed via [Dependabot](https://github.com/0merUfuk/the-matrix/blob/main/.github/dependabot.yml).
- Workflows are monitored by [OpenSSF Scorecard](https://github.com/0merUfuk/the-matrix/blob/main/.github/workflows/scorecard.yml) with results published to the OpenSSF Scorecard public registry.
