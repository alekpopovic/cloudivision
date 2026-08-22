#!/usr/bin/env python3
"""Update cloudivision prompt tracker.

Usage:
  python3 scripts/mark_prompt_done.py 21
  python3 scripts/mark_prompt_done.py 27 --status blocked --note "Waiting for envtest setup"
"""
from __future__ import annotations
import argparse
import json
from pathlib import Path
from datetime import datetime, timezone

ROOT = Path(__file__).resolve().parents[1]
TRACKER = ROOT / 'tracker' / 'prompt-tracker.json'
TRACKER_MD = ROOT / 'tracker' / 'PROMPT_TRACKER.md'

VALID = {'done', 'in_progress', 'blocked', 'pending', 'skipped'}

def render_tracker_md(tr):
    lines = [
        '# cloudivision Prompt Tracker',
        '',
        f"Project: `{tr['project']}`",
        f"Created at: `{tr.get('created_at', '')}`",
        f"Updated at: `{tr.get('updated_at', '')}`",
        f"Last executed prompt: `{tr.get('last_executed_prompt')}`",
        f"Next prompt: `{tr.get('next_prompt')}`",
        '',
        '## How to update',
        '',
        'Run:',
        '',
        '```bash',
        'python3 scripts/mark_prompt_done.py 21',
        '```',
        '',
        'Replace `21` with the last prompt that was successfully executed.',
        '',
        '## Status legend',
        '',
    ]
    for k, v in tr.get('status_legend', {}).items():
        lines.append(f'- `{k}` — {v}')
    lines += ['', '## Prompt status', '', '| ID | Status | Phase | Title | File | Notes |', '|---:|---|---|---|---|---|']
    for item in tr.get('prompts', []):
        notes = (item.get('notes') or '').replace('|', '\\|')
        lines.append(f"| {item['id']:02d} | {item['status']} | {item['phase']} | {item['title']} | `{item['file']}` | {notes} |")
    return '\n'.join(lines) + '\n'

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('prompt_id', type=int)
    parser.add_argument('--status', choices=sorted(VALID), default='done')
    parser.add_argument('--note', default='')
    parser.add_argument('--only-one', action='store_true', help='Update only the selected prompt, not all previous prompts.')
    args = parser.parse_args()

    tr = json.loads(TRACKER.read_text(encoding='utf-8'))
    ids = [p['id'] for p in tr['prompts']]
    if args.prompt_id not in ids:
        raise SystemExit(f'Prompt ID {args.prompt_id} does not exist. Valid range: {min(ids)}-{max(ids)}')

    now = datetime.now(timezone.utc).strftime('%Y-%m-%dT%H:%M:%SZ')
    for p in tr['prompts']:
        if args.only_one:
            should_update = p['id'] == args.prompt_id
        else:
            should_update = p['id'] <= args.prompt_id if args.status == 'done' else p['id'] == args.prompt_id
        if should_update:
            previous_status = p['status']
            p['status'] = args.status
            if args.status == 'done' and previous_status != 'done':
                p['executed_at'] = now
            if args.note and p['id'] == args.prompt_id:
                existing = p.get('notes') or ''
                p['notes'] = (existing + ' ' + args.note).strip()

    done_ids = [p['id'] for p in tr['prompts'] if p['status'] == 'done']
    tr['last_executed_prompt'] = max(done_ids) if done_ids else None
    pending_after = [p['id'] for p in tr['prompts'] if p['status'] == 'pending' and (tr['last_executed_prompt'] is None or p['id'] > tr['last_executed_prompt'])]
    tr['next_prompt'] = min(pending_after) if pending_after else None
    tr['updated_at'] = now

    TRACKER.write_text(json.dumps(tr, indent=2, ensure_ascii=False) + '\n', encoding='utf-8')
    TRACKER_MD.write_text(render_tracker_md(tr), encoding='utf-8')
    print(f"Updated tracker: last_executed_prompt={tr['last_executed_prompt']}, next_prompt={tr['next_prompt']}")

if __name__ == '__main__':
    main()
