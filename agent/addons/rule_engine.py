import fnmatch
import logging
import re

from mitmproxy import http

logger = logging.getLogger("rule_engine")


ACTION_INTERCEPT = "intercept"
ACTION_MODIFY_HEADERS = "modify_headers"
ACTION_MODIFY_BODY = "modify_body"
ACTION_REDIRECT = "redirect"
ACTION_DROP = "drop"
ACTION_RECORD = "record"
ACTIONS = {
    ACTION_INTERCEPT,
    ACTION_MODIFY_HEADERS,
    ACTION_MODIFY_BODY,
    ACTION_REDIRECT,
    ACTION_DROP,
    ACTION_RECORD,
}


class RuleEngine:
    def __init__(self, rules):
        self.rules = []
        self.set_rules(rules or [])

    def set_rules(self, rules):
        parsed = []
        for r in rules:
            try:
                parsed.append(self._compile(r))
            except Exception as e:
                logger.warning("skipping invalid rule %s: %s", r.get("id"), e)
                continue
            if parsed and parsed[-1]["action"] not in ACTIONS:
                logger.warning("skipping unknown action in rule %s", r.get("id"))
                parsed.pop()
        parsed.sort(key=lambda r: r["priority"], reverse=True)
        self.rules = parsed
        logger.info("loaded %d rules", len(self.rules))

    def _compile(self, rule):
        match = rule.get("match", {})
        return {
            "id": rule.get("id"),
            "node": rule.get("node", "*"),
            "priority": int(rule.get("priority", 0)),
            "action": rule["action"],
            "enabled": bool(rule.get("enabled", True)),
            "spec": rule.get("spec", {}),
            "host_re": self._compile_glob(match.get("host", "*")),
            "path_re": self._compile_glob(match.get("path", "*")),
            "method": (match.get("method") or "*").upper(),
        }

    def _compile_glob(self, pattern):
        return re.compile(fnmatch.translate(pattern or "*"))

    def matching(self, flow):
        for r in self.rules:
            if not r["enabled"]:
                continue
            if r["action"] not in ACTIONS:
                continue
            req = flow.request
            if r["method"] != "*" and (req.method or "").upper() != r["method"]:
                continue
            if not r["host_re"].match(req.host):
                continue
            if not r["path_re"].match(req.path.split("?", 1)[0]):
                continue
            yield r

    def request(self, flow):
        for rule in self.matching(flow):
            self._apply_request(flow, rule)

    def _apply_request(self, flow, rule):
        spec = rule["spec"]
        if rule["action"] == ACTION_DROP:
            flow.kill()
        elif rule["action"] == ACTION_REDIRECT:
            target = spec.get("url")
            if target:
                flow.request.url = target
        elif rule["action"] == ACTION_MODIFY_HEADERS:
            for k, v in spec.get("set", {}).items():
                flow.request.headers[k] = v
            for k in spec.get("remove", []):
                flow.request.headers.pop(k, None)
        elif rule["action"] == ACTION_MODIFY_BODY:
            ctype = flow.request.headers.get("content-type", "")
            text = spec.get("text")
            if "application/json" in ctype and text is not None:
                try:
                    import json
                    data = flow.request.json()
                    data.update(json.loads(text))
                    flow.request.text = json.dumps(data)
                except Exception as e:
                    logger.warning("body modify failed: %s", e)
            elif text is not None:
                flow.request.text = text
        elif rule["action"] == ACTION_INTERCEPT:
            flow.intercept()
        # ACTION_RECORD handled by flow_capture; no request-side change

    def response(self, flow):
        for rule in self.matching(flow):
            if rule["action"] == ACTION_MODIFY_HEADERS:
                for k, v in rule["spec"].get("resp_set", {}).items():
                    flow.response.headers[k] = v
                for k in rule["spec"].get("resp_remove", []):
                    flow.response.headers.pop(k, None)
