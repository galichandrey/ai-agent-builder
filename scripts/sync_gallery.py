#!/usr/bin/env python3
"""Bulk-import gallery templates into the LangFlow dashboard and verify each.

Drives bin/langflow-mcp over stdio JSON-RPC: list_templates(source=gallery) ->
create_flow_from_template(params={model_name: MODEL}) -> build_flow per template.
Existing flows with the same name are skipped, so the script is idempotent.

Usage:
  scripts/sync_gallery.py [--model hy3-free] [--out results.json] [--limit N]

Environment:
  LANGFLOW_URL   default http://127.0.0.1:7860
  LANGFLOW_API_KEY  required (or pass via --api-key)
"""
import argparse
import gzip
import json
import os
import re
import subprocess
import sys
import urllib.request

REPO = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
DEFAULT_BIN = os.path.join(REPO, "bin", "langflow-mcp")


class MCP:
    """Minimal MCP stdio client for driving langflow-mcp."""

    def __init__(self, binary, url, api_key):
        env = dict(os.environ,
                   LANGFLOW_MCP_LANGFLOW_URL=url,
                   LANGFLOW_MCP_API_KEY=api_key)
        self.p = subprocess.Popen([binary], stdin=subprocess.PIPE,
                                  stdout=subprocess.PIPE, stderr=sys.stderr, env=env)
        self.id = 0
        self._send("initialize", {"protocolVersion": "2024-11-05",
                                  "capabilities": {},
                                  "clientInfo": {"name": "sync-gallery", "version": "1"}})
        init = self._recv()
        assert "result" in init, init
        self._notify("notifications/initialized", {})

    def _send(self, method, params):
        self.id += 1
        msg = {"jsonrpc": "2.0", "id": self.id, "method": method}
        if params is not None:
            msg["params"] = params
        self.p.stdin.write((json.dumps(msg) + "\n").encode())
        self.p.stdin.flush()

    def _notify(self, method, params):
        msg = {"jsonrpc": "2.0", "method": method, "params": params}
        self.p.stdin.write((json.dumps(msg) + "\n").encode())
        self.p.stdin.flush()

    def _recv(self):
        while True:
            line = self.p.stdout.readline()
            if not line:
                raise RuntimeError("server closed")
            line = line.strip()
            if line:
                return json.loads(line)

    def call(self, name, args=None):
        self._send("tools/call", {"name": name, "arguments": args or {}})
        resp = self._recv()
        if "error" in resp:
            raise RuntimeError(f"{name}: {resp['error']}")
        content = resp["result"].get("content", [])
        text = content[0]["text"] if content else ""
        if resp["result"].get("isError"):
            raise RuntimeError(f"{name} tool error: {text}")
        try:
            return json.loads(text)
        except Exception:
            return text


def list_existing_flow_names(url, api_key):
    req = urllib.request.Request(url + "/api/v1/flows/",
                                 headers={"x-api-key": api_key,
                                          "Accept-Encoding": "gzip"})
    try:
        r = urllib.request.urlopen(req, timeout=60)
        raw = r.read()
        if r.headers.get("Content-Encoding") == "gzip":
            raw = gzip.decompress(raw)
        flows = json.loads(raw or b"[]")
        return {f.get("name", "") for f in
                (flows.get("flows") if isinstance(flows, dict) else flows) or []}
    except Exception as e:
        print(f"warning: cannot list existing flows ({e})", file=sys.stderr)
        return set()


def main():
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument("--binary", default=DEFAULT_BIN)
    ap.add_argument("--url", default=os.environ.get("LANGFLOW_URL", "http://127.0.0.1:7860"))
    ap.add_argument("--api-key", default=os.environ.get("LANGFLOW_API_KEY", ""))
    ap.add_argument("--model", default="hy3-free",
                    help="model applied to every instantiated flow (default: hy3-free)")
    ap.add_argument("--out", default="scripts/sync_gallery_results.json")
    ap.add_argument("--limit", type=int, default=0, help="only import first N templates")
    args = ap.parse_args()
    if not args.api_key:
        sys.exit("error: --api-key or LANGFLOW_API_KEY is required")

    existing = list_existing_flow_names(args.url, args.api_key)
    mcp = MCP(args.binary, args.url, args.api_key)
    cat = mcp.call("list_templates", {"source": "gallery"})
    templates = cat["templates"]
    if args.limit:
        templates = templates[:args.limit]
    print(f"importing {len(templates)} gallery templates (model: {args.model})\n")

    ok = comp = broken = skipped = 0
    results = {}
    for i, t in enumerate(templates, 1):
        name = t["name"] + " (from template)"
        if name in existing:
            skipped += 1
            print(f"  [{i}] SKIP {t['name'][:46]} (already imported)")
            continue
        try:
            inst = mcp.call("create_flow_from_template", {
                "template_name": t["slug"], "params": {"model_name": args.model}})
        except Exception as e:
            broken += 1
            results[t["slug"]] = {"import": str(e)[:120]}
            print(f"  [{i}] x IMPORT FAIL {t['name'][:44]}")
            continue
        fid = inst["flow_id"]
        try:
            build = mcp.call("build_flow", {"flow_id": fid})
            txt = json.dumps(build)
            comps = sorted(set(re.findall(
                r'[Ee]rror building [Cc]omponent ([A-Za-z0-9_ @:\-]+?)[",\\]', txt)))
            if '"error":true' not in txt and 'error building' not in txt.lower() and not comps:
                ok += 1
                results[t["slug"]] = {"flow_id": fid, "build": "ok"}
                print(f"  [{i}] OK BUILD    {t['name'][:46]}")
            else:
                comp += 1
                results[t["slug"]] = {"flow_id": fid, "build": "components",
                                      "components": comps}
                print(f"  [{i}] ~ COMPONENTS {t['name'][:40]:42} {', '.join(comps)[:44]}")
        except Exception as e:
            msg = str(e)
            comps = sorted(set(re.findall(
                r'Error building Component ([A-Za-z0-9_ @:\-]+)', msg)))[:5]
            if comps:
                comp += 1
                results[t["slug"]] = {"flow_id": fid, "build": "components", "components": comps}
                print(f"  [{i}] ~ COMPONENTS {t['name'][:40]:42} {', '.join(comps)[:44]}")
            else:
                broken += 1
                results[t["slug"]] = {"flow_id": fid, "build": "broken", "error": msg[:200]}
                print(f"  [{i}] x BROKEN     {t['name'][:44]} {msg[:50]}")

    with open(args.out, "w") as f:
        json.dump(results, f, indent=1)
    print(f"\nTOTAL: build_ok={ok}, component_notes={comp}, broken={broken}, skipped={skipped}")
    print(f"results written to {args.out}")


if __name__ == "__main__":
    main()
