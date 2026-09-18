#!/usr/bin/env python3
"""Exercise the deployed built-in Agent and retain every projected event."""

from __future__ import annotations

import argparse
import json
import os
import re
import statistics
import subprocess
import time
from collections import defaultdict
from pathlib import Path


def percentile(values: list[float], p: float) -> float:
    values = sorted(values)
    position = (len(values) - 1) * p
    low = int(position)
    high = min(low + 1, len(values) - 1)
    return values[low] + (values[high] - values[low]) * (position - low)


def project(payload: dict, expected: str, kind: str, latency_ms: float, exit_status: int, stderr: str) -> dict:
    events = payload.get("data", {}).get("events", [])
    answer_groups: dict[str, list[str]] = defaultdict(list)
    answer_order: list[str] = []
    tool_calls, tool_errors = {}, []
    lifecycle = None
    for event in events:
        response_type = event.get("response_type")
        data = event.get("data") or {}
        if response_type == "answer":
            event_id = data.get("event_id", "unknown")
            if event_id not in answer_groups:
                answer_order.append(event_id)
            answer_groups[event_id].append(event.get("content") or "")
        elif response_type == "tool_call" and data.get("arguments") is not None:
            tool_calls[data.get("tool_call_id")] = data.get("tool_name")
        elif response_type == "tool_result" and data.get("error"):
            tool_errors.append(data.get("error"))
        elif response_type == "agent_final":
            lifecycle = data
    final_answer = "".join(answer_groups[answer_order[-1]]) if answer_order else ""
    paragraphs = [re.sub(r"\s+", " ", p).strip() for p in re.split(r"\n\s*\n", final_answer) if p.strip()]
    duplicate_paragraphs = len(paragraphs) - len(set(paragraphs))
    return {
        "kind": kind,
        "exit_status": exit_status,
        "latency_ms": round(latency_ms, 3),
        "session_id": payload.get("data", {}).get("session_id", ""),
        "event_count": len(events),
        "answer_stage_count": len(answer_order),
        "tool_call_count": len(tool_calls),
        "tools": list(tool_calls.values()),
        "tool_error_count": len(tool_errors),
        "final_answer": final_answer,
        "expected_present": expected in final_answer,
        "duplicate_final_paragraphs": duplicate_paragraphs,
        "lifecycle": lifecycle,
        "stderr": stderr,
        "raw": payload,
    }


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--cli", type=Path, required=True)
    parser.add_argument("--token-file", type=Path, required=True)
    parser.add_argument("--output-dir", type=Path, required=True)
    parser.add_argument("--repeat", type=int, default=20)
    parser.add_argument("--unknown", type=int, default=5)
    args = parser.parse_args()
    args.output_dir.mkdir(parents=True, exist_ok=True)
    env = os.environ.copy()
    env.update({
        "LORELATTICE_TOKEN": args.token_file.read_text().strip(),
        "LORELATTICE_HOST": "http://localhost",
        "LORELATTICE_LOG_LEVEL": "error",
    })
    cases = [
        ("repeat", "截至2026年6月30日公司有多少名全职员工？", "286")
        for _ in range(args.repeat)
    ] + [
        ("unknown", f"知识库中Zephyr-99计划第{i + 1}阶段的正式预算是多少？", "")
        for i in range(args.unknown)
    ]
    results = []
    started = time.perf_counter()
    for index, (kind, query, expected) in enumerate(cases, 1):
        command = [str(args.cli), "session", "ask", "--agent", "builtin-smart-reasoning", "--reference", "--verbose", "--format", "json", query]
        begin = time.perf_counter()
        proc = subprocess.run(command, env=env, text=True, capture_output=True, timeout=130)
        latency_ms = (time.perf_counter() - begin) * 1000
        try:
            payload = json.loads(proc.stdout)
        except json.JSONDecodeError:
            payload = {"ok": False, "parse_error": proc.stdout}
        row = project(payload, expected, kind, latency_ms, proc.returncode, proc.stderr)
        row["index"] = index
        row["query"] = query
        results.append(row)
        with (args.output_dir / "agent_raw.jsonl").open("a", encoding="utf-8") as handle:
            handle.write(json.dumps(row, ensure_ascii=False) + "\n")
        print(f"{index}/{len(cases)} kind={kind} exit={proc.returncode} tools={row['tool_call_count']} expected={row['expected_present']} latency_ms={latency_ms:.1f}", flush=True)
    repeat_rows = [row for row in results if row["kind"] == "repeat"]
    unknown_rows = [row for row in results if row["kind"] == "unknown"]
    latencies = [row["latency_ms"] for row in results]
    unknown_grounded = [
        row for row in unknown_rows
        if re.search(r"未找到|没有找到|未检索到|无法确认|暂无|不存在|没有.*信息|未.*提及", row["final_answer"])
    ]
    summary = {
        "agent_id": "builtin-smart-reasoning",
        "runs": len(results),
        "repeat_runs": len(repeat_rows),
        "unknown_runs": len(unknown_rows),
        "process_success_rate": sum(row["exit_status"] == 0 for row in results) / len(results),
        "repeat_answer_accuracy": sum(row["expected_present"] for row in repeat_rows) / len(repeat_rows),
        "repeat_duplicate_free_rate": sum(row["duplicate_final_paragraphs"] == 0 for row in repeat_rows) / len(repeat_rows),
        "unknown_grounded_refusal_rate": len(unknown_grounded) / len(unknown_rows),
        "tool_result_success_rate": sum(row["tool_error_count"] == 0 for row in results) / len(results),
        "tool_calls_total": sum(row["tool_call_count"] for row in results),
        "tool_calls_max_per_run": max(row["tool_call_count"] for row in results),
        "latency_ms": {
            "mean": statistics.mean(latencies),
            "p50": percentile(latencies, 0.50),
            "p95": percentile(latencies, 0.95),
            "max": max(latencies),
        },
        "wall_seconds": time.perf_counter() - started,
    }
    (args.output_dir / "agent_summary.json").write_text(json.dumps(summary, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(json.dumps(summary, ensure_ascii=False, indent=2))
    return 0 if summary["process_success_rate"] == 1 else 1


if __name__ == "__main__":
    raise SystemExit(main())
