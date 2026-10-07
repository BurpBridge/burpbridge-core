# Activating contributor and merge checks

These files configure the intended policy, but local files alone do not activate GitHub rules or publish issues. The publishing tool never commits or pushes the repository.

## Bootstrap order

First inspect the live repository settings and enable Actions with a read-only default token and bot PR approval disabled:

```sh
python3 scripts/publish_github_setup.py audit
python3 scripts/publish_github_setup.py actions --apply
```

The Actions command preserves existing allowed-action restrictions and verifies the resulting settings. Organization policies may prevent changes. It does not activate merge protections; the workflows must be published first. GitHub's [Actions permissions API](https://docs.github.com/en/rest/actions/permissions) documents these settings.

1. Review the new workflows, contribution guide, scripts, and issue seeds. Commit this setup with your own `git commit -s -S` and publish it through the repository's current contribution process. Do not fabricate a contributor's signature.
2. Confirm the files exist on the remote default branch. Open a small, signed and sign-off-compliant PR to that branch and leave it open for activation. Both `Go checks` (on the test-merge SHA) and the `DCO and GPG` status (on its head SHA) must pass. The policy workflow needs its trusted script on the base branch before it can enforce future PRs. Fork workflows may need a maintainer's normal first-contributor Actions approval.
3. With authenticated `gh` and repository administration access, run:

   ```sh
   python3 scripts/publish_github_setup.py issues
   python3 scripts/publish_github_setup.py issues --apply
   python3 scripts/publish_github_setup.py rules
   python3 scripts/publish_github_setup.py rules --apply
   ```

The issue publisher creates missing labels and skips matching existing issues, including closed issues. It does not change existing label descriptions or reopen closed work. The rules publisher checks that the remote workflow/script bytes match the reviewed local files, checks for successful PR contexts, and creates or updates only its named ruleset. Other existing rulesets remain in place.

After applying the rules, rerun `python3 scripts/publish_github_setup.py audit` to inspect effective default-branch rules, workflows, Actions permissions, and merge methods. For a report to share, redirect it to `/tmp/burpbridge-settings.json` rather than committing a snapshot of live settings.

If a required check fails because of an existing build problem, fix that problem through review before enabling merge requirements. Do not silently remove the check to make the branch look healthy.

## Rules

The default branch requires one approving review, dismissal of stale approvals, approval by someone other than the last pusher, resolved conversations, verified signed commits, and an up-to-date branch. `Go checks` and `DCO and GPG` are required from the GitHub Actions integration. No bypass actors are configured. PRs are required, branch deletion and force pushes are blocked, and only merge commits are enabled to preserve contributor signatures and trailers.

The GitHub signed-commit rule accepts multiple signature formats. The additional commit-policy status requires **OpenPGP/GPG** and valid GitHub verification on every PR commit. Every author and trailer-listed co-author needs a matching DCO sign-off. Bot commits follow the same rules and often need a maintainer-authored replacement.

Before a GitHub UI merge, the merging maintainer must add their own matching `Signed-off-by` trailer to the merge message. The generated merge commit is signed by GitHub. Its DCO trailer is a manual maintainer responsibility: the PR-head policy check audits the incoming commits, not the merge commit GitHub creates afterward.

The commit-policy workflow uses `pull_request_target` solely for metadata. It checks out the exact trusted base SHA, never fetches PR code, has no persisted Git credentials, and posts a status explicitly on the PR head SHA. Keep this boundary intact. Changes to the checker are tested in ordinary read-only PR CI and only govern other PRs after being reviewed and merged.

GitHub cannot prevent a reviewer from clicking Approve early. Maintainers should wait for passing checks; the rules block **merging** until all requirements are met. CODEOWNERS enforcement is not enabled because no maintainer team has been verified. Add a real team and enable code-owner reviews if governance/workflow changes should need a specific team's approval.

## Reference and limitations

The user requested [OSSAfrica/skillguard](https://github.com/OSSAfrica/skillguard) as a reference. Its public overview was accessible, but its exact CONTRIBUTING.md and workflow contents were unavailable during setup; this configuration does not claim to reproduce unverified settings.

Useful primary references: [DCO 1.1](https://developercertificate.org/), [GitHub commit signing](https://docs.github.com/en/authentication/managing-commit-signature-verification/signing-commits), and [available ruleset rules](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/available-rules-for-rulesets).

Local Python policy tests do not establish that the engine builds or that live GitHub permissions are sufficient. Validate the Go workflow and live rule enforcement before declaring this setup active.
