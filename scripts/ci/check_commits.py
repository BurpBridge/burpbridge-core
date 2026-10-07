"""Check GitHub PR commit metadata using trusted base-branch code only."""

import json
import os
import re
import sys
import urllib.request

CONTEXT = "DCO and GPG"
IDENTITY = re.compile(r"(.+?)\s+<([^<>\s]+@[^<>\s]+)>")
TRAILER = re.compile(r"([A-Za-z][A-Za-z0-9-]*):\s*(.*)")


def identity(name, email):
    return (" ".join(name.split()), email.strip().casefold())


def validate_commit(record):
    commit = record["commit"]
    errors = []
    # Only the final trailer block counts; body examples are not certification.
    lines = commit["message"].rstrip().splitlines()
    block = []
    for line in reversed(lines):
        match = TRAILER.fullmatch(line)
        if not match:
            break
        block.append((match[1].casefold(), match[2]))
    signers = set()
    authors = {identity(commit["author"]["name"], commit["author"]["email"])}
    for key, value in block:
        if key not in {"signed-off-by", "co-authored-by"}:
            continue
        parsed = IDENTITY.fullmatch(value)
        if not parsed:
            errors.append(f"Malformed {key} trailer")
            continue
        target = signers if key == "signed-off-by" else authors
        target.add(identity(parsed[1], parsed[2]))
    if not authors.issubset(signers):
        errors.append("Missing author/co-author matching DCO sign-off; use git commit -s")
    verification = commit.get("verification", {})
    if verification.get("verified") is not True or verification.get("reason") != "valid":
        errors.append("GitHub must report a verified, valid signature")
    signature = verification.get("signature") or ""
    if not signature.startswith("-----BEGIN PGP SIGNATURE-----"):
        errors.append("An OpenPGP/GPG signature is required; use git commit -S")
    return errors


class GitHub:
    def __init__(self, repository, token):
        if not re.fullmatch(r"[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+", repository):
            raise ValueError("Invalid repository")
        self.root = f"https://api.github.com/repos/{repository}"
        self.token = token

    def request(self, path, data=None):
        req = urllib.request.Request(
            self.root + path,
            data=None if data is None else json.dumps(data).encode(),
            headers={
                "Authorization": f"Bearer {self.token}",
                "Accept": "application/vnd.github+json",
                "X-GitHub-Api-Version": "2022-11-28",
                "Content-Type": "application/json",
            },
        )
        with urllib.request.urlopen(req, timeout=30) as response:
            return json.load(response)


def check_pull_request(api, number, expected_head):
    pr = api.request(f"/pulls/{number}")
    if pr["head"]["sha"] != expected_head:
        raise ValueError("PR head changed; wait for the newest policy run")
    count = pr["commits"]
    if not 1 <= count <= 250:
        raise ValueError("PR must contain 1–250 commits; refusing incomplete validation")
    records = []
    for page in range(1, (count + 99) // 100 + 1):
        records.extend(api.request(f"/pulls/{number}/commits?per_page=100&page={page}"))
    if len(records) != count or len({r["sha"] for r in records}) != count:
        raise ValueError("Incomplete or inconsistent commit list; rerun the policy check")
    if records[-1]["sha"] != expected_head:
        raise ValueError("Commit list does not match the expected PR head")
    failures = []
    for record in records:
        # List-PR-commits responses may omit verification; fetch full metadata.
        if "verification" not in record["commit"]:
            record = api.request(f"/commits/{record['sha']}")
        for error in validate_commit(record):
            failures.append(f"{record['sha'][:12]}: {error}")
    if api.request(f"/pulls/{number}")["head"]["sha"] != expected_head:
        raise ValueError("PR head changed during validation")
    return failures


def main():
    api = GitHub(os.environ["GITHUB_REPOSITORY"], os.environ["GH_TOKEN"])
    number = int(os.environ["PR_NUMBER"])
    head = os.environ["EXPECTED_HEAD"]
    if not re.fullmatch(r"[0-9a-f]{40}", head) or number < 1:
        raise ValueError("Invalid PR number or head SHA")
    status = {
        "context": CONTEXT,
        "target_url": f"https://github.com/{os.environ['GITHUB_REPOSITORY']}/actions/runs/{os.environ['GITHUB_RUN_ID']}",
    }
    # Explicitly attach the required status to the PR HEAD, not the trusted base.
    api.request(f"/statuses/{head}", {**status, "state": "pending", "description": "Checking every PR commit"})
    try:
        failures = check_pull_request(api, number, head)
        for failure in failures:
            print(failure)
        state = "failure" if failures else "success"
        description = "Commit policy failed; see run logs" if failures else "Every commit has DCO and a verified GPG signature"
        api.request(f"/statuses/{head}", {**status, "state": state, "description": description})
        return bool(failures)
    except Exception as exc:
        # Do not print API response bodies, tokens, or untrusted commit messages.
        detail = str(exc) if isinstance(exc, ValueError) else type(exc).__name__
        print(f"Commit policy could not complete: {detail}", file=sys.stderr)
        api.request(f"/statuses/{head}", {**status, "state": "failure", "description": "Could not validate all commits; see run logs"})
        return 1


if __name__ == "__main__":
    sys.exit(main())
