#!/usr/bin/env python3
"""Run the three Agent surfaces on a frozen wiki-strength dataset."""

from __future__ import annotations

import argparse
import json
import os
import re
import statistics
import subprocess
import time
from pathlib import Path


def normalize(value: str) -> str:
    return re.sub(r"[\s,，。；;：:、*`#_\-（）()\[\]{}]", "", value or "").lower()


def final_answer(events: list[dict]) -> tuple[str, int]:
    groups: dict[str, list[str]] = {}
    order: list[str] = []
    for event in events:
        if event.get("response_type") != "answer":
            continue
        data = event.get("data") or {}
        event_id = data.get("event_id", "unknown")
        if event_id not in groups:
            groups[event_id] = []
            order.append(event_id)
        if event.get("content"):
            groups[event_id].append(event["content"])
    return ("".join(groups[order[-1]]) if order else "", len(order))


def project(mode: str, case: dict, payload: dict, latency_ms: float, exit_status: int, stderr: str) -> dict:
    events = payload.get("data", {}).get("events", []) if isinstance(payload, dict) else []
    answer, answer_stages = final_answer(events)
    tools: list[str] = []
    tool_errors: list[str] = []
    read_pages: set[str] = set()
    complete = False
    server_steps = None
    server_duration_ms = None
    for event in events:
        data = event.get("data") or {}
        response_type = event.get("response_type")
        if response_type == "tool_call" and data.get("arguments") is not None:
            tools.append(data.get("tool_name", "unknown"))
        elif response_type == "tool_result":
            if data.get("error"):
                tool_errors.append(str(data["error"]))
            if data.get("tool_name") == "wiki_read_page":
                output = str(data.get("output") or "")
                read_pages.update(re.findall(r"(?s)<wiki_page>.*?<metadata>.*?<link>\[\[([^]|]+)", output))
        elif response_type in {"error", "agent_error"}:
            tool_errors.append(str(data.get("error") or event.get("content") or "agent_error"))
        elif response_type == "complete":
            complete = True
            server_steps = data.get("total_steps")
            server_duration_ms = data.get("total_duration_ms")

    normalized_answer = normalize(answer)
    groups = case["required_term_groups"]
    group_hits = [any(normalize(term) in normalized_answer for term in group) for group in groups]
    term_recall = statistics.mean(group_hits) if group_hits else None
    expected_pages = set(case["expected_wiki_pages"])
    anchors = set(case["anchor_pages"])
    page_recall = len(expected_pages & read_pages) / len(expected_pages) if expected_pages else None
    anchor_hit = bool(anchors & read_pages) if mode in {"wiki", "hybrid"} else None
    paragraphs = [normalize(item) for item in re.split(r"\n\s*\n", answer) if normalize(item)]
    duplicate_paragraphs = len(paragraphs) - len(set(paragraphs))
    execution_complete = exit_status == 0 and complete and bool(answer.strip())
    semantic_success = execution_complete and term_recall == 1.0 and duplicate_paragraphs == 0
    navigation_success = None
    if mode in {"wiki", "hybrid"}:
        navigation_success = semantic_success and bool(anchor_hit) and not tool_errors

    return {
        "case_id": case["id"], "mode": mode, "category": case["category"], "question": case["question"],
        "exit_status": exit_status, "latency_ms": round(latency_ms, 3), "server_duration_ms": server_duration_ms,
        "server_steps": server_steps, "event_count": len(events), "answer_stage_count": answer_stages,
        "answer_chars": len(answer), "tool_call_count": len(tools), "tools": tools,
        "tool_error_count": len(tool_errors), "tool_errors": tool_errors,
        "navigation_not_found_error": any("Wiki page" in error and "not found" in error for error in tool_errors),
        "read_wiki_pages": sorted(read_pages), "expected_wiki_pages": sorted(expected_pages),
        "anchor_pages": sorted(anchors), "anchor_page_hit": anchor_hit, "wiki_page_recall": page_recall if mode in {"wiki", "hybrid"} else None,
        "required_term_groups": groups, "term_group_hits": group_hits, "evidence_term_recall": term_recall,
        "duplicate_final_paragraphs": duplicate_paragraphs, "execution_complete": execution_complete,
        "tool_error_free": not tool_errors, "semantic_success": semantic_success,
        "navigation_success": navigation_success, "final_answer": answer, "stderr": stderr, "raw": payload,
    }


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--cli", type=Path, required=True)
    parser.add_argument("--token-file", type=Path, required=True)
    parser.add_argument("--agents", type=Path, required=True, help="TSV: mode<TAB>agent_id")
    parser.add_argument("--dataset", type=Path, required=True)
    parser.add_argument("--output-dir", type=Path, required=True)
    parser.add_argument("--base-url", default="http://localhost:8080")
    parser.add_argument("--timeout", type=int, default=150)
    parser.add_argument("--cooldown", type=float, default=1.0)
    args = parser.parse_args()
    args.output_dir.mkdir(parents=True, exist_ok=True)
    agents = dict(line.split("\t", 1) for line in args.agents.read_text().splitlines() if line.strip())
    modes = ["rag", "wiki", "hybrid"]
    cases = [json.loads(line) for line in args.dataset.read_text(encoding="utf-8").splitlines()]
    output = args.output_dir / "results.jsonl"
    done = set()
    if output.exists():
        done = {(row["case_id"], row["mode"]) for row in (json.loads(line) for line in output.read_text().splitlines())}
    env = os.environ.copy()
    env.update({"LORELATTICE_TOKEN": args.token_file.read_text().strip(), "LORELATTICE_HOST": args.base_url, "LORELATTICE_LOG_LEVEL": "error"})
    total = len(cases) * len(modes)
    completed = len(done)
    for index, case in enumerate(cases):
        order = modes[index % 3:] + modes[:index % 3]
        for mode in order:
            if (case["id"], mode) in done:
                continue
            command = [str(args.cli), "session", "ask", "--agent", agents[mode], "--reference", "--verbose", "--format", "json", case["question"]]
            started = time.perf_counter()
            try:
                proc = subprocess.run(command, env=env, text=True, capture_output=True, timeout=args.timeout)
                exit_status, stdout, stderr = proc.returncode, proc.stdout, proc.stderr
            except subprocess.TimeoutExpired as exc:
                exit_status, stdout, stderr = 124, exc.stdout or "", (exc.stderr or "") + f"\nTIMEOUT after {args.timeout}s"
            latency_ms = (time.perf_counter() - started) * 1000
            try:
                payload = json.loads(stdout)
            except (json.JSONDecodeError, TypeError):
                payload = {"ok": False, "parse_error": stdout}
            row = project(mode, case, payload, latency_ms, exit_status, stderr)
            with output.open("a", encoding="utf-8") as handle:
                handle.write(json.dumps(row, ensure_ascii=False) + "\n")
            completed += 1
            print(
                f"RUN_RESULT {completed}/{total} case={row['case_id']} mode={mode} exit={exit_status} "
                f"semantic={row['semantic_success']} navigation={row['navigation_success']} terms={row['evidence_term_recall']} "
                f"anchor={row['anchor_page_hit']} errors={row['tool_error_count']} latency_ms={latency_ms:.1f}", flush=True,
            )
            time.sleep(args.cooldown)
    print(f"RUN_COMPLETE rows={completed} expected={total}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
