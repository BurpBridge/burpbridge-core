"""Preview or apply the reviewed issue seeds and default-branch rules via gh.

Run from any directory; paths are resolved relative to this script. This tool does
not commit or push files. Rules are applied only after trusted workflows exist on
the default branch and both required contexts have reported success for an open PR.
"""

import argparse
import json
from pathlib import Path
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[1]
WORKFLOW_PERMISSIONS = {
    "default_workflow_permissions": "read",
    "can_approve_pull_request_reviews": False,
}


def gh(*args, data=None):
    command = ["gh", "api", *args]
    if data is not None:
        command += ["--input", "-"]
    result = subprocess.run(command, input=None if data is None else json.dumps(data),
                            text=True, capture_output=True, check=False)
    if result.returncode:
        raise RuntimeError(result.stderr.strip() or "GitHub API request failed")
    return json.loads(result.stdout) if result.stdout.strip() else None


def pages(endpoint):
    return [item for page in gh(endpoint, "--paginate", "--slurp") for item in page]


def publish_issues(seeds):
    repo = seeds["repository"]
    metadata = gh(f"repos/{repo}")
    if not metadata.get("has_issues"):
        raise RuntimeError("Issues are disabled; enable them before publishing")
    # Read everything before writing, and leave existing labels/issues untouched.
    labels = {label["name"] for label in pages(f"repos/{repo}/labels?per_page=100")}
    existing = pages(f"repos/{repo}/issues?state=all&per_page=100")
    for label in seeds["labels"]:
        if label["name"] not in labels:
            gh(f"repos/{repo}/labels", "--method", "POST", data=label)
    for issue in seeds["issues"]:
        marker = f"<!-- burpbridge-assessment:{issue['slug']} -->"
        match = next((item for item in existing if "pull_request" not in item and
                      (marker in (item.get("body") or "") or item["title"] == issue["title"])), None)
        if match:
            print(f"Existing: {match['html_url']}")
            continue
        created = gh(f"repos/{repo}/issues", "--method", "POST", data={
            key: issue[key] for key in ("title", "body", "labels")
        })
        existing.append(created)
        print(f"Created: {created['html_url']}")


def audit_settings(repo):
    from urllib.parse import quote
    metadata = gh(f"repos/{repo}")
    report = {"repository": {key: metadata.get(key) for key in (
        "full_name", "default_branch", "permissions", "allow_merge_commit",
        "allow_squash_merge", "allow_rebase_merge",
    )}}
    branch = quote(metadata["default_branch"], safe="")
    endpoints = {
        "actions": f"repos/{repo}/actions/permissions",
        "workflow_permissions": f"repos/{repo}/actions/permissions/workflow",
        "workflows": f"repos/{repo}/actions/workflows?per_page=100",
        "effective_default_branch_rules": f"repos/{repo}/rules/branches/{branch}",
        "rulesets": f"repos/{repo}/rulesets?per_page=100",
    }
    for name, endpoint in endpoints.items():
        try:
            report[name] = gh(endpoint)
        except RuntimeError as exc:
            report[name] = {"error": str(exc)}
    print(json.dumps(report, indent=2))
    return not any(isinstance(value, dict) and "error" in value for value in report.values())


def configure_actions(repo):
    metadata = gh(f"repos/{repo}")
    if not metadata.get("permissions", {}).get("admin"):
        raise RuntimeError("Repository administration permission is required for Actions settings")
    # Read both before writing; keep existing allowed-action restrictions intact.
    actions = gh(f"repos/{repo}/actions/permissions")
    workflow = gh(f"repos/{repo}/actions/permissions/workflow")
    if actions.get("enabled") is not True:
        gh(f"repos/{repo}/actions/permissions", "--method", "PUT", data={"enabled": True})
    if any(workflow.get(key) != value for key, value in WORKFLOW_PERMISSIONS.items()):
        gh(f"repos/{repo}/actions/permissions/workflow", "--method", "PUT", data=WORKFLOW_PERMISSIONS)
    actions = gh(f"repos/{repo}/actions/permissions")
    workflow = gh(f"repos/{repo}/actions/permissions/workflow")
    if actions.get("enabled") is not True or any(
        workflow.get(key) != value for key, value in WORKFLOW_PERMISSIONS.items()
    ):
        raise RuntimeError("Actions settings did not take effect; inspect organization restrictions")
    print("Verified: Actions enabled, default token read-only, Actions cannot approve PRs")


def configure_rules(repo):
    metadata = gh(f"repos/{repo}")
    if not metadata.get("permissions", {}).get("admin"):
        raise RuntimeError("Repository administration permission is required for rules")
    branch = metadata["default_branch"]
    # Check the exact reviewed files, not merely that similarly named files exist.
    import base64
    for path in (".github/workflows/ci.yml", ".github/workflows/commit-policy.yml", "scripts/ci/check_commits.py"):
        remote = gh(f"repos/{repo}/contents/{path}?ref={branch}")
        if base64.b64decode(remote["content"]) != (ROOT / path).read_bytes():
            raise RuntimeError(f"Publish the reviewed {path} on {branch} before enabling rules")
    # Require an observed successful PR run, since commit-policy posts on PR heads.
    pulls = pages(f"repos/{repo}/pulls?state=open&per_page=100")
    ready = False
    for pr in pulls[:30]:
        if pr["base"]["ref"] != branch:
            continue
        pr = gh(f"repos/{repo}/pulls/{pr['number']}")
        sha = pr["head"]["sha"]
        # Ordinary PR CI runs on GitHub's test-merge SHA; policy runs on head SHA.
        merge_sha = pr.get("merge_commit_sha")
        if not merge_sha or pr["state"] != "open":
            continue
        checks = gh(f"repos/{repo}/commits/{merge_sha}/check-runs?per_page=100")["check_runs"]
        statuses = gh(f"repos/{repo}/commits/{sha}/status")["statuses"]
        go_ok = any(c["name"] == "Go checks" and c["conclusion"] == "success" and
                    c.get("app", {}).get("id") == 15368 for c in checks)
        policy = next((s for s in statuses if s["context"] == "DCO and GPG"), None)
        if go_ok and policy and policy["state"] == "success":
            ready = True
            break
    if not ready:
        raise RuntimeError("Run a signed, sign-off-compliant PR through both checks before enabling rules")
    rules = json.loads((ROOT / ".github/rulesets/default-branch.json").read_text())
    existing = pages(f"repos/{repo}/rulesets?includes_parents=false&per_page=100")
    matches = [r for r in existing if r["name"] == rules["name"]]
    if len(matches) > 1:
        raise RuntimeError("Duplicate named rulesets; reconcile them before applying")
    if matches:
        result = gh(f"repos/{repo}/rulesets/{matches[0]['id']}", "--method", "PUT", data=rules)
    else:
        result = gh(f"repos/{repo}/rulesets", "--method", "POST", data=rules)
    gh(f"repos/{repo}", "--method", "PATCH", data={
        "allow_merge_commit": True, "allow_squash_merge": False, "allow_rebase_merge": False,
    })
    live = gh(f"repos/{repo}/rulesets/{result['id']}")
    if live["enforcement"] != "active":
        raise RuntimeError("Ruleset was written but is not active")
    print(f"Active ruleset: https://github.com/{repo}/rules/{result['id']}")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("action", choices=["issues", "actions", "rules", "audit"])
    parser.add_argument("--apply", action="store_true", help="Write to GitHub; otherwise only preview")
    args = parser.parse_args()
    seeds = json.loads((ROOT / ".github/issue-seeds.json").read_text())
    if args.action == "audit" and args.apply:
        parser.error("audit is read-only; omit --apply")
    if not args.apply and args.action != "audit":
        if args.action == "issues":
            for issue in seeds["issues"]:
                print(f"{issue['title']} [{', '.join(issue['labels'])}]")
            print(f"Preview: {len(seeds['issues'])} issues for {seeds['repository']}")
        elif args.action == "actions":
            print(json.dumps({"enabled": True, **WORKFLOW_PERMISSIONS}, indent=2))
        else:
            print((ROOT / ".github/rulesets/default-branch.json").read_text())
        return 0
    try:
        if args.action == "audit":
            return 0 if audit_settings(seeds["repository"]) else 1
        elif args.action == "actions":
            configure_actions(seeds["repository"])
        elif args.action == "issues":
            publish_issues(seeds)
        else:
            configure_rules(seeds["repository"])
    except (RuntimeError, OSError) as exc:
        print(str(exc), file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
