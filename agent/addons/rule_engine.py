import logging

logger = logging.getLogger("rule_engine")


class RuleEngine:
    def __init__(self, rules):
        self.rules = rules

    def set_rules(self, rules):
        # TODO: sort by priority, validate spec
        self.rules = rules
        logger.info("loaded %d rules", len(rules))

    def request(self, flow):
        # TODO: match flow against rules; apply action (intercept/modify/redirect/drop)
        pass

    def response(self, flow):
        # TODO: apply response-side rules
        pass
