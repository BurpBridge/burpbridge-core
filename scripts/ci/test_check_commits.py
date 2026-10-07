import copy
import unittest

from check_commits import check_pull_request, validate_commit


def commit(sha="a" * 40):
    return {
        "sha": sha,
        "commit": {
            "author": {"name": "First Contributor", "email": "first@example.com"},
            "message": "docs: clarify setup\n\nSigned-off-by: First Contributor <first@example.com>",
            "verification": {
                "verified": True,
                "reason": "valid",
                "signature": "-----BEGIN PGP SIGNATURE-----\nfixture",
            },
        },
    }


class CommitPolicyTests(unittest.TestCase):
    def test_valid_commit(self):
        self.assertEqual(validate_commit(commit()), [])

    def test_missing_and_wrong_identity_signoffs(self):
        for message in ["docs: change", "docs: change\n\nSigned-off-by: Someone Else <else@example.com>"]:
            record = commit()
            record["commit"]["message"] = message
            self.assertTrue(validate_commit(record))

    def test_body_example_does_not_count(self):
        record = commit()
        record["commit"]["message"] += "\n\nThis is only a documentation example."
        self.assertTrue(validate_commit(record))

    def test_coauthor_needs_own_signoff(self):
        record = commit()
        record["commit"]["message"] += "\nCo-authored-by: Second Contributor <second@example.com>"
        self.assertTrue(validate_commit(record))
        record["commit"]["message"] += "\nSigned-off-by: Second Contributor <second@example.com>"
        self.assertEqual(validate_commit(record), [])

    def test_malformed_trailer(self):
        record = commit()
        record["commit"]["message"] += "\nCo-authored-by: missing-email"
        self.assertTrue(validate_commit(record))

    def test_unsigned_invalid_and_non_gpg_commits_fail(self):
        for field, value in [("verified", False), ("reason", "unknown_key"), ("signature", "-----BEGIN SSH SIGNATURE-----"), ("signature", None)]:
            record = commit()
            record["commit"]["verification"][field] = value
            self.assertTrue(validate_commit(record))

    def test_missing_verification_fails(self):
        record = commit()
        del record["commit"]["verification"]
        self.assertTrue(validate_commit(record))

    def test_paginate_and_fetch_missing_verification(self):
        records = [commit(f"{i:040x}") for i in range(1, 102)]
        class API:
            def request(self, path):
                if "/commits?" in path:
                    page = int(path.split("page=")[-1])
                    result = copy.deepcopy(records[(page - 1) * 100:page * 100])
                    for record in result:
                        del record["commit"]["verification"]
                    return result
                if path.startswith("/commits/"):
                    return commit(path.split("/")[-1])
                return {"head": {"sha": records[-1]["sha"]}, "commits": len(records)}
        self.assertEqual(check_pull_request(API(), 1, records[-1]["sha"]), [])

    def test_incomplete_duplicate_and_changed_head_fail(self):
        for count, records, head in [(2, [commit()], "a" * 40), (2, [commit(), commit()], "a" * 40), (1, [commit()], "b" * 40), (251, [], "a" * 40)]:
            class API:
                def request(self, path):
                    if "/commits?" in path:
                        return records
                    return {"head": {"sha": head}, "commits": count}
            with self.assertRaises(ValueError):
                check_pull_request(API(), 1, "a" * 40)

    def test_head_changes_during_check(self):
        class API:
            calls = 0
            def request(self, path):
                if "/commits?" in path:
                    return [commit()]
                self.calls += 1
                return {"head": {"sha": ("a" if self.calls == 1 else "b") * 40}, "commits": 1}
        with self.assertRaises(ValueError):
            check_pull_request(API(), 1, "a" * 40)


if __name__ == "__main__":
    unittest.main()
