import json
from pathlib import Path
import sys
import unittest
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import publish_github_setup as publisher


class PublishTests(unittest.TestCase):
    def test_actions_configuration_preserves_allowed_action_restrictions(self):
        state = {"enabled": False, "allowed_actions": "selected"}
        permissions = {"default_workflow_permissions": "write", "can_approve_pull_request_reviews": True}
        writes = []
        def api(path, *args, data=None):
            if path.endswith("/workflow"):
                target = permissions
            elif path.endswith("/actions/permissions"):
                target = state
            else:
                return {"permissions": {"admin": True}}
            if data is not None:
                writes.append(data.copy())
                target.update(data)
            return target.copy()
        with patch.object(publisher, "gh", side_effect=api), patch("builtins.print"):
            publisher.configure_actions("BurpBridge/burpbridge-core")
            publisher.configure_actions("BurpBridge/burpbridge-core")
        self.assertEqual(state["allowed_actions"], "selected")
        self.assertEqual(writes, [{"enabled": True}, publisher.WORKFLOW_PERMISSIONS])

    def test_actions_require_admin_and_verify_result(self):
        with patch.object(publisher, "gh", return_value={"permissions": {"admin": False}}) as api:
            with self.assertRaises(RuntimeError):
                publisher.configure_actions("BurpBridge/burpbridge-core")
        self.assertEqual(api.call_count, 1)
        def api(path, *args, data=None):
            if path.endswith("/actions/permissions"):
                return {"enabled": False}
            if path.endswith("/workflow"):
                return publisher.WORKFLOW_PERMISSIONS.copy()
            return {"permissions": {"admin": True}}
        with patch.object(publisher, "gh", side_effect=api):
            with self.assertRaises(RuntimeError):
                publisher.configure_actions("BurpBridge/burpbridge-core")

    def test_issue_publication_is_idempotent_including_closed_issues(self):
        seeds = {
            "repository": "BurpBridge/burpbridge-core",
            "labels": [{"name": "bug"}, {"name": "good first issue"}],
            "issues": [
                {"slug": "old", "title": "Existing", "body": "old", "labels": ["bug"]},
                {"slug": "new", "title": "New", "body": "new", "labels": ["good first issue"]},
            ],
        }
        writes = []
        def api(path, *args, data=None):
            if data is not None:
                writes.append((path, data))
                return {"title": data.get("title"), "body": data.get("body"), "html_url": "https://github.com/example/new"}
            return {"has_issues": True}
        def pages(path):
            if "/labels?" in path:
                return [{"name": "bug"}]
            return [{"title": "Renamed", "body": "<!-- burpbridge-assessment:old -->", "state": "closed", "html_url": "https://github.com/example/old"}]
        with patch.object(publisher, "gh", side_effect=api), patch.object(publisher, "pages", side_effect=pages), patch("builtins.print"):
            publisher.publish_issues(seeds)
        self.assertEqual([data.get("name") for _, data in writes if "name" in data], ["good first issue"])
        self.assertEqual([data.get("title") for _, data in writes if "title" in data], ["New"])

    def test_no_writes_if_github_is_unreachable(self):
        seeds = {"repository": "BurpBridge/burpbridge-core"}
        with patch.object(publisher, "gh", side_effect=RuntimeError("unreachable")) as api:
            with self.assertRaises(RuntimeError):
                publisher.publish_issues(seeds)
        self.assertEqual(api.call_count, 1)

    def test_rules_require_admin_before_any_write(self):
        with patch.object(publisher, "gh", return_value={"permissions": {"admin": False}}) as api:
            with self.assertRaises(RuntimeError):
                publisher.configure_rules("BurpBridge/burpbridge-core")
        self.assertEqual(api.call_count, 1)

    def test_rules_refuse_unpublished_workflow(self):
        import base64
        writes = []
        def api(path, *args, data=None):
            if data:
                writes.append(data)
            if "/contents/" in path:
                return {"content": base64.b64encode(b"unreviewed workflow").decode()}
            return {"permissions": {"admin": True}, "default_branch": "main"}
        with patch.object(publisher, "gh", side_effect=api):
            with self.assertRaises(RuntimeError):
                publisher.configure_rules("BurpBridge/burpbridge-core")
        self.assertEqual(writes, [])

    def test_manifest_has_unique_slugs_and_beginner_acceptance_criteria(self):
        seeds = json.loads((publisher.ROOT / ".github/issue-seeds.json").read_text())
        slugs = [i["slug"] for i in seeds["issues"]]
        self.assertEqual(len(slugs), len(set(slugs)))
        labels = {label["name"] for label in seeds["labels"]}
        beginners = []
        for issue in seeds["issues"]:
            self.assertTrue(set(issue["labels"]).issubset(labels))
            self.assertIn("## Acceptance criteria", issue["body"])
            if "good first issue" in issue["labels"]:
                beginners.append(issue)
                self.assertIn("## First-time contributor guidance", issue["body"])
        self.assertGreaterEqual(len(beginners), 4)

    def test_rules_use_merge_sha_for_ci_and_head_sha_for_commit_policy(self):
        import base64
        writes = []
        rules = json.loads((publisher.ROOT / ".github/rulesets/default-branch.json").read_text())
        def api(path, *args, data=None):
            if data is not None:
                writes.append((path, data))
                return {"id": 42}
            if "/contents/" in path:
                file = path.split("/contents/")[1].split("?")[0]
                return {"content": base64.b64encode((publisher.ROOT / file).read_bytes()).decode()}
            if "/pulls/1" in path:
                return {"state": "open", "head": {"sha": "head"}, "merge_commit_sha": "merge"}
            if "/commits/merge/check-runs" in path:
                return {"check_runs": [{"name": "Go checks", "conclusion": "success", "app": {"id": 15368}}]}
            if "/commits/head/status" in path:
                return {"statuses": [{"context": "DCO and GPG", "state": "success"}]}
            if "/rulesets/42" in path:
                return {**rules, "id": 42}
            if path == "repos/BurpBridge/burpbridge-core":
                return {"permissions": {"admin": True}, "default_branch": "main"}
            raise AssertionError(f"Unexpected request: {path}")
        def pages(path):
            if "/pulls?" in path:
                return [{"number": 1, "base": {"ref": "main"}}]
            if "/rulesets?" in path:
                return [{"name": rules["name"], "id": 42}]
            raise AssertionError(path)
        with patch.object(publisher, "gh", side_effect=api), patch.object(publisher, "pages", side_effect=pages), patch("builtins.print"):
            publisher.configure_rules("BurpBridge/burpbridge-core")
        self.assertEqual(writes[0][0], "repos/BurpBridge/burpbridge-core/rulesets/42")
        self.assertEqual(writes[0][1], rules)
        self.assertEqual(writes[1][1]["allow_squash_merge"], False)


if __name__ == "__main__":
    unittest.main()
