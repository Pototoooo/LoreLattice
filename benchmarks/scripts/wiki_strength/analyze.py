#!/usr/bin/env python3
"""Aggregate semantic and navigation metrics for the wiki-strength pilot."""

from __future__ import annotations

import argparse
import csv
import json
import statistics
from collections import Counter
from pathlib import Path


def percentile(values: list[float], p: float) -> float:
    values = sorted(values)
    position = (len(values) - 1) * p
    low = int(position)
    high = min(low + 1, len(values) - 1)
    return values[low] + (values[high] - values[low]) * (position - low)


def mean(rows: list[dict], key: str) -> float | None:
    values = [row[key] for row in rows if row.get(key) is not None]
    return statistics.mean(values) if values else None


def aggregate(rows: list[dict]) -> dict:
    latencies = [row["latency_ms"] for row in rows]
    return {
        "runs": len(rows),
        "execution_complete_rate": mean(rows, "execution_complete"),
        "semantic_success_rate": mean(rows, "semantic_success"),
        "mean_evidence_term_recall": mean(rows, "evidence_term_recall"),
        "tool_error_free_rate": mean(rows, "tool_error_free"),
        "navigation_not_found_error_rate": mean(rows, "navigation_not_found_error"),
        "anchor_page_hit_rate": mean(rows, "anchor_page_hit"),
        "mean_wiki_page_recall": mean(rows, "wiki_page_recall"),
        "navigation_success_rate": mean(rows, "navigation_success"),
        "duplicate_free_rate": statistics.mean(row["duplicate_final_paragraphs"] == 0 for row in rows),
        "mean_tool_calls": statistics.mean(row["tool_call_count"] for row in rows),
        "rerank_path_call_rate": statistics.mean("knowledge_search" in row["tools"] for row in rows),
        "latency_ms": {
            "mean": statistics.mean(latencies), "p50": percentile(latencies, .5),
            "p95": percentile(latencies, .95), "max": max(latencies),
        },
        "tool_distribution": dict(sorted(Counter(tool for row in rows for tool in row["tools"]).items())),
        "failed_cases": [row["case_id"] for row in rows if not row["semantic_success"]],
    }


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--input", type=Path, required=True)
    parser.add_argument("--output-dir", type=Path, required=True)
    args = parser.parse_args()
    rows = [json.loads(line) for line in args.input.read_text(encoding="utf-8").splitlines()]
    args.output_dir.mkdir(parents=True, exist_ok=True)
    modes = ["rag", "wiki", "hybrid"]
    valid = [row for row in rows if row["exit_status"] == 0]
    by_mode = {mode: [row for row in valid if row["mode"] == mode] for mode in modes}
    summary = {
        "protocol": {
            "dataset": "wiki_strength_v1", "cases": len({row["case_id"] for row in rows}),
            "raw_runs": len(rows), "valid_runs": len(valid), "modes": modes,
            "categories": sorted({row["category"] for row in rows}),
            "semantic_success_definition": "complete answer + all frozen term groups + no duplicate final paragraph",
            "navigation_success_definition": "semantic success + anchor wiki page read + no tool error",
            "order": "Latin-square interleaved", "workers": 1,
        },
        "modes": {mode: aggregate(by_mode[mode]) for mode in modes},
        "category": {
            category: {mode: aggregate([row for row in by_mode[mode] if row["category"] == category]) for mode in modes}
            for category in sorted({row["category"] for row in rows})
        },
        "environment_failures": [
            {"case_id": row["case_id"], "mode": row["mode"], "exit_status": row["exit_status"], "stderr": row["stderr"]}
            for row in rows if row["exit_status"] != 0
        ],
        "delta_vs_rag": {},
    }
    rag = summary["modes"]["rag"]
    for mode in ("wiki", "hybrid"):
        current = summary["modes"][mode]
        summary["delta_vs_rag"][mode] = {
            "semantic_success_percentage_points": (current["semantic_success_rate"] - rag["semantic_success_rate"]) * 100,
            "evidence_term_recall_percentage_points": (current["mean_evidence_term_recall"] - rag["mean_evidence_term_recall"]) * 100,
            "tool_error_free_percentage_points": (current["tool_error_free_rate"] - rag["tool_error_free_rate"]) * 100,
            "p95_latency_change_percent": (current["latency_ms"]["p95"] / rag["latency_ms"]["p95"] - 1) * 100,
            "mean_tool_calls_change_percent": (current["mean_tool_calls"] / rag["mean_tool_calls"] - 1) * 100,
        }
    (args.output_dir / "SUMMARY.json").write_text(json.dumps(summary, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    with (args.output_dir / "METRICS.csv").open("w", newline="", encoding="utf-8") as handle:
        writer = csv.writer(handle)
        writer.writerow(["mode", "runs", "semantic_success", "term_recall", "tool_error_free", "navigation_not_found_error", "anchor_hit", "page_recall", "navigation_success", "p50_ms", "p95_ms", "mean_tool_calls", "rerank_path"])
        for mode in modes:
            item = summary["modes"][mode]
            writer.writerow([mode, item["runs"], item["semantic_success_rate"], item["mean_evidence_term_recall"], item["tool_error_free_rate"], item["navigation_not_found_error_rate"], item["anchor_page_hit_rate"], item["mean_wiki_page_recall"], item["navigation_success_rate"], item["latency_ms"]["p50"], item["latency_ms"]["p95"], item["mean_tool_calls"], item["rerank_path_call_rate"]])
    print(f"ANALYZE_RESULT=PASS rows={len(rows)} valid={len(valid)} output={args.output_dir / 'SUMMARY.json'}")


if __name__ == "__main__":
    main()
