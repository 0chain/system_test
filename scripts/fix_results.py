#!/usr/bin/env python3
"""Parse test logs with multiple runs and deduplicate by test name (best result wins).

Reads from:
  /tmp/test_SUITE.log           — live log from current run (streaming)
  test_results/SUITE_runN.txt   — historical run files (all runs)
  test_results/runs_meta.json   — metadata: suite, run#, mode, time, pass/fail counts

Generates a single combined HTML page (summary + subtests) at RESULTS_HTML.
Also writes a redirect at SUBTESTS_HTML so the old URL still works.

Usage:
    python3 fix_results.py          # Generate once
    python3 fix_results.py --loop   # Regenerate every 5 seconds
"""
import re, os, sys, time, html, json, glob
from datetime import datetime
from collections import OrderedDict

SUITES = ["sdk", "api", "cli", "zs3", "mc", "rclone"]
RESULTS_HTML  = "/var/log/0chain/test_results.html"
SUBTESTS_HTML = "/var/log/0chain/test_subtests.html"

# Where run_tests.sh writes per-run output files
SCRIPT_DIR   = os.path.dirname(os.path.abspath(__file__))
RESULTS_DIR  = os.path.join(SCRIPT_DIR, "..", "test_results")
RUNS_META    = os.path.join(RESULTS_DIR, "runs_meta.json")

# Priority: PASS > SKIP > FAIL
PRIORITY = {"PASS": 3, "SKIP": 2, "FAIL": 1}

# Mode labels for display
MODE_LABEL = {
    "initial":      "Initial run",
    "full_retest":  "Full re-test",
    "rerun_failed": "Retry (failed)",
}


# ── run metadata ──────────────────────────────────────────────────────────────

def parse_runs_meta():
    """Load runs_meta.json; return list of run entries (newest last)."""
    if not os.path.exists(RUNS_META):
        return []
    try:
        with open(RUNS_META) as f:
            data = json.load(f)
        return data if isinstance(data, list) else []
    except Exception:
        return []


def last_full_retest_index(runs):
    """Return index of the last full_retest entry, or -1 if none."""
    for i in range(len(runs) - 1, -1, -1):
        if runs[i].get("mode") == "full_retest":
            return i
    return -1


# ── log parsing ───────────────────────────────────────────────────────────────

def _parse_single_log(filepath, run_num=None):
    """
    Parse one Go test log file.
    Returns {test_name: (status, dur, output_lines, run_num)} — best-result-wins.
    """
    results = OrderedDict()
    current_outputs = {}
    suite_done = False
    if not os.path.exists(filepath):
        return results, False

    with open(filepath, "r") as f:
        for line in f:
            line_clean = re.sub(r"\x1b\[[0-9;]*m", "", line.rstrip("\n"))

            m_run = re.match(r"^\s*=== RUN\s+(\S+)", line_clean)
            if m_run:
                current_outputs[m_run.group(1).strip()] = []
                continue

            m = re.match(r"^\s*--- (PASS|FAIL|SKIP): (.+?) \((\d+\.\d+s)\)", line_clean)
            if m:
                status, name, dur = m.group(1), m.group(2), m.group(3)
                output = current_outputs.get(name, [])
                pri = PRIORITY.get(status, 0)
                existing = results.get(name)
                if existing is None or pri > PRIORITY.get(existing[0], 0):
                    results[name] = (status, dur, output, run_num)
                current_outputs.pop(name, None)
                continue

            if re.match(r"^(ok|FAIL)\s+", line_clean):
                suite_done = True

            m_log = re.match(r"^\s+(\S+\.go:\d+:)\s*(.*)", line_clean)
            if m_log:
                for tn in reversed(list(current_outputs)):
                    current_outputs[tn].append(line_clean.strip())
                    break
            elif any(line_clean.strip().startswith(x) for x in
                     ("Error Trace:", "Error:", "Messages:", "Expected",
                      "expected", "received", "Received")):
                for tn in reversed(list(current_outputs)):
                    current_outputs[tn].append(line_clean.strip())
                    break

    return results, suite_done


def parse_consolidated_for_suite(suite, since_run=None):
    """
    Merge results from run files for a suite (best result per test).
    For the live log, also checks /tmp/test_SUITE.log.

    Args:
        suite     — suite name
        since_run — if set, only include run files with run_num >= since_run

    Returns:
        merged   — {test_name: (status, dur, output_lines, run_num)}
        run_nums — sorted list of run numbers that have files
        running  — list of currently-running test names (from live log)
        suite_done — True if the live log shows suite completion
    """
    merged = OrderedDict()
    run_nums = []

    # Read historical run files (filtered by since_run if specified)
    pattern = os.path.join(RESULTS_DIR, f"{suite}_run*.txt")
    run_files = sorted(glob.glob(pattern),
                       key=lambda p: int(re.search(r"_run(\d+)", p).group(1) or 0))
    for rf in run_files:
        m = re.search(r"_run(\d+)\.txt$", rf)
        rn = int(m.group(1)) if m else 0
        if since_run is not None and rn < since_run:
            continue
        run_nums.append(rn)
        partial, _ = _parse_single_log(rf, run_num=rn)
        for name, (status, dur, output, run_n) in partial.items():
            existing = merged.get(name)
            if existing is None or PRIORITY.get(status, 0) > PRIORITY.get(existing[0], 0):
                merged[name] = (status, dur, output, run_n)

    # Live log (current run, may be in-progress)
    live_log = f"/tmp/test_{suite}.log"
    live_results, suite_done = _parse_single_log(live_log, run_num="live")
    running = []

    if not suite_done and os.path.exists(live_log):
        last_run_tests = []
        finished = set()
        with open(live_log) as f:
            for line in f:
                line = re.sub(r"\x1b\[[0-9;]*m", "", line)
                mr = re.match(r"^\s*=== RUN\s+(.+/.+)", line)
                if mr:
                    last_run_tests.append(mr.group(1).strip())
                md = re.match(r"^\s*--- (PASS|FAIL|SKIP): (.+?) \(", line)
                if md:
                    finished.add(md.group(2))
        running = [t for t in last_run_tests if t not in finished][-5:]

    # Merge live results (only if they improve on historical data)
    for name, (status, dur, output, _) in live_results.items():
        existing = merged.get(name)
        if existing is None or PRIORITY.get(status, 0) > PRIORITY.get(existing[0], 0):
            merged[name] = (status, dur, output, "live")

    return merged, run_nums, running, suite_done


# ── HTML helpers ──────────────────────────────────────────────────────────────

def _runs_history_html(runs, active_suite=None):
    """Generate a compact runs history table from runs_meta entries."""
    if not runs:
        return ""
    lines = []
    lines.append('<div class="run-history"><div class="run-history-title">Run History</div>')
    lines.append('<table style="margin-bottom:10px"><tr>'
                 '<th>Suite</th><th>Run #</th><th>Mode</th><th>Time</th>'
                 '<th>Pass</th><th>Fail</th><th>Skip</th><th>Status</th></tr>')
    for entry in runs:
        suite = entry.get("suite", "?")
        if active_suite and suite != active_suite:
            continue
        run_n  = entry.get("run", "?")
        mode   = entry.get("mode", "?")
        ts     = entry.get("timestamp", "")[:16].replace("T", " ")
        passed = entry.get("passed", 0)
        failed = entry.get("failed", 0)
        skipped= entry.get("skipped", 0)
        mode_lbl = MODE_LABEL.get(mode, mode)
        status_cls  = "F" if failed > 0 else "P"
        status_text = "FAIL" if failed > 0 else "PASS"
        mode_style = ""
        if mode == "full_retest":
            mode_style = "background:#e8f5e9;font-weight:bold"
        elif mode == "rerun_failed":
            mode_style = "background:#fff8e1"
        lines.append(
            f'<tr class="{status_cls}">'
            f'<td>{html.escape(suite)}</td>'
            f'<td style="text-align:center">{run_n}</td>'
            f'<td style="{mode_style}">{html.escape(mode_lbl)}</td>'
            f'<td>{html.escape(ts)}</td>'
            f'<td style="color:green">{passed}</td>'
            f'<td style="color:red;font-weight:bold">{failed}</td>'
            f'<td style="color:orange">{skipped}</td>'
            f'<td class="{status_cls}">{status_text}</td>'
            f'</tr>'
        )
    lines.append('</table></div>')
    return "\n".join(lines)


def _run_badge(run_num, initial_status=None, final_status=None):
    """Small badge showing run source + whether it was fixed in a retry."""
    if run_num == "live":
        badge = '<span style="font-size:10px;color:#1a73e8">[live]</span>'
    elif run_num is not None:
        badge = f'<span style="font-size:10px;color:#888">[run {run_num}]</span>'
    else:
        badge = ""
    if initial_status and final_status and initial_status != final_status \
            and PRIORITY.get(final_status, 0) > PRIORITY.get(initial_status, 0):
        badge += ' <span style="font-size:10px;background:#c8e6c9;color:#1b5e20;padding:1px 4px;border-radius:3px">fixed in retry</span>'
    return badge


def _extract_error_summary(output_lines):
    if not output_lines:
        return []
    meaningful = [l for l in output_lines
                  if not any(x in l for x in ["=== RUN", "=== PAUSE", "=== CONT", "--- PASS", "--- FAIL"])]
    return meaningful[-20:]


def _extract_skip_reason(output_lines):
    if not output_lines:
        return ""
    for line in output_lines:
        if any(x in line for x in ["Skip", "skip", "not configured", "not deployed",
                                    "disabled", "not available", "not set"]):
            return line
    return output_lines[-1] if output_lines else ""


# ── generate_combined_html (single page: summary + subtests) ─────────────────

def generate_combined_html():
    """Generate a single HTML page with summary table + full subtest detail."""
    now  = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
    runs = parse_runs_meta()
    lft  = last_full_retest_index(runs)

    lines = []
    lines.append('<!DOCTYPE html><html><head><meta charset="UTF-8"><title>Test Results</title>')
    lines.append('<style>')
    lines.append('body{font-family:monospace;margin:10px;font-size:12px;background:#fafafa}')
    lines.append('table{border-collapse:collapse;width:100%;margin-bottom:15px}')
    lines.append('th,td{border:1px solid #ccc;padding:4px 8px;text-align:left;vertical-align:top}')
    lines.append('th{background:#1a73e8;color:#fff;position:sticky;top:0;z-index:1}')
    lines.append('.P{color:green}.F{color:red;font-weight:bold}.S{color:orange}.R{color:blue}')
    lines.append('.updated{animation:flash 0.6s}@keyframes flash{0%{background:#fff3cd}100%{background:transparent}}')
    lines.append('#status{position:fixed;top:5px;right:10px;font-size:11px;color:#666;background:#fff;padding:2px 8px;border-radius:3px;border:1px solid #ddd;z-index:10}')
    lines.append('.run-history{margin-bottom:15px}.run-history-title{font-weight:bold;font-size:12px;color:#444;margin-bottom:4px}')
    lines.append('td:nth-child(5){max-width:250px;word-break:break-all;overflow-wrap:break-word}')
    lines.append('td:nth-child(4),th:nth-child(4){min-width:200px;max-width:450px;word-break:break-word}')
    lines.append('.test-parent{color:#1a73e8;font-weight:bold;font-size:11px}')
    lines.append('.test-sub{font-size:12px;margin-top:2px}')
    lines.append('.err{background:#fff0f0;color:#c00;font-size:11px;white-space:pre-wrap;max-width:900px;word-break:break-word}')
    lines.append('.skip-reason{background:#fff8e1;color:#795548;font-size:11px;white-space:pre-wrap}')
    lines.append('.pass-ok{color:#388e3c;font-size:11px}')
    lines.append('.filter-bar{margin:10px 0;padding:8px;background:#fff;border:1px solid #ddd;border-radius:4px}')
    lines.append('.filter-bar input{padding:4px 8px;font-family:monospace;font-size:12px;width:300px}')
    lines.append('.filter-bar button{margin-left:5px;padding:4px 12px;cursor:pointer;font-family:monospace}')
    lines.append('.filter-bar label{margin-left:15px;cursor:pointer}')
    lines.append('.cnt{font-size:11px;color:#666;margin-left:10px}')
    lines.append('.section-header{font-size:13px;font-weight:bold;color:#333;margin:15px 0 5px;padding-bottom:3px;border-bottom:2px solid #1a73e8}')
    lines.append('</style></head><body>')
    lines.append('<div id="status">Live</div>')
    lines.append('<div id="content">')
    lines.append(f'<b>0Chain System Test Results</b> &mdash; <span id="ts">{now}</span>'
                 f' (auto-refreshes 5s)<hr>')
    lines.append('<i>Results deduplicated across retry runs (best result per test)</i><br><br>')

    # Run history
    lines.append(_runs_history_html(runs))

    # Determine since_run for deduplication
    since_run = None
    if lft >= 0:
        ts = runs[lft].get("timestamp", "")[:16].replace("T", " ")
        lines.append(f'<div style="background:#e8f5e9;border:1px solid #4caf50;padding:6px 10px;'
                     f'margin:8px 0;border-radius:4px">&#x2705; <b>Clean full re-test</b> — '
                     f'started {ts}. Data below reflects fresh run only.</div>')
        try:
            since_run = int(runs[lft].get("run", 0))
        except (ValueError, TypeError):
            since_run = None

    # Build consolidated results
    all_consolidated = {}
    all_running = {}
    all_detailed = {}
    total_p = total_f = total_s = 0

    for suite in SUITES:
        merged, run_nums, running, _ = parse_consolidated_for_suite(suite, since_run=since_run)
        all_consolidated[suite] = {n: (v[0], v[1]) for n, v in merged.items()}
        all_detailed[suite] = merged
        all_running[suite] = running
        total_p += sum(1 for s, *_ in merged.values() if s == "PASS")
        total_f += sum(1 for s, *_ in merged.values() if s == "FAIL")
        total_s += sum(1 for s, *_ in merged.values() if s == "SKIP")

    # ── Summary table ──
    lines.append('<div class="section-header">Suite Summary</div>')
    lines.append('<table><tr><th>Suite</th><th>Pass</th><th>Fail</th><th>Skip</th>'
                 '<th>Running</th><th>Runs</th><th>Log</th></tr>')
    for suite in SUITES:
        results = all_consolidated[suite]
        p  = sum(1 for s, _ in results.values() if s == "PASS")
        fl = sum(1 for s, _ in results.values() if s == "FAIL")
        sk = sum(1 for s, _ in results.values() if s == "SKIP")
        n_runs = len(glob.glob(os.path.join(RESULTS_DIR, f"{suite}_run*.txt")))
        n_runs_str = str(n_runs) if n_runs else "-"
        logfile = "/tmp/test_" + suite + ".log"

        if not results and not os.path.exists(logfile):
            lines.append(f'<tr><td>{suite.upper()}</td><td colspan=6>not started</td></tr>')
        else:
            cls = "F" if fl > 0 else "P"
            rn  = len(all_running.get(suite) or [])
            lines.append(f'<tr class="{cls}"><td><b>{suite.upper()}</b></td>'
                         f'<td>{p}</td><td>{fl}</td><td>{sk}</td><td>{rn}</td>'
                         f'<td>{n_runs_str}</td>'
                         f'<td><a href="/test/{suite}">log</a></td></tr>')

    total_all = total_p + total_f + total_s
    lines.append(f'<tr style="font-weight:bold"><td>TOTAL</td><td>{total_p}</td>'
                 f'<td>{total_f}</td><td>{total_s}</td><td></td><td></td><td></td></tr>')
    lines.append('</table>')
    lines.append(f'<b>Pass: {total_p} | Fail: {total_f} | Skip: {total_s} | '
                 f'Total: {total_all} &mdash; <i>consolidated (best per test)</i></b>')

    # ── Subtests detail ──
    lines.append('<div class="section-header">Subtest Detail</div>')

    # Filter bar
    lines.append('<div class="filter-bar">')
    lines.append('Filter: <input type="text" id="filter" placeholder="Type to filter test names...">')
    lines.append('<button onclick="applyFilter()">Go</button>')
    lines.append('<label><input type="checkbox" id="showPass" checked onchange="applyFilter()"> Pass</label>')
    lines.append('<label><input type="checkbox" id="showFail" checked onchange="applyFilter()"> Fail</label>')
    lines.append('<label><input type="checkbox" id="showSkip" checked onchange="applyFilter()"> Skip</label>')
    lines.append('<label><input type="checkbox" id="showRunning" checked onchange="applyFilter()"> Running</label>')
    lines.append('<span id="countLabel" class="cnt"></span>')
    lines.append('</div>')

    # Build run1 status index for "fixed in retry" badges
    def get_run1_status(suite, name):
        rf = os.path.join(RESULTS_DIR, f"{suite}_run1.txt")
        if not os.path.exists(rf):
            return None
        partial, _ = _parse_single_log(rf, run_num=1)
        entry = partial.get(name)
        return entry[0] if entry else None

    lines.append('<table id="results"><tr>'
                 '<th>#</th><th>Suite</th><th>Test</th><th>Status</th>'
                 '<th>Duration</th><th>Source</th><th>Output / Error</th></tr>')

    row_num = 0

    def emit_row(suite, name, status, dur, output, run_num, initial_status):
        nonlocal row_num
        row_num += 1
        cls = {"PASS": "P", "FAIL": "F", "SKIP": "S"}.get(status, "R")
        badge = _run_badge(run_num, initial_status, status)

        if status == "FAIL":
            err_lines = _extract_error_summary(output)
            err_html = html.escape("\n".join(err_lines)) if err_lines else "<i>No error details</i>"
            out_cell = f'<td class="err">{err_html}</td>'
        elif status == "SKIP":
            reason = _extract_skip_reason(output)
            out_cell = f'<td class="skip-reason">{html.escape(reason) if reason else "<i>No reason</i>"}</td>'
        elif status == "RUNNING":
            out_cell = '<td style="color:blue;font-size:11px">in progress...</td>'
        else:
            out_cell = '<td class="pass-ok">OK</td>'

        parts = name.split("/", 1)
        if len(parts) == 2:
            parent_html = html.escape(parts[0])
            sub_html = html.escape(parts[1].replace("_", " "))
            name_cell = (f'<span class="test-parent">{parent_html}</span>'
                         f'<br><span class="test-sub">{sub_html}</span>')
        else:
            name_cell = html.escape(name)

        data_name = name.lower().replace("_", " ")
        lines.append(
            f'<tr class="{cls}" data-status="{status}" data-name="{html.escape(data_name)}">'
            f'<td>{row_num}</td><td>{html.escape(suite)}</td>'
            f'<td>{name_cell}</td><td>{status}</td>'
            f'<td>{dur}</td><td>{badge}</td>{out_cell}</tr>'
        )

    # FAIL first
    for suite in SUITES:
        for name, vals in all_detailed[suite].items():
            if vals[0] == "FAIL":
                emit_row(suite, name, vals[0], vals[1], vals[2], vals[3],
                         get_run1_status(suite, name))

    # RUNNING
    for suite in SUITES:
        for name in (all_running.get(suite) or []):
            if name not in all_detailed[suite]:
                emit_row(suite, name, "RUNNING", "...", [], "live", None)

    # SKIP
    for suite in SUITES:
        for name, vals in all_detailed[suite].items():
            if vals[0] == "SKIP":
                emit_row(suite, name, vals[0], vals[1], vals[2], vals[3],
                         get_run1_status(suite, name))

    # PASS
    for suite in SUITES:
        for name, vals in all_detailed[suite].items():
            if vals[0] == "PASS":
                emit_row(suite, name, vals[0], vals[1], vals[2], vals[3],
                         get_run1_status(suite, name))

    lines.append('</table>')
    lines.append('</div>')  # end #content

    # ── JavaScript: filter + auto-refresh ──
    lines.append('<script>')
    lines.append(
        'function applyFilter(){'
        'var q=document.getElementById("filter").value.toLowerCase();'
        'var showP=document.getElementById("showPass").checked;'
        'var showF=document.getElementById("showFail").checked;'
        'var showS=document.getElementById("showSkip").checked;'
        'var showR=document.getElementById("showRunning").checked;'
        'var rows=document.querySelectorAll("#results tr[data-status]");'
        'var shown=0;'
        'rows.forEach(function(r){'
        'var st=r.getAttribute("data-status");'
        'var fullName=r.getAttribute("data-name")||"";'
        'var vis=(st==="PASS"&&showP)||(st==="FAIL"&&showF)||(st==="SKIP"&&showS)||(st==="RUNNING"&&showR);'
        'if(vis&&q&&fullName.indexOf(q)<0)vis=false;'
        'r.style.display=vis?"":"none";'
        'if(vis)shown++;});'
        'document.getElementById("countLabel").textContent="Showing "+shown+" of "+rows.length;'
        '}'
    )
    lines.append('document.getElementById("filter").addEventListener("keyup",'
                 'function(e){if(e.key==="Enter")applyFilter();});')
    lines.append('applyFilter();')
    # Auto-refresh every 5s via JS fetch (replaces only the #content div to avoid scroll jump)
    lines.append(
        '(function(){'
        'var iv=5000,st=document.getElementById("status");'
        'function r(){'
        'st.textContent="Updating...";'
        'fetch(location.href,{cache:"no-store"})'
        '.then(function(res){return res.text()})'
        '.then(function(h){'
        'var p=new DOMParser(),d=p.parseFromString(h,"text/html");'
        'var nc=d.getElementById("content"),oc=document.getElementById("content");'
        'if(nc&&oc&&nc.innerHTML!==oc.innerHTML){'
        'var fq=document.getElementById("filter").value;'
        'var fp=document.getElementById("showPass").checked;'
        'var ff=document.getElementById("showFail").checked;'
        'var fs=document.getElementById("showSkip").checked;'
        'var fr=document.getElementById("showRunning").checked;'
        'oc.innerHTML=nc.innerHTML;'
        'document.getElementById("filter").value=fq;'
        'document.getElementById("showPass").checked=fp;'
        'document.getElementById("showFail").checked=ff;'
        'document.getElementById("showSkip").checked=fs;'
        'document.getElementById("showRunning").checked=fr;'
        'applyFilter();'
        'oc.querySelectorAll("tr.F,tr.P,tr.S").forEach(function(r){'
        'r.classList.add("updated");'
        'setTimeout(function(){r.classList.remove("updated")},600);});}'
        'st.textContent="Live";'
        '}).catch(function(){st.textContent="Offline";});}'
        'setInterval(r,iv);})();'
    )
    lines.append('</script>')
    lines.append('</body></html>')

    # Write atomically
    tmp = RESULTS_HTML + ".tmp"
    with open(tmp, "w") as f:
        f.write("\n".join(lines))
    os.replace(tmp, RESULTS_HTML)

    # Write a redirect at SUBTESTS_HTML so the old URL still works
    redirect_html = (
        '<!DOCTYPE html><html><head><meta charset="UTF-8">'
        '<meta http-equiv="refresh" content="0;url=/test/results">'
        '</head><body><a href="/test/results">Redirecting to test results...</a>'
        '</body></html>'
    )
    tmp2 = SUBTESTS_HTML + ".tmp"
    with open(tmp2, "w") as f:
        f.write(redirect_html)
    os.replace(tmp2, SUBTESTS_HTML)

    return total_p, total_f, total_s, total_all


# ── Legacy wrappers (kept for compatibility) ──────────────────────────────────

def generate_html():
    return generate_combined_html()


def generate_subtests_html():
    # Already written as redirect by generate_combined_html()
    pass


if __name__ == "__main__":
    os.makedirs("/var/log/0chain", exist_ok=True)
    if "--loop" in sys.argv:
        print("Starting results generator loop (Ctrl+C to stop)...")
        while True:
            try:
                p, fl, sk, total = generate_combined_html()
                sys.stdout.write(
                    "\r" + datetime.now().strftime("%H:%M:%S") + " | " +
                    str(p) + " PASS, " + str(fl) + " FAIL, " + str(sk) +
                    " SKIP (" + str(total) + " total)  "
                )
                sys.stdout.flush()
            except Exception as e:
                sys.stderr.write("Error: " + str(e) + "\n")
            time.sleep(5)
    else:
        p, fl, sk, total = generate_combined_html()
        print(f"Generated {RESULTS_HTML}: {p} PASS, {fl} FAIL, {sk} SKIP ({total} total)")
        print(f"Written redirect at {SUBTESTS_HTML}")
