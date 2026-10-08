# Security configuration

The reference is [OSSAfrica/skillguard](https://github.com/OSSAfrica/skillguard).
BurpBridge retains its stricter DCO/GPG, review, and merge requirements.

## Repository settings

Secret scanning, secret push protection, Dependabot alerts and security updates,
private vulnerability reporting, and web commit sign-offs are enabled.
Actions use a read-only token by default and cannot approve pull requests.
The default branch requires `Go checks`, `DCO and GPG`, signed commits,
an up-to-date branch, one approval, last-push approval, and resolved discussions.
Only merge commits are allowed; there are no ruleset bypass actors.

## Workflows

- `CodeQL` analyzes Go, Python, and GitHub Actions on pull requests, main pushes,
  and weekly. Fork code is never executed by a privileged target workflow.
- `Scorecard supply-chain security` publishes weekly/main-branch supply-chain
  findings to the code scanning dashboard and as a retained SARIF artifact.
- `Dependency SBOM` scans source on main, uploads an SPDX inventory, and submits
  dependencies to GitHub. It does not need a Docker image.
- Dependabot opens weekly Go module and Actions updates. Bot commits still need
  the same DCO/GPG requirements; maintainers may need signed replacements.

All external actions are pinned to full commit hashes. Scorecard and SBOM jobs
need the explicitly declared publication permissions; ordinary CI remains read-only.
The scheduled/default-branch workflows start after this configuration is merged.

## Code scanning merge protection

The active ruleset now requires CodeQL results, security alerts
`high_or_higher`, and ordinary alerts `errors_and_warnings`, matching SkillGuard.
Successful analysis of this pull request was verified before enabling the rule.
Keep the existing rules and bypass policy intact when updating the ruleset.
Older disposable setup helpers may overwrite later manual rule additions; review
the entire resulting ruleset before applying them again.

## Organization configuration

Organization policy APIs require organization-owner access and the CLI token's
`admin:org` scope. Repository administration alone is insufficient.
Compare organization Actions policies, security configurations/defaults, web
commit sign-off policy, and rulesets separately. Preserve BurpBridge's narrower
read access instead of copying OSSAfrica's default write access. Changes that
remove member access, including enforcing two-factor authentication, need a
membership impact review before rollout.

GitHub Code Quality has been enabled separately for Go and Python, with AI
findings on pushes, matching SkillGuard. Its initial setup scan must succeed
before adding the corresponding errors-only merge rule. CLI/container release automation from
SkillGuard is omitted because this repository is a shared Go engine.
