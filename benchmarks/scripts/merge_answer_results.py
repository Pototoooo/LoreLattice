#!/usr/bin/env python3
"""Merge first-pass and controlled retry answer runs into an auditable quality set."""

from __future__ import annotations

import argparse
import csv
import json
from collections import Counter
from pathlib import Path


def read_jsonl(path: Path) -> list[dict]:
    return [json.loads(line) for line in path.read_text(encoding="utf-8").splitlines() if line.strip()]


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--answer-dir", type=Path, required=True)
    parser.add_argument("--adjudication", type=Path, required=True)
    args = parser.parse_args()

    merged = {row["id"]: row for row in read_jsonl(args.answer_dir / "answer_raw.jsonl")}
    selected_from = {query_id: "initial_workers4" for query_id in merged}
    for relative, source in (
        ("rerun-single/answer_raw.jsonl", "controlled_retry_workers1"),
        ("retry-final/answer_raw.jsonl", "final_single_retry"),
    ):
        for row in read_jsonl(args.answer_dir / relative):
            if row["exit_status"] == 0 and row["has_citation"]:
                merged[row["id"]] = row
                selected_from[row["id"]] = source

    with args.adjudication.open(encoding="utf-8", newline="") as handle:
        reviews = {row["id"]: row for row in csv.DictReader(handle)}
    if set(merged) != set(reviews):
        raise SystemExit(f"review IDs differ: answers={len(merged)} reviews={len(reviews)}")

    output_dir = args.answer_dir / "final"
    output_dir.mkdir(parents=True, exist_ok=True)
    final_rows = []
    for query_id in sorted(merged):
        row = dict(merged[query_id])
        row["selected_from"] = selected_from[query_id]
        row["adjudication"] = reviews[query_id]
        final_rows.append(row)

    with (output_dir / "answer_final.jsonl").open("w", encoding="utf-8") as handle:
        for row in final_rows:
            handle.write(json.dumps(row, ensure_ascii=False) + "\n")

    status_counts = Counter(row["adjudication"]["overall_status"] for row in final_rows)
    dimension_counts = {
        dimension: Counter(row["adjudication"][dimension] for row in final_rows)
        for dimension in ("factual_completeness", "unsupported_content", "citation_quality")
    }
    count = len(final_rows)
    summary = {
        "dataset_queries": count,
        "independent_facts": len({row["fact_id"] for row in final_rows}),
        "selection": Counter(selected_from.values()),
        "overall_status_counts": status_counts,
        "exact_pass_rate": status_counts["pass"] / count,
        "pass_or_partial_rate": (status_counts["pass"] + status_counts["partial"]) / count,
        "dimension_counts": dimension_counts,
        "citation_presence_rate": sum(row["has_citation"] for row in final_rows) / count,
        "relevant_citation_rate": sum(row["relevant_citation"] for row in final_rows) / count,
        "all_declared_sources_cited_rate": sum(row["all_sources_cited"] for row in final_rows) / count,
        "review_method": "Codex answer-by-answer qualitative adjudication against frozen expected answers and source declarations; not an independent human blind review",
        "latency_note": "Mixed retry paths; use the individual run summaries for latency and concurrency claims.",
    }
    (output_dir / "answer_final_summary.json").write_text(
        json.dumps(summary, ensure_ascii=False, indent=2, default=dict) + "\n", encoding="utf-8"
    )
    print(json.dumps(summary, ensure_ascii=False, indent=2, default=dict))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
