"""Serves the generated events and console lines the way WorldAPI does, for Grafana's Infinity tables."""
import json
import os
import time
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from urllib.parse import parse_qs, urlparse

HERE = os.environ.get("MOCK_OUT", os.path.dirname(os.path.abspath(__file__)))
EVENTS = json.load(open(os.path.join(HERE, "events.json")))
LOGS = json.load(open(os.path.join(HERE, "logs.json")))


class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        url = urlparse(self.path)
        q = {k: v[0] for k, v in parse_qs(url.query).items()}
        limit = int(q.get("limit", 100))
        if url.path == "/api/v1/events":
            kinds = set(q["type"].split(",")) if "type" in q else None
            lo, hi = int(q.get("from", 0)), int(q.get("to", 10**15))
            items = [e for e in reversed(EVENTS) if (kinds is None or e["type"] in kinds) and lo <= e["atMillis"] <= hi]
            body = {"items": items[:limit], "nextCursor": None}
        elif url.path == "/api/v1/server/logs":
            levels = {"warn": {"warn", "error", "fatal"}, "error": {"error", "fatal"}}.get(q.get("level"), None)
            # The day runs a little past now; the console has written nothing from the future yet.
            now = time.strftime("%Y-%m-%dT%H:%M:%S", time.gmtime())
            lines = [l for l in reversed(LOGS) if (levels is None or l["level"] in levels) and l["at"] <= now]
            body = {"lines": lines[:limit]}
        else:
            self.send_response(404)
            self.end_headers()
            return
        data = json.dumps(body).encode()
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)


ThreadingHTTPServer(("0.0.0.0", 8110), Handler).serve_forever()
