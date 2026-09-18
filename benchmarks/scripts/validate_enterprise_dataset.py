#!/usr/bin/env python3
"""Verify IDs, grouping, and that every expected term exists in ground-truth docs."""

import json
import re
from collections import Counter, defaultdict
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
BASE = ROOT / "benchmarks/datasets/enterprise_knowledge_v1"
rows = [json.loads(line) for line in (BASE / "queries.jsonl").read_text().splitlines()]
assert len(rows) == 120
assert len({row["id"] for row in rows}) == 120

groups = defaultdict(list)
for row in rows:
    groups[row["fact_id"]].append(row)
assert len(groups) == 60 and all(len(items) == 2 for items in groups.values())

def normalize(value):
    return re.sub(r"\s+", "", value).replace(",", "").replace("，", "")

for row in rows:
    evidence = "\n".join((BASE / "source_docs" / name).read_text() for name in row["source_docs"])
    evidence = normalize(evidence)
    missing = [term for term in row["required_evidence_terms"] if normalize(term) not in evidence]
    assert not missing, f"{row['id']} missing={missing}"

print("VALIDATION_RESULT queries=120 facts=60 variants_per_fact=2 missing_evidence=0")
print("CATEGORY_COUNTS", dict(Counter(row["category"] for row in rows)))
