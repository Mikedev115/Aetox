#!/usr/bin/env python3
"""usage-report.py — what the last few sessions actually cost, from the
desktop's own token_usage table.

The desktop records one row per provider request: prompt tokens, how many of
those the provider served from its cache, and completion tokens. This turns
that into the number a person wants — per session, how many requests, how many
tokens were paid at full price, and what that comes to in money at the
catalog's price for the model — so a change can be judged by what it cost
rather than by how it felt.

Usage:
    python scripts/usage-report.py            # today's sessions
    python scripts/usage-report.py --last 8   # the last 8 sessions
    python scripts/usage-report.py --since 2026-09-05T12:00

Prices come from model-catalog.json beside the database (USD per million
tokens: input, cacheRead, output). A model the catalog does not price is shown
in tokens only.
"""

import argparse
import datetime as dt
import json
import os
import sqlite3
import sys


def data_root() -> str:
    override = os.environ.get("AETOX_DATA_ROOT", "").strip()
    if override:
        return override
    if sys.platform == "win32":
        return os.path.join(os.environ["APPDATA"], "aetox")
    return os.path.join(os.path.expanduser("~"), ".config", "aetox")


def prices(root: str) -> dict:
    """model name -> (input, cacheRead, output) USD per million tokens."""
    path = os.path.join(root, "model-catalog.json")
    out = {}
    try:
        catalog = json.load(open(path, encoding="utf-8"))
    except (OSError, ValueError):
        return out

    def walk(o):
        if isinstance(o, dict):
            p = o.get("price")
            if isinstance(p, dict) and "input" in p and "output" in p:
                return p
            for v in o.values():
                walk(v)
        return None

    # The catalog is keyed "provider/model" at some depth; index by the model
    # half so a row's bare model name finds it.
    def index(o):
        if isinstance(o, dict):
            for k, v in o.items():
                if isinstance(v, dict) and isinstance(v.get("price"), dict):
                    p = v["price"]
                    name = k.split("/")[-1]
                    out.setdefault(name, (p.get("input", 0), p.get("cacheRead", p.get("input", 0)), p.get("output", 0)))
                index(v)

    index(catalog)
    return out


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--last", type=int, default=0, help="the last N sessions instead of today's")
    ap.add_argument("--since", default="", help="ISO time; sessions with a request at or after it")
    args = ap.parse_args()

    root = data_root()
    db = sqlite3.connect(os.path.join(root, "aetox.db"))
    price = prices(root)

    where, params = "", []
    if args.since:
        where, params = "where time >= ?", [args.since]
    elif not args.last:
        where, params = "where time >= ?", [dt.date.today().isoformat()]
    sessions = [r[0] for r in db.execute(
        f"select session_id from token_usage {where} group by session_id order by max(time) desc", params)]
    if args.last:
        sessions = sessions[: args.last]
    if not sessions:
        print("no usage rows in that range")
        return 0

    print(f"{'session':<22} {'model':<18} {'req':>4} {'prompt':>9} {'cached':>9} {'uncached':>9} {'output':>7} {'cache%':>6} {'USD':>8}")
    total_usd = 0.0
    for sid in sessions:
        rows = db.execute(
            "select model, prompt_tokens, coalesce(cached_prompt_tokens,0), completion_tokens, min(time), max(time) "
            "from token_usage where session_id = ? group by model", (sid,)).fetchall()
        # Per model within the session, since a session can switch models.
        for model, _, _, _, t0, t1 in rows:
            per = db.execute(
                "select count(*), sum(prompt_tokens), sum(coalesce(cached_prompt_tokens,0)), sum(completion_tokens) "
                "from token_usage where session_id = ? and model = ?", (sid, model)).fetchone()
            n, prompt, cached, output = per
            uncached = prompt - cached
            usd = ""
            if model in price:
                pin, pcache, pout = price[model]
                cost = (uncached * pin + cached * pcache + output * pout) / 1_000_000
                total_usd += cost
                usd = f"{cost:8.4f}"
            pct = f"{cached * 100 // max(prompt, 1):5d}%"
            print(f"{sid:<22} {model:<18} {n:>4} {prompt:>9,} {cached:>9,} {uncached:>9,} {output:>7,} {pct:>6} {usd:>8}   {t0[11:16]}-{t1[11:16]}")
    print(f"\ntotal {total_usd:.4f} USD across {len(sessions)} sessions (priced models only)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
