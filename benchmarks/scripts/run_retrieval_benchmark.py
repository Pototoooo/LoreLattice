#!/usr/bin/env python3
"""Run source-document and evidence retrieval metrics against LoreLattice."""

import argparse
import json
import math
import re
import statistics
import time
import urllib.error
import urllib.request
from collections import defaultdict
from concurrent.futures import ThreadPoolExecutor, as_completed
from pathlib import Path


def percentile(values, q):
    if not values:
        return None
    ordered = sorted(values)
    index = (len(ordered) - 1) * q
    low, high = math.floor(index), math.ceil(index)
    if low == high:
        return ordered[low]
    return ordered[low] * (high - index) + ordered[high] * (index - low)


def normalize(value):
    return re.sub(r"\s+", "", value).replace(",", "").replace("，", "")


def request_one(base_url, kb_id, token, row, mode, limit):
    body = {
        "query_text": row["question"],
        "vector_threshold": 0,
        "keyword_threshold": 0,
        "match_count": limit,
        "disable_keywords_match": mode == "vector",
        "disable_vector_match": mode == "keyword",
    }
    request = urllib.request.Request(
        f"{base_url}/api/v1/knowledge-bases/{kb_id}/hybrid-search",
        data=json.dumps(body, ensure_ascii=False).encode(),
        headers={"Authorization": f"Bearer {token}", "Content-Type": "application/json"},
        method="POST",
    )
    started = time.perf_counter()
    last_error = ""
    for attempt in range(1, 3):
        try:
            with urllib.request.urlopen(request, timeout=30) as response:
                payload = json.load(response)
                latency_ms = (time.perf_counter() - started) * 1000
                chunks = (payload.get("data") or [])[:limit]
                return score_result(row, mode, chunks, latency_ms, response.status, "", attempt)
        except (urllib.error.URLError, urllib.error.HTTPError, TimeoutError) as exc:
            last_error = repr(exc)
            if attempt < 2:
                time.sleep(0.5)
    return score_result(row, mode, [], (time.perf_counter() - started) * 1000, 0, last_error, 2)


def score_result(row, mode, chunks, latency_ms, status, error, attempts):
    relevant = set(row["source_docs"])
    titles = [chunk.get("knowledge_filename") or chunk.get("knowledge_title") or "" for chunk in chunks]
    contents = [chunk.get("content") or "" for chunk in chunks]
    first_rank = next((index + 1 for index, title in enumerate(titles) if title in relevant), None)
    evidence = normalize("\n".join(contents))
    found_terms = [term for term in row["required_evidence_terms"] if normalize(term) in evidence]
    result = {
        "id": row["id"], "fact_id": row["fact_id"], "variant": row["variant"],
        "category": row["category"], "mode": mode, "question": row["question"],
        "source_docs": row["source_docs"], "status": status, "error": error,
        "attempts": attempts, "latency_ms": round(latency_ms, 3),
        "returned": len(chunks), "titles": titles,
        "scores": [chunk.get("score") for chunk in chunks],
        "first_relevant_rank": first_rank,
        "all_sources_at_8": relevant.issubset(set(titles)),
        "evidence_terms_found": found_terms,
        "evidence_term_recall": len(found_terms) / len(row["required_evidence_terms"]),
        "chunks": [{"id": c.get("id"), "title": t, "content": content, "score": c.get("score")}
                   for c, t, content in zip(chunks, titles, contents)],
    }
    for k in (1, 3, 5, 8):
        top = titles[:k]
        result[f"hit_at_{k}"] = any(title in relevant for title in top)
        result[f"source_recall_at_{k}"] = len(relevant.intersection(top)) / len(relevant)
    return result


def aggregate(results):
    valid = [r for r in results if not r["error"]]
    latencies = [r["latency_ms"] for r in valid]
    summary = {
        "queries": len(results), "successful": len(valid), "errors": len(results) - len(valid),
        "latency_ms": {"mean": statistics.mean(latencies) if latencies else None,
                       "p50": percentile(latencies, .50), "p95": percentile(latencies, .95),
                       "p99": percentile(latencies, .99), "max": max(latencies) if latencies else None},
    }
    if valid:
        for k in (1, 3, 5, 8):
            summary[f"hit_at_{k}"] = statistics.mean(r[f"hit_at_{k}"] for r in valid)
            summary[f"source_recall_at_{k}"] = statistics.mean(r[f"source_recall_at_{k}"] for r in valid)
        summary["mrr_at_8"] = statistics.mean(1 / r["first_relevant_rank"] if r["first_relevant_rank"] else 0 for r in valid)
        summary["all_sources_at_8"] = statistics.mean(r["all_sources_at_8"] for r in valid)
        summary["evidence_term_recall_at_8"] = statistics.mean(r["evidence_term_recall"] for r in valid)
    return summary


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--token-file", required=True)
    parser.add_argument("--kb-id", required=True)
    parser.add_argument("--base-url", default="http://localhost:8080")
    parser.add_argument("--workers", type=int, default=4)
    parser.add_argument("--output-dir", required=True)
    args = parser.parse_args()

    root = Path(__file__).resolve().parents[2]
    rows = [json.loads(line) for line in (root / "benchmarks/datasets/enterprise_knowledge_v1/queries.jsonl").read_text().splitlines()]
    token = Path(args.token_file).read_text().strip()
    output = Path(args.output_dir)
    output.mkdir(parents=True, exist_ok=True)
    all_results = []
    started = time.perf_counter()
    for mode in ("hybrid", "vector", "keyword"):
        mode_results = []
        with ThreadPoolExecutor(max_workers=args.workers) as pool:
            futures = [pool.submit(request_one, args.base_url.rstrip("/"), args.kb_id, token, row, mode, 8) for row in rows]
            for future in as_completed(futures):
                mode_results.append(future.result())
        mode_results.sort(key=lambda item: item["id"])
        all_results.extend(mode_results)
        print("MODE_RESULT", mode, json.dumps(aggregate(mode_results), ensure_ascii=False, sort_keys=True))

    raw_path = output / "retrieval_raw.jsonl"
    with raw_path.open("w", encoding="utf-8") as handle:
        for item in all_results:
            handle.write(json.dumps(item, ensure_ascii=False) + "\n")

    summaries = {}
    for mode in ("hybrid", "vector", "keyword"):
        subset = [r for r in all_results if r["mode"] == mode]
        summaries[mode] = {"overall": aggregate(subset)}
        for category in ("single", "multi"):
            summaries[mode][category] = aggregate([r for r in subset if r["category"] == category])
        for variant in ("canonical", "paraphrase"):
            summaries[mode][variant] = aggregate([r for r in subset if r["variant"] == variant])
    summaries["run"] = {"workers": args.workers, "top_k": 8, "wall_seconds": time.perf_counter() - started,
                        "kb_id": args.kb_id, "dataset_queries": len(rows), "independent_facts": 60}
    (output / "retrieval_summary.json").write_text(json.dumps(summaries, ensure_ascii=False, indent=2) + "\n")
    print("BENCHMARK_RESULT", json.dumps(summaries["run"], ensure_ascii=False, sort_keys=True))


if __name__ == "__main__":
    main()
