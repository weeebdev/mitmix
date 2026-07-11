import tempfile
import json
from local_store import LocalStore


def make_store():
    return LocalStore(tempfile.mktemp(suffix=".db"))


def test_empty_store():
    s = make_store()
    assert s.unsynced_count() == 0
    assert s.rule_count() == 0
    assert s.load_rules() == []


def test_save_and_load_rules():
    s = make_store()
    rules = [{"id": "r1", "action": "record"}, {"id": "r2", "action": "drop"}]
    s.save_rules(rules)
    assert s.rule_count() == 2
    loaded = s.load_rules()
    assert len(loaded) == 2
    assert loaded[0]["id"] == "r1"


def test_overwrite_rules():
    s = make_store()
    s.save_rules([{"id": "r1"}])
    s.save_rules([{"id": "r2"}])
    assert s.rule_count() == 1
    assert s.load_rules()[0]["id"] == "r2"


def test_append_and_sync_flow():
    s = make_store()
    s.append_flow({"host": "a.com"})
    s.append_flow({"host": "b.com"})
    assert s.unsynced_count() == 2

    unsynced = s.get_unsynced(limit=10)
    assert len(unsynced) == 2
    assert unsynced[0][1]["host"] == "a.com"
    assert unsynced[1][1]["host"] == "b.com"

    s.mark_synced(unsynced[1][0])
    assert s.unsynced_count() == 0


def test_get_unsynced_returns_new_only():
    s = make_store()
    s.append_flow({"n": 1})
    s.append_flow({"n": 2})

    first = s.get_unsynced(limit=10)
    s.mark_synced(first[0][0])

    s.append_flow({"n": 3})

    second = s.get_unsynced(limit=10)
    assert len(second) == 2
    assert second[0][1]["n"] == 2
    assert second[1][1]["n"] == 3


def test_mark_synced_only_increases():
    s = make_store()
    s.append_flow({"n": 1})
    e = s.get_unsynced(limit=10)
    s.mark_synced(e[0][0])
    assert s.unsynced_count() == 0

    s.mark_synced(0)
    assert s.unsynced_count() == 0


def test_cleanup_old():
    s = make_store()
    s.append_flow({"host": "old"})
    e = s.get_unsynced(limit=10)
    s.mark_synced(e[0][0])

    s.append_flow({"host": "unsynced"})
    # synced flow should be cleaned, unsynced should stay
    unsynced = s.get_unsynced(limit=10)
    assert len(unsynced) == 1
    assert unsynced[0][1]["host"] == "unsynced"
