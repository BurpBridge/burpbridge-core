# Contributing to BurpBridge

First-time contributors are welcome. Start with a [good first issue](https://github.com/BurpBridge/burpbridge-core/issues?q=is%3Aissue%20is%3Aopen%20label%3A%22good%20first%20issue%22), comment that you would like to work on it, and ask for guidance if the scope is unclear. Documentation contributions are useful too.

## Development setup

Fork the repository, clone your fork, and create a topic branch. Install the Go version specified in `go.mod` (currently at least 1.25.5). Android NDK and Xcode are only needed for mobile library builds, not documentation or ordinary Go checks. See [DEVELOPMENT.md](DEVELOPMENT.md) for platform setup.

```sh
git switch -c docs/my-first-contribution
go mod download
```

This repository contains the shared engine and a desktop harness. Native VPN applications are separate integrations. iOS packetFlow support is pending, UDP is not forwarded, and TCP interception currently accepts only ports 80 and 443.

## Every commit needs a DCO sign-off and a GPG signature

By adding a `Signed-off-by` trailer, you certify the [Developer Certificate of Origin, version 1.1](https://developercertificate.org/). Read it before signing. The sign-off must match the commit author's name and email. Each person listed in a `Co-authored-by` trailer must also provide a matching sign-off; obtain their certification rather than signing on their behalf. There are no automatic bot exemptions.

A DCO sign-off (`-s`) and a cryptographic GPG signature (`-S`) serve different purposes. Both are required on every commit introduced by your PR, including merge commits. GitHub must report each signature as verified and valid. SSH and S/MIME signatures do not satisfy this project's GPG policy.

### Set up your identity and GPG key

Use your own name and an email verified on your GitHub account. A verified GitHub noreply email can be used if it is also included in your GPG key.

```sh
git config user.name "Your Name"
git config user.email "your-verified-email@example.com"
gpg --full-generate-key
gpg --list-secret-keys --keyid-format LONG
gpg --armor --export YOUR_KEY_ID
```

Add the exported **public** key to GitHub under Settings → SSH and GPG keys → New GPG key. Keep your private key private. Configure this clone:

```sh
git config gpg.format openpgp
git config user.signingkey YOUR_KEY_ID
git config commit.gpgsign true
git config tag.gpgsign true
export GPG_TTY=$(tty)
git commit -s -S -m "docs: clarify desktop setup"
git log -1 --show-signature
```

The resulting commit message includes `Signed-off-by: Your Name <your-verified-email@example.com>`. After pushing, check the commit's Verified badge on GitHub. See GitHub's [GPG setup guide](https://docs.github.com/en/authentication/managing-commit-signature-verification/generating-a-new-gpg-key) and [commit signing guide](https://docs.github.com/en/authentication/managing-commit-signature-verification/signing-commits).

### Repair your own topic-branch commits

For the latest commit, use `git commit --amend --no-edit -s -S`. For multiple commits you authored, use an interactive rebase against your branch's merge base, edit each affected commit, and amend it with `-s -S`. Rewriting a signed commit invalidates the previous signature, so sign it again. Do not add your own sign-off in place of another author's certification. Ask a maintainer for help with shared branches or merge commits.

If your already-pushed topic branch was rewritten, push that branch with `git push --force-with-lease`. Never force-push the default branch. Changing the PR description or posting a DCO comment does not repair commit trailers.

## Before opening a pull request

Use a focused change and a descriptive title, preferably `fix: ...`, `docs: ...`, or `test: ...`. Explain the problem, resulting behavior, related issue, and validation. Add regression tests for behavior changes; documentation-only fixes do not need artificial tests.

```sh
git ls-files -z '*.go' | xargs -0 gofmt -w
go build ./...
go vet ./...
go test -race ./...
python3 -m unittest discover -s scripts/ci -p 'test_*.py'
```

Keep PRs below 251 commits: GitHub's PR-commit endpoint returns at most 250, and the policy check fails closed rather than skipping commits. Avoid generated binaries, IDE settings, credentials, and unrelated formatting changes. Redact sensitive network details from logs.

## Review and merge policy

The intended default-branch rules require `Go checks` and `DCO and GPG`, an up-to-date branch, at least one maintainer approval, resolved review conversations, and verified signed commits. New changes dismiss stale approvals and require approval from someone other than the last pusher. Direct pushes, force pushes, and branch deletion are blocked; no bypass actors are configured.

Maintainers should approve after checks pass and all feedback is resolved. GitHub enforces these requirements at **merge time**; it does not prevent someone from clicking Approve while checks are pending. The rules only become active when an administrator applies the configuration in [.github/MAINTAINERS.md](.github/MAINTAINERS.md).

Prefer a merge commit to preserve contributors' original GPG signatures and DCO trailers. A maintainer-created merge commit must also satisfy GitHub's signed-commit rule. When merging through GitHub, the merging maintainer must add their own matching `Signed-off-by` trailer to the merge message. Do not use squash or rebase merging to discard the audited commit history.

## Reporting problems

Use the issue templates for bugs and features. Include reproduction steps, expected and actual behavior, platform, Go version, and redacted logs. Report exploitable vulnerabilities through [private vulnerability reporting](https://github.com/BurpBridge/burpbridge-core/security/advisories/new); see [SECURITY.md](SECURITY.md).

Follow the [code of conduct](CODE_OF_CONDUCT.md), explain disagreements constructively, and allow maintainers time to respond.
