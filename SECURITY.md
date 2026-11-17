# Security Policy

## Supported versions

Only the latest published v1 release of ghstats receives security fixes.

| Version | Supported |
| ------- | --------- |
| latest v1 | Yes |
| older | No |

## Reporting a vulnerability

**Do not open a public issue for security reports.**

Use GitHub's private vulnerability reporting instead:
**[Report a vulnerability](https://github.com/wewesss/ghstats/security/advisories/new)**

Include:

- A description of the vulnerability and its potential impact.
- Steps to reproduce or a minimal proof of concept.
- The ghstats version (or the `wewesss/ghstats@` ref), the runner OS and your configuration with private names removed.

I will acknowledge your report within **72 hours** and aim to release a fix within **14 days** depending on severity. I will credit you in the release notes unless you prefer to remain anonymous.

## Scope

In scope:

- **The token.** ghstats reads a personal access token and must never write it to a file, a log line, a snapshot or a rendered card.
- **The private data repository.** Snapshots and the traffic history contain private repository names and stay in a private repository; a way to leak them to a public one is a vulnerability.
- **Private repository names in the outputs.** `stats.json`, the SVG cards, the link badges and `README.snippet.md` never contain a private repository name.
- **The release download in the Action.** The Action downloads the release archive and verifies it against `checksums.txt` before it installs the binary; a way around that check is a vulnerability.
- **Injection into the generated SVG and markup.** Values from the configuration (labels, URLs, icon paths, base path) are escaped or refused.

Out of scope: GitHub itself, the runner, your workflow file, and the permissions you give your own token. Reports outside this scope should be directed to the relevant upstream project.
