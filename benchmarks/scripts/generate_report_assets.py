#!/usr/bin/env python3
"""Create a machine-readable metric table and dependency-free SVG scorecard."""

from __future__ import annotations

import csv
import json
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
RESULTS = ROOT / "results/2026-08-25"
REPORT = ROOT / "reports/2026-08-25"


def load(relative: str) -> dict:
    return json.loads((RESULTS / relative).read_text(encoding="utf-8"))


def main() -> int:
    retrieval = load("retrieval/retrieval_summary.json")
    answer = load("answer/final/answer_final_summary.json")
    agent = load("agent/agent_summary.json")
    api = load("performance/api-rate-k6-summary.json")["metrics"]
    frontend = load("performance/frontend-k6-summary.json")["metrics"]
    metrics = [
        ("retrieval", "hybrid_hit_at_8", retrieval["hybrid"]["overall"]["hit_at_8"], "rate", "120 queries, top8"),
        ("retrieval", "hybrid_mrr_at_8", retrieval["hybrid"]["overall"]["mrr_at_8"], "rate", "120 queries, top8"),
        ("retrieval", "keyword_hit_at_8", retrieval["keyword"]["overall"]["hit_at_8"], "rate", "120 queries, top8"),
        ("answer", "adjudicated_exact_pass", answer["exact_pass_rate"], "rate", "Codex review, 120 queries"),
        ("answer", "relevant_citation", answer["relevant_citation_rate"], "rate", "controlled final set"),
        ("answer", "all_declared_sources_cited", answer["all_declared_sources_cited_rate"], "rate", "controlled final set"),
        ("agent", "repeat_expected_fact", agent["repeat_answer_accuracy"], "rate", "20 sequential repetitions"),
        ("agent", "repeat_duplicate_free", agent["repeat_duplicate_free_rate"], "rate", "20 sequential repetitions"),
        ("agent", "unknown_grounded_refusal", agent["unknown_grounded_refusal_rate"], "rate", "5 unknown queries"),
        ("performance", "api_completed_requests", api["iterations"]["values"]["count"], "count", "10/100/300 target RPS"),
        ("performance", "api_p95_ms", api["http_req_duration"]["values"]["p(95)"], "ms", "local authenticated read"),
        ("performance", "frontend_static_p95_ms", frontend["http_req_duration"]["values"]["p(95)"], "ms", "local static shell"),
    ]
    REPORT.mkdir(parents=True, exist_ok=True)
    with (REPORT / "METRICS.csv").open("w", encoding="utf-8", newline="") as handle:
        writer = csv.writer(handle)
        writer.writerow(["group", "metric", "value", "unit", "scope"])
        writer.writerows(metrics)

    bars = [
        ("Hybrid Hit@8", retrieval["hybrid"]["overall"]["hit_at_8"]),
        ("Hybrid MRR@8", retrieval["hybrid"]["overall"]["mrr_at_8"]),
        ("Answer exact", answer["exact_pass_rate"]),
        ("Relevant citation", answer["relevant_citation_rate"]),
        ("Agent 20x accurate", agent["repeat_answer_accuracy"]),
        ("Unknown refusal", agent["unknown_grounded_refusal_rate"]),
    ]
    width, height = 960, 470
    svg = [
        f'<svg xmlns="http://www.w3.org/2000/svg" width="{width}" height="{height}" viewBox="0 0 {width} {height}">',
        '<rect width="100%" height="100%" rx="24" fill="#111827"/>',
        '<text x="48" y="58" fill="#f9fafb" font-family="Arial,sans-serif" font-size="28" font-weight="700">LoreLattice benchmark scorecard</text>',
        '<text x="48" y="88" fill="#9ca3af" font-family="Arial,sans-serif" font-size="15">60 independent facts · 120 queries · Apple M1 Pro · 2026-08-25/26</text>',
    ]
    for index, (label, value) in enumerate(bars):
        y = 125 + index * 52
        svg += [
            f'<text x="48" y="{y + 18}" fill="#e5e7eb" font-family="Arial,sans-serif" font-size="16">{label}</text>',
            f'<rect x="260" y="{y}" width="600" height="24" rx="12" fill="#374151"/>',
            f'<rect x="260" y="{y}" width="{600 * value:.1f}" height="24" rx="12" fill="#10b981"/>',
            f'<text x="875" y="{y + 18}" fill="#f9fafb" font-family="Arial,sans-serif" font-size="16" text-anchor="end">{value * 100:.2f}%</text>',
        ]
    svg += [
        '<text x="48" y="445" fill="#9ca3af" font-family="Arial,sans-serif" font-size="13">Answer score uses Codex qualitative adjudication, not independent human blind review.</text>',
        '</svg>',
    ]
    (REPORT / "benchmark-scorecard.svg").write_text("\n".join(svg) + "\n", encoding="utf-8")
    print(REPORT / "METRICS.csv")
    print(REPORT / "benchmark-scorecard.svg")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
