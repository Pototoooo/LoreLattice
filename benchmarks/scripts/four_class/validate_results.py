#!/usr/bin/env python3
from __future__ import annotations

import argparse
import json
from collections import Counter
from pathlib import Path


MODES = ("rag", "wiki", "hybrid")
CATEGORIES = (
    "single_hop_factual",
    "local_synthesis",
    "multi_hop",
    "global_thematic",
)


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--input", type=Path, required=True)
    parser.add_argument("--dataset", type=Path, required=True)
    parser.add_argument("--repeats", type=int, default=2)
    args = parser.parse_args()

    rows = [json.loads(line) for line in args.input.read_text().splitlines() if line.strip()]
    cases = [json.loads(line) for line in args.dataset.read_text().splitlines() if line.strip()]
    expected = len(cases) * len(MODES) * args.repeats
    keys = [(row["case_id"], row["mode"], row["repeat"]) for row in rows]
    expected_per_cell = Counter((case["category"], mode) for case in cases for mode in MODES)
    expected_per_cell = Counter({key: value * args.repeats for key, value in expected_per_cell.items()})
    actual_per_cell = Counter((row["category"], row["mode"]) for row in rows)

    checks = {
        "row_count": len(rows) == expected,
        "unique_run_keys": len(keys) == len(set(keys)),
        "balanced_cells": actual_per_cell == expected_per_cell,
        "known_modes": set(row["mode"] for row in rows) == set(MODES),
        "known_categories": set(row["category"] for row in rows) == set(CATEGORIES),
        "all_exit_zero": all(row["exit_status"] == 0 for row in rows),
        "all_execution_complete": all(row["execution_complete"] for row in rows),
        "all_have_answers": all(row["answer_chars"] > 0 for row in rows),
        "all_have_usage": all(row["total_tokens"] > 0 for row in rows),
        "metric_ranges": all(
            0 <= row[name] <= 1
            for row in rows
            for name in ("fact_completeness", "retrieval_recall", "citation_recall")
        ),
        "rag_rerank_path_observed": any(row["rerank_path_used"] for row in rows if row["mode"] == "rag"),
        "hybrid_rerank_path_observed": any(row["rerank_path_used"] for row in rows if row["mode"] == "hybrid"),
    }
    failures = [name for name, passed in checks.items() if not passed]
    print(
        "QUALITY_RESULT=%s rows=%d expected=%d unique=%d exit_failures=%d incomplete=%d zero_usage=%d"
        % (
            "PASS" if not failures else "FAIL",
            len(rows),
            expected,
            len(set(keys)),
            sum(row["exit_status"] != 0 for row in rows),
            sum(not row["execution_complete"] for row in rows),
            sum(row["total_tokens"] <= 0 for row in rows),
        )
    )
    for name, passed in checks.items():
        print(f"CHECK {name}={'PASS' if passed else 'FAIL'}")
    print("CELL_COUNTS " + json.dumps(dict(sorted((f"{k[0]}/{k[1]}", v) for k, v in actual_per_cell.items()))))
    if failures:
        print("FAILURES " + ",".join(failures))
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
