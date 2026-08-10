#!/usr/bin/env python3
"""Regenerate an internal/calib profile from real provider measurements.

MAINTAINER-ONLY. This script is never run by library users and the
library never calls it or any network endpoint at runtime — see
internal/calib's package doc. It is the one place in this project where
network calls to a provider's counting API are expected: run it by hand,
review the diff, and commit the resulting profile JSON.

Usage:
    export ANTHROPIC_API_KEY=...
    python3 scripts/calibrate.py --provider anthropic \\
        --target-model claude-3-5-sonnet-20241022 \\
        --base-tokenizer o200k_base \\
        --corpus testdata/corpus \\
        --out internal/calib/profiles/anthropic-claude-v1.json

The corpus directory must contain one .txt file per calibration sample,
organized in subdirectories named after the Bucket they belong to:

    testdata/corpus/latin/*.txt
    testdata/corpus/cjk/*.txt
    testdata/corpus/code/*.txt
    testdata/corpus/other/*.txt

For each sample this script:
  1. Counts it with the local base tokenizer (via `go run` against this
     module, so the "base" count is produced by the exact same code path
     CountForModel will use — not a reimplementation that could drift).
  2. Counts it with the target provider's token-counting endpoint.
  3. Records the ratio target/base.

It then aggregates mean/p95/max ratio and mean/p95/max absolute
percentage error per bucket, and writes a Profile JSON matching
internal/calib.Profile's schema.

Do not hand-edit sample_count, the ratios, or corpus_sha256 in the output
file. Every number in a committed profile must trace back to a
calibration run recorded by this script; a profile with invented
constants is worse than no profile, because it launders a guess through a
struct that looks measured. If you don't yet have API access to run a
real calibration, leave the profile as the shipped uncalibrated identity
placeholder (sample_count: 0, all ratios 1.0) rather than filling in
numbers by hand.
"""

from __future__ import annotations

import argparse
import dataclasses
import hashlib
import json
import os
import statistics
import subprocess
import sys
import time
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parent.parent
BUCKETS = ("latin", "cjk", "code", "other")


@dataclasses.dataclass
class Sample:
    bucket: str
    path: Path
    text: str


def load_corpus(corpus_dir: Path) -> list[Sample]:
    corpus_dir = corpus_dir.resolve()  # s.path must be absolute for relative_to(REPO_ROOT)
    samples: list[Sample] = []
    for bucket in BUCKETS:
        bucket_dir = corpus_dir / bucket
        if not bucket_dir.is_dir():
            continue
        for f in sorted(bucket_dir.glob("*.txt")):
            samples.append(Sample(bucket=bucket, path=f, text=f.read_text(encoding="utf-8")))
    if not samples:
        sys.exit(f"no corpus samples found under {corpus_dir} (expected {corpus_dir}/<bucket>/*.txt)")
    return samples


def corpus_sha256(samples: list[Sample]) -> str:
    h = hashlib.sha256()
    for s in sorted(samples, key=lambda s: str(s.path)):
        h.update(str(s.path.relative_to(REPO_ROOT)).encode("utf-8"))
        h.update(b"\0")
        h.update(s.text.encode("utf-8"))
        h.update(b"\0")
    return h.hexdigest()


def count_base_tokens(base_tokenizer: str, texts: list[str]) -> list[int]:
    """Count every sample with the library's own base tokenizer, via
    `go run` against a tiny helper program in this repo, so the "base"
    number is guaranteed to be produced by the exact same code
    CountForModel uses at runtime — never a reimplementation."""
    helper = REPO_ROOT / "scripts" / "count_helper.go"
    payload = json.dumps({"encoding": base_tokenizer, "texts": texts})
    proc = subprocess.run(
        ["go", "run", str(helper)],
        input=payload,
        capture_output=True,
        text=True,
        cwd=REPO_ROOT,
    )
    if proc.returncode != 0:
        sys.exit(f"go run {helper} failed:\n{proc.stderr}")
    return json.loads(proc.stdout)["counts"]


def count_anthropic_tokens(model: str, texts: list[str]) -> list[int]:
    """Count every sample with Anthropic's Messages count_tokens endpoint.
    Requires ANTHROPIC_API_KEY. Imported lazily so `pip install
    anthropic` is only required when actually calibrating Anthropic."""
    import anthropic  # type: ignore

    client = anthropic.Anthropic()
    counts = []
    for text in texts:
        resp = client.messages.count_tokens(
            model=model,
            messages=[{"role": "user", "content": text}],
        )
        counts.append(resp.input_tokens)
        time.sleep(0.05)  # be a polite client; this is a one-off offline run, not a hot path
    return counts


def count_gemini_tokens(model: str, texts: list[str]) -> list[int]:
    """Count every sample with Gemini's countTokens endpoint. Requires
    GOOGLE_API_KEY. Imported lazily for the same reason as Anthropic."""
    import google.generativeai as genai  # type: ignore

    genai.configure(api_key=os.environ["GOOGLE_API_KEY"])
    m = genai.GenerativeModel(model)
    counts = []
    for text in texts:
        counts.append(m.count_tokens(text).total_tokens)
        time.sleep(0.05)
    return counts


PROVIDER_COUNTERS = {
    "anthropic": count_anthropic_tokens,
    "gemini": count_gemini_tokens,
}


def pct_abs_error(estimate: float, actual: float) -> float:
    if actual == 0:
        return 0.0
    return abs(estimate - actual) / actual * 100.0


def aggregate_bucket(pairs: list[tuple[int, int]]) -> dict:
    """pairs is a list of (base_count, target_count) for one bucket."""
    ratios = [t / b for b, t in pairs if b > 0]
    errors = [pct_abs_error(round(b * (sum(ratios) / len(ratios))), t) for b, t in pairs if b > 0]
    ratios.sort()
    errors.sort()

    def p95(xs: list[float]) -> float:
        if not xs:
            return 0.0
        idx = min(len(xs) - 1, int(round(0.95 * (len(xs) - 1))))
        return xs[idx]

    return {
        "mean": statistics.mean(ratios) if ratios else 1.0,
        "p95": p95(ratios) if ratios else 1.0,
        "max": max(ratios) if ratios else 1.0,
        "mean_abs_error_pct": statistics.mean(errors) if errors else 0.0,
        "p95_abs_error_pct": p95(errors) if errors else 0.0,
        "max_abs_error_pct": max(errors) if errors else 0.0,
    }


def main() -> None:
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--provider", required=True, choices=sorted(PROVIDER_COUNTERS))
    ap.add_argument("--target-model", required=True, help="model ID passed to the provider's counting API")
    ap.add_argument("--base-tokenizer", required=True, help='embedded Encoding name, e.g. "o200k_base"')
    ap.add_argument("--corpus", required=True, type=Path)
    ap.add_argument("--out", required=True, type=Path)
    ap.add_argument("--profile-id", required=True, help='e.g. "anthropic-claude-v1"')
    args = ap.parse_args()

    samples = load_corpus(args.corpus)
    sha = corpus_sha256(samples)

    texts = [s.text for s in samples]
    print(f"counting {len(texts)} samples with base tokenizer {args.base_tokenizer}...", file=sys.stderr)
    base_counts = count_base_tokens(args.base_tokenizer, texts)

    print(f"counting {len(texts)} samples against {args.provider} ({args.target_model})...", file=sys.stderr)
    target_counts = PROVIDER_COUNTERS[args.provider](args.target_model, texts)

    by_bucket: dict[str, list[tuple[int, int]]] = {b: [] for b in BUCKETS}
    for s, b, t in zip(samples, base_counts, target_counts):
        by_bucket[s.bucket].append((b, t))

    ratios = {}
    for bucket in BUCKETS:
        pairs = by_bucket[bucket]
        if not pairs:
            print(f"warning: no samples for bucket {bucket!r}; leaving it as identity (mean=1.0)", file=sys.stderr)
            ratios[bucket] = {"mean": 1.0, "p95": 1.0, "max": 1.0, "mean_abs_error_pct": 0.0, "p95_abs_error_pct": 0.0, "max_abs_error_pct": 0.0}
        else:
            ratios[bucket] = aggregate_bucket(pairs)

    profile = {
        "profile_id": args.profile_id,
        "target_model": args.target_model,
        "provider": args.provider,
        "base_tokenizer": args.base_tokenizer,
        "ratios": ratios,
        "sample_count": len(samples),
        "corpus_sha256": sha,
        "generated_at": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
        "tool_version": "calibrate.py/1",
        "notes": f"Calibrated from {len(samples)} samples across {sum(1 for b in BUCKETS if by_bucket[b])} buckets. "
                 f"Regenerate with: python3 scripts/calibrate.py --provider {args.provider} "
                 f"--target-model {args.target_model} --base-tokenizer {args.base_tokenizer} "
                 f"--corpus {args.corpus} --out {args.out} --profile-id {args.profile_id}",
    }

    args.out.write_text(json.dumps(profile, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
    print(f"wrote {args.out} ({len(samples)} samples, corpus_sha256={sha[:12]}...)", file=sys.stderr)


if __name__ == "__main__":
    main()
