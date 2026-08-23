#!/usr/bin/env python3
"""Scrape langflow.org use-case templates into the local gallery library.

Pipeline per template:
  sitemap slug -> detail page (SSR HTML) -> title/description/categories
               -> public railway deployment flow_id -> flow JSON (no auth)
               -> sanitize secrets -> native envelope
               -> templates/gallery/<category>/<subcategory>/<slug>.json

Usage:
  python3 scrape_gallery.py --one <slug>          # PoC single template
  python3 scrape_gallery.py [--limit N]           # bulk from sitemap
"""
import argparse, html, json, re, sys, time, urllib.request, gzip

SITE = "https://www.langflow.org"
GALLERY = "templates/gallery"
UA = {"User-Agent": "Mozilla/5.0 (X11; Linux x86_64) gallery-ingest/1.0"}

SECRET_NAME_RE = re.compile(r"(^|_)(api_?key|token|secret|password)$", re.I)
SK_VALUE_RE = re.compile(r"\b(sk-[A-Za-z0-9_-]{16,}|Bearer\s+[A-Za-z0-9._-]{20,})")

RAILWAY_RE = re.compile(
    r"https://([a-z0-9-]+)\.up\.railway\.app/flow/([0-9a-fA-F-]{36})")
CATS_RE = re.compile(r"/use-cases\?categories=([^\"\\]+)")


def http_get(url, timeout=60):
    req = urllib.request.Request(url, headers={**UA, "Accept-Encoding": "gzip"})
    r = urllib.request.urlopen(req, timeout=timeout)
    raw = r.read()
    if r.headers.get("Content-Encoding") == "gzip":
        raw = gzip.decompress(raw)
    return raw


def fetch_sitemap_slugs():
    xml = http_get(f"{SITE}/sitemap.xml").decode()
    return sorted(set(re.findall(
        r"<loc>" + re.escape(SITE) + r"/templates/(use-langflow-to-[^<]+)</loc>", xml)))


def parse_detail_page(page_html):
    """-> dict(title, description, category, subcategory, host, flow_id) or raises"""
    m = RAILWAY_RE.search(page_html)
    if not m:
        raise ValueError("no railway flow url in page")
    host, flow_id = m.group(1), m.group(2)

    t = re.search(r"<title>([^<]+)</title>", page_html)
    title = html.unescape(t.group(1)).split(" | Langflow")[0].strip() if t else ""

    d = re.search(r'name="description" content="([^"]*)"', page_html)
    desc = html.unescape(d.group(1)).strip() if d else ""

    cats = []  # ordered unique: primary first ("Business"), then "Business-Sales %26 Marketing"-style
    for raw in CATS_RE.findall(page_html):
        c = html.unescape(urllib.parse.unquote_plus(raw)).strip()
        if c and c not in cats:
            cats.append(c)
    # categories come as ["Business", "Business-Sales & Marketing"]; derive pair
    category, subcategory = "", ""
    for c in cats:
        parts = c.split("-", 1)
        if not category:
            category = parts[0]
        elif len(parts) == 2 and parts[0] == category and not subcategory:
            subcategory = parts[1]
    return {"title": title, "description": desc,
            "category": category or "misc", "subcategory": subcategory,
            "host": host, "flow_id": flow_id}


def sanitize_flow(data):
    """Mirror internal/templates/sanitize.go rule + blank sk-/Bearer literals."""
    warnings = []
    for n in data.get("nodes", []):
        nd = n.get("data", {})
        ntype = nd.get("type", "?")
        tpl = nd.get("node", {}).get("template", {})
        for fname, field in tpl.items():
            if not isinstance(field, dict):
                continue
            is_secret = bool(field.get("password")) or SECRET_NAME_RE.search(fname)
            val = field.get("value")
            if not is_secret or val in (None, ""):
                continue
            field["value"] = ""
            warnings.append(f"{ntype}.{fname}: secret value blanked")
    txt = json.dumps(data)
    leaked = SK_VALUE_RE.findall(txt)
    for lit in set(leaked):
        warnings.append(f"literal secret pattern left after blanking: {lit[:12]}...")
    return data, warnings


def save_template(meta, flow_json):
    import pathlib
    name = meta["title"] or meta["slug"]
    env = {
        "name": name,
        "description": meta["description"],
        "endpoint_name": None,
        "is_component": False,
        "last_tested_version": "1.11",
        "locked": False,
        "tags": [c for c in (meta["category"], meta["subcategory"]) if c],
        "data": flow_json.get("data"),
    }
    cat = slug_dir(meta["category"])
    sub = slug_dir(meta["subcategory"]) if meta["subcategory"] else None
    parts = [GALLERY, cat] + ([sub] if sub else [])
    out = pathlib.Path(*parts) / f"{meta['slug']}.json"
    out.parent.mkdir(parents=True, exist_ok=True)
    out.write_text(json.dumps(env, ensure_ascii=False))
    return str(out)


def slug_dir(s):
    s = re.sub(r"[^a-z0-9]+", "_", s.lower()).strip("_")
    return s or "misc"


def ingest_slug(slug):
    page = http_get(f"{SITE}/templates/{slug}").decode(errors="replace")
    meta = parse_detail_page(page)
    meta["slug"] = slug
    raw = http_get(f"https://{meta['host']}.up.railway.app/api/v1/flows/{meta['flow_id']}")
    flow = json.loads(raw)
    data, warnings = sanitize_flow(flow.get("data") or {})
    path = save_template(meta, {"data": data})
    return {**meta, "path": path, "nodes": len(data.get("nodes", [])),
            "edges": len(data.get("edges", [])), "warnings": warnings}


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--one", help="ingest a single template slug (PoC)")
    ap.add_argument("--limit", type=int, default=0)
    a = ap.parse_args()

    slugs = [a.one] if a.one else fetch_sitemap_slugs()
    if a.limit:
        slugs = slugs[:a.limit]
    print(f"{len(slugs)} template(s) to ingest\n")
    ok = fail = 0
    manifest = []
    for i, slug in enumerate(slugs, 1):
        try:
            r = ingest_slug(slug)
            ok += 1
            print(f"  [{i}/{len(slugs)}] OK   {r['title'][:44]:46} "
                  f"{r['nodes']}n/{r['edges']}e  {r['path'].replace(GALLERY + '/', '')}"
                  + (f"  W:{len(r['warnings'])}" if r["warnings"] else ""))
            manifest.append({"slug": slug, **{k: r[k] for k in
                             ("title", "description", "category", "subcategory",
                              "host", "flow_id", "path", "warnings")}})
        except Exception as e:
            fail += 1
            print(f"  [{i}/{len(slugs)}] FAIL {slug[:52]:54} {str(e)[:70]}")
            manifest.append({"slug": slug, "error": str(e)[:300]})
        time.sleep(0.4)  # be polite
    json.dump(manifest, open("templates/gallery/manifest.json", "w"),
              ensure_ascii=False, indent=1)
    print(f"\nDONE: {ok} ingested, {fail} failed")


if __name__ == "__main__":
    main()
