import logging
import re
import subprocess
import sys
import time

logger = logging.getLogger("process_resolver")


class ProcessResolver:
    def __init__(self, cache_ttl=30):
        self._cache = {}
        self._cache_ttl = cache_ttl

    def resolve(self, ip, port):
        if ip not in ("127.0.0.1", "::1", "localhost", "0.0.0.0"):
            return None
        if port is None:
            return None
        key = (ip, int(port))
        now = time.time()
        cached = self._cache.get(key)
        if cached and now - cached["ts"] < self._cache_ttl:
            return cached["name"]
        name = self._resolve(port)
        self._cache[key] = {"name": name, "ts": now}
        return name

    def _resolve(self, port):
        if sys.platform == "darwin":
            return self._resolve_macos(port)
        elif sys.platform.startswith("linux"):
            return self._resolve_linux(port)
        return None

    def _resolve_macos(self, port):
        try:
            result = subprocess.run(
                ["lsof", "-i", "TCP:%d" % port, "-F", "pcn"],
                capture_output=True,
                text=True,
                timeout=5,
            )
            if result.returncode != 0 or not result.stdout.strip():
                return None
            lines = result.stdout.strip().split("\n")
            name = None
            for line in lines:
                if line.startswith("c") and len(line) > 1:
                    name = line[1:]
            return name
        except Exception as e:
            logger.debug("lsof error on port %d: %s", port, e)
            return None

    def _resolve_linux(self, port):
        try:
            result = subprocess.run(
                ["ss", "-Hp", "sport", "= :%d" % port],
                capture_output=True,
                text=True,
                timeout=5,
            )
            if result.returncode != 0 or not result.stdout.strip():
                return None
            line = result.stdout.strip()
            m = re.search(r'users:\(\("([^"]+)"', line)
            if m:
                return m.group(1)
            return None
        except Exception as e:
            logger.debug("ss error on port %d: %s", port, e)
            return None

    def clear_cache(self):
        self._cache.clear()
