class PocketBaseClient:
    def __init__(self, hub_url, token):
        self.hub_url = hub_url
        self.token = token
        # TODO: wrap pocketbase SDK; auth as node, batch flow writes
