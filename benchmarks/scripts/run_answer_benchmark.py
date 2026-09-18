#!/usr/bin/env python3
"""Run 120 fresh-session KnowledgeQA turns and record strict, auditable signals."""

import argparse
import json
import math
import os
import re
import statistics
import subprocess
import time
from concurrent.futures import ThreadPoolExecutor, as_completed
from pathlib import Path


def percentile(values, q):
    values = sorted(values)
    if not values:
        return None
    pos = (len(values) - 1) * q
    lo, hi = math.floor(pos), math.ceil(pos)
    return values[lo] if lo == hi else values[lo] * (hi - pos) + values[hi] * (pos - lo)


def normalize(value):
    return re.sub(r"[^0-9A-Za-z\u4e00-\u9fff.%≥≤]+", "", value).lower()


def assertions(expected):
    parts = re.split(r"[，,；;、>|]|(?:并且)|(?:以及)", expected)
    return [normalize(part) for part in parts if len(normalize(part)) >= 2]


def run_one(binary, env, kb_id, row):
    started = time.perf_counter()
    command = [binary, "--format", "json", "chat", row["question"], "--kb", kb_id]
    try:
        proc = subprocess.run(command, env=env, capture_output=True, text=True, timeout=120)
    except subprocess.TimeoutExpired as exc:
        return {"id": row["id"], "fact_id": row["fact_id"], "category": row["category"],
                "variant": row["variant"], "question": row["question"], "exit_status": 124,
                "latency_ms": round((time.perf_counter() - started) * 1000, 3), "error": "timeout"}
    latency = (time.perf_counter() - started) * 1000
    if proc.returncode != 0:
        return {"id": row["id"], "fact_id": row["fact_id"], "category": row["category"],
                "variant": row["variant"], "question": row["question"], "exit_status": proc.returncode,
                "latency_ms": round(latency, 3), "error": proc.stderr[-1000:]}
    payload = json.loads(proc.stdout)["data"]
    events = payload.get("events") or []
    answer = "".join(event.get("content", "") for event in events if event.get("response_type") == "answer")
    citation_pairs = re.findall(r'<kb\s+doc="([^"]+)"\s+chunk_id="([^"]+)"', answer)
    cited_docs = list(dict.fromkeys(doc for doc, _ in citation_pairs))
    expected_assertions = assertions(row["expected_answer"])
    normalized_answer = normalize(answer)
    passed_assertions = [item for item in expected_assertions if item in normalized_answer]
    source_set = set(row["source_docs"])
    return {
        "id": row["id"], "fact_id": row["fact_id"], "category": row["category"],
        "variant": row["variant"], "question": row["question"], "expected_answer": row["expected_answer"],
        "source_docs": row["source_docs"], "exit_status": 0, "latency_ms": round(latency, 3), "error": "",
        "session_id": payload.get("session_id"), "assistant_message_id": payload.get("assistant_message_id"),
        "answer": answer, "answer_chars": len(answer), "citation_count": len(citation_pairs),
        "cited_docs": cited_docs, "has_citation": bool(citation_pairs),
        "relevant_citation": bool(source_set.intersection(cited_docs)),
        "all_sources_cited": source_set.issubset(set(cited_docs)),
        "assertions": expected_assertions, "passed_assertions": passed_assertions,
        "assertion_recall": len(passed_assertions) / len(expected_assertions) if expected_assertions else 1,
        "strict_assertion_pass": len(passed_assertions) == len(expected_assertions),
        "answer_event_count": sum(1 for event in events if event.get("response_type") == "answer" and event.get("content")),
    }


def aggregate(results):
    valid = [row for row in results if not row.get("error")]
    latencies = [row["latency_ms"] for row in valid]
    return {
        "queries": len(results), "successful": len(valid), "errors": len(results)-len(valid),
        "strict_assertion_pass_rate": statistics.mean(row["strict_assertion_pass"] for row in valid) if valid else None,
        "mean_assertion_recall": statistics.mean(row["assertion_recall"] for row in valid) if valid else None,
        "citation_presence_rate": statistics.mean(row["has_citation"] for row in valid) if valid else None,
        "relevant_citation_rate": statistics.mean(row["relevant_citation"] for row in valid) if valid else None,
        "all_sources_cited_rate": statistics.mean(row["all_sources_cited"] for row in valid) if valid else None,
        "latency_ms": {"mean": statistics.mean(latencies) if latencies else None,
                       "p50": percentile(latencies,.5), "p95": percentile(latencies,.95),
                       "p99": percentile(latencies,.99), "max": max(latencies) if latencies else None},
    }


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--binary", required=True)
    parser.add_argument("--token-file", required=True)
    parser.add_argument("--kb-id", required=True)
    parser.add_argument("--base-url", default="http://localhost:8080")
    parser.add_argument("--workers", type=int, default=4)
    parser.add_argument("--ids-file")
    parser.add_argument("--output-dir", required=True)
    args = parser.parse_args()
    root = Path(__file__).resolve().parents[2]
    rows = [json.loads(line) for line in (root/"benchmarks/datasets/enterprise_knowledge_v1/queries.jsonl").read_text().splitlines()]
    if args.ids_file:
        wanted = {line.strip() for line in Path(args.ids_file).read_text().splitlines() if line.strip()}
        rows = [row for row in rows if row["id"] in wanted]
    env = os.environ.copy()
    env.update({"LORELATTICE_TOKEN": Path(args.token_file).read_text().strip(),
                "LORELATTICE_HOST": args.base_url, "LORELATTICE_LOG_LEVEL": "error"})
    output = Path(args.output_dir); output.mkdir(parents=True, exist_ok=True)
    started = time.perf_counter(); results = []
    with ThreadPoolExecutor(max_workers=args.workers) as pool:
        futures = [pool.submit(run_one, args.binary, env, args.kb_id, row) for row in rows]
        for future in as_completed(futures):
            results.append(future.result())
    results.sort(key=lambda row: row["id"])
    with (output/"answer_raw.jsonl").open("w",encoding="utf-8") as handle:
        for row in results: handle.write(json.dumps(row,ensure_ascii=False)+"\n")
    summary = {"overall": aggregate(results)}
    for category in ("single","multi"):
        summary[category] = aggregate([row for row in results if row["category"]==category])
    for variant in ("canonical","paraphrase"):
        summary[variant] = aggregate([row for row in results if row["variant"]==variant])
    summary["run"] = {"workers":args.workers,"wall_seconds":time.perf_counter()-started,
                      "queries":len(rows),"independent_facts":len({row['fact_id'] for row in rows}),
                      "judge":"deterministic assertions"}
    (output/"answer_summary.json").write_text(json.dumps(summary,ensure_ascii=False,indent=2)+"\n")
    print("ANSWER_RESULT",json.dumps(summary,ensure_ascii=False,sort_keys=True))


if __name__ == "__main__":
    main()
