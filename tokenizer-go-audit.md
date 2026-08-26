# Code Audit Report — tokenizer-go

> **Generated:** 2026-08-10
> **Analyzer:** Deep Codebase Analyzer
> **Files Reviewed:** 45 Go files + 2 embedded vocabularies, CI config, docs
> **Version audited:** v1.0.0 (`35b92fb`)
> **Note:** No Go toolchain was available in the analysis sandbox (network-restricted). All findings are from static reading of the source. Items marked **[verify]** should be confirmed with a compiler/benchmark run before acting.

---

## 1. PROJECT OVERVIEW

`tokenizer-go` (module `github.com/xorvus/tokenizer-go`) is a pure-Go reimplementation of OpenAI's `tiktoken` byte-pair-encoding tokenizer. No CGO, no runtime downloads — `cl100k_base.tiktoken` (1.7 MB) and `o200k_base.tiktoken` (3.5 MB) are compiled in via `//go:embed` in `embed.go`.

**What it actually does:**

- Splits text with a Unicode regex (`openai.PatternCL100K` / `PatternO200K`) into "pieces", then runs greedy byte-pair merges per piece against a rank table (`internal/bpe/bpe.go`).
- Exposes three encodings: `cl100k_base`, `o200k_base`, `o200k_harmony` (the last is `o200k_base` ranks + five extra Harmony special tokens, `internal/openai/o200k.go:12`).
- Maps model names → encodings through *two independent mechanisms* (`openai.ModelToEncoding` and `Registry` in `registry.go`) — see §3.
- Offers batch APIs with goroutine fan-out, context-aware wrappers, custom-vocabulary construction (`New(Config)`), and a partially-built multi-provider abstraction (`Provider`, `Engine`, `Calibration`, `Accuracy`).

**Target use case**, per README: AI gateways / proxies that need to count tokens per request before forwarding to a model provider. That framing matters — it makes the caching and DoS findings in §7/§8 severe rather than academic.

**Actual scope vs. advertised scope:** the README advertises a multi-provider tokenizer ("Anthropic, Qwen, Kimi, DeepSeek, Grok, Mistral, Gemini" appear as `Provider` constants in `provider.go`). Only OpenAI is implemented. `docs/superpowers/plans/2026-08-06-multi-provider-tokenizer.md` confirms this is an in-progress plan, but the type surface for it is already frozen into a v1.0.0 SemVer promise.

---

## 2. PROJECT STRUCTURE ANALYSIS

```
/                      public API (package tokenizer) — 13 files, all flat
  tokenizer.go         Tokenizer type, constructors, all Encode/Decode/Count methods (390 LOC)
  registry.go          Registry + ModelSpec resolution (112 LOC)
  embed.go             //go:embed of the two .tiktoken vocab files
  errors.go            12 sentinel errors
  options.go           Options struct + sanitizeOptions
  count_options.go     CountConfig + functional options
  count_result.go      CountResult, Accuracy, Calibration
  engine.go            Engine interface, EngineMetadata, Algorithm  ← unused
  capabilities.go      Capabilities bitmask                          ← never read
  provider.go          Provider constants                            ← 7 of 8 unimplemented
  model.go             ModelSpec                                     ← 5 of 9 fields unused
  resolution.go        Resolution
  encoding.go          Encoding constants

internal/bpe/          the actual tokenizer engine
  bpe.go               RankIndex (3-tier lookup) + byte-pair merge loop
  core.go              CoreBPE struct, construction, validation, special-token regex
  encode.go            regex splitting, batch fan-out, special-token scanning (422 LOC)
  decode.go            21 LOC token→bytes
  byte_cursor.go       rune-index → byte-offset cursor for non-ASCII
  regexp2_codegen_*.go 1185 LOC of generated regexp2 engine code (registered via init())

internal/openai/       vocab parsing + model tables
  vocabulary.go        base64 .tiktoken parser
  models.go            ModelToEncoding / ModelPrefixToEncoding maps
  cl100k.go, o200k.go  patterns + special-token tables

test/                  separate `package main` benchmark harness (not part of the module's test suite)
scripts/               generate_fixtures.py — generates parity fixtures that nothing reads
```

**Layering assessment.** The `public API → internal/bpe → internal/openai` direction is clean and correctly enforced by Go's `internal/` rule. The problem is not the layering, it's that the top layer is *two* layers glued together: a thin, working tiktoken port, and a speculative multi-provider framework with no implementations behind it. Roughly 6 of the 13 root files (`engine.go`, `capabilities.go`, `provider.go`, `model.go`, `resolution.go`, `count_options.go`) exist to serve a system that does not yet exist.

The root package is also flat with no sub-grouping — 13 files where `Tokenizer`'s 20 methods all live in one 390-line file alongside the `CountForModel` free function.

---

## 3. ARCHITECTURE ANALYSIS

**Pattern:** a library-style layered design. `Tokenizer` is a thin, stateless facade (`tokenizer.go:21-24`) delegating to `*bpe.CoreBPE`. There is no dependency injection, no interfaces at the boundary (the `Engine` interface exists in `engine.go:19` but `Tokenizer` does not implement it as a declared contract, and nothing consumes it).

### 3.1 Dependency flow

```
Tokenizer ──► bpe.CoreBPE ──► RankIndex ──► map[string]int / [65536]int32 / map[uint64]int32
    │                └──────► regexp2.Regexp (backtracking engine)
    ├──► openai.ParseVocabulary (construction only)
    ├──► openai.ModelToEncoding      ← path A for model resolution
    └──► Registry (globalRegistry)   ← path B for model resolution
```

### 3.2 Architectural violation #1 — two competing model registries

This is the most consequential structural defect.

`ForModel(model)` (`tokenizer.go:148`) resolves through `EncodingForModel`, which reads `openai.ModelToEncoding` and `openai.ModelPrefixToEncoding` (`tokenizer.go:156-166`).

`CountForModel(model, text)` (`tokenizer.go:355`) resolves through `ResolveModel` → `globalRegistry.Resolve` (`registry.go:70`), which reads a **completely separate** table hand-built in `registry.go:38-68`.

The two tables disagree. Concretely:

| Model string | `ForModel` (openai maps) | `CountForModel` (Registry) |
|---|---|---|
| `gpt-4.1` | `o200k_base` ✅ | **`ErrUnknownModel`** ❌ (absent from Registry) |
| `o4-mini` | `o200k_base` ✅ | **`ErrUnknownModel`** ❌ |
| `chatgpt-4o-latest` | `o200k_base` ✅ | **`ErrUnknownModel`** ❌ |
| `gpt-35-turbo` | `cl100k_base` ✅ | **`ErrUnknownModel`** ❌ |
| `gpt-4o-mini` | `o200k_base` ✅ | `o200k_base` ✅ |
| `text-davinci-003` | `p50k_base` → then `ErrUnknownEncoding` ❌ | `ErrUnknownModel` ❌ |

Two public entry points to the same question return different answers. For a token-counting gateway this is a silent billing/limit divergence depending on which API the caller happened to pick.

### 3.3 Architectural violation #2 — nondeterministic prefix matching

`tokenizer.go:160`:

```go
for prefix, enc := range openai.ModelPrefixToEncoding {
    if strings.HasPrefix(model, prefix) {
        return Encoding(enc), nil
    }
}
```

This ranges over a **Go map**, whose iteration order is deliberately randomized per run. `ModelPrefixToEncoding` contains overlapping prefixes:

- `"ft:gpt-4"` → `cl100k_base`
- `"ft:gpt-4o"` → `o200k_base`

A fine-tuned model named `ft:gpt-4o:acme::abc123` matches **both**. Whichever the map yields first wins. **The same binary, same input, will return `cl100k_base` on one process start and `o200k_base` on the next.** Token counts for fine-tuned GPT-4o models are therefore nondeterministic across restarts — cl100k and o200k produce materially different counts for the same text.

`registry.go` gets this right (`sort.Slice` longest-prefix-first at line 66), which makes the divergence more frustrating: the correct algorithm exists in the codebase, it's just not the one `ForModel` uses. **[verify]** — trivially reproducible with `for i in {1..20}; do go run ...; done`.

Note the Registry's own sort is `sort.Slice`, which is **not stable**. Among the equal-length-8 prefixes (`"gpt-oss-"`, `"gpt-4.5-"`, `"ft:gpt-4"`) order is currently harmless because none overlap, but the invariant is undefended — adding one overlapping equal-length prefix silently reintroduces nondeterminism. Use `sort.SliceStable` with an explicit lexical tiebreak.

### 3.4 Architectural violation #3 — configuration that goes nowhere

`Tokenizer` carries an `options Options` field (`tokenizer.go:23`), set by `newTokenizer` and `WithOptions`. Grep for reads of that field across the whole repo:

```
$ grep -rn "\.options" --include=*.go .
tokenizer.go:23   options Options      ← declaration
tokenizer.go:93   options: DefaultOptions(),
tokenizer.go:171  options: sanitizeOptions(opts),
```

**Zero reads.** `WithOptions` is a no-op that allocates a new `Tokenizer` and discards the configuration. The values it purports to control are hardcoded elsewhere:

- `MaxWorkers` → `maxBatchWorkers = 4` and `maxParallelWorkers = 4` (`encode.go:102-103`)
- `BatchByteThreshold` → literal `16384` in `partitionByBytes` (`encode.go:183`)
- `ParallelByteThreshold` → literal `2048` in `EncodeOrdinarySequential` (`encode.go:109`), plus `numPieces >= 10000 && len(subText) >= 1000000` in `EncodeSubTextMatches` (`encode.go:339`)

README bullet "**Resource Options**: Configurable worker pools and byte thresholds via `tok.WithOptions(Options{...})`" is false. And `TestWithOptionsConfiguration` (`context_test.go:55`) does not catch it because it only asserts `len(tokens) != 0` after applying options — an assertion that passes whether or not options do anything.

### 3.5 Data flow (encode path)

```
Tokenizer.EncodeOrdinary(text)
  └─ CoreBPE.EncodeOrdinaryNative               encode.go:418
       ret := make([]int, 0, (len(text)+2)/3)   ← capacity heuristic
       └─ EncodeSubTextMatches(text, ret)        encode.go:333
            └─ Regex.FindAllStringIndex(text,-1) ← MATERIALIZES ALL MATCHES
            └─ for each piece: EncodePieceTo
                 └─ RankIndex.Lookup(piece)      ← whole-piece fast path
                 └─ BytePairEncodeToWithIndex    bpe.go:188  ← O(n²) merge
```

Note what is **not** in this path: the `MatchIterator` streaming matcher. See §7.1.

---

## 4. EXECUTION FLOW

### 4.1 Initialization

There is no explicit init phase for the tokenizer itself, but two things happen at package load:

1. `registry.go:22` — `var globalRegistry = newDefaultRegistry()` builds ~16 model specs and sorts 10 prefix entries. Cheap, runs once.
2. `internal/bpe/regexp2_codegen_cl100k.go:424` and the o200k equivalent — `func init()` calls `regexp2.RegisterEngine(<pattern literal>, ...)`, registering precompiled matcher functions keyed by the exact pattern string. This is why `regexp2.Compile(openai.PatternCL100K, regexp2.None)` in `core.go:91` gets the fast generated engine rather than the interpreter. Clever, but see §14 on the fragility: the registration key is a hand-copied string literal that must byte-match `openai.PatternCL100K`. If someone edits the pattern in `cl100k.go` without regenerating, the library **silently** falls back to the slow interpreted path with no test or build failure.

### 4.2 Tokenizer construction — the hot problem

```go
func GetEncoding(name Encoding) (*Tokenizer, error) {
    switch name {
    case CL100KBase:  return getEmbeddedCL100K()   // embed.go:17
    ...
}

func getEmbeddedCL100K() (*Tokenizer, error) {
    return NewFromVocabulary(strings.NewReader(cl100kVocab), ...)
}
```

**Every single call to `GetEncoding` re-does full construction from scratch.** There is no `sync.Once`, no `map[Encoding]*Tokenizer` cache, nothing. Each call performs:

1. `openai.ParseVocabulary` — a `bufio.Scanner` over 1.7 MB (cl100k) or 3.5 MB (o200k) of text; per line: `strings.TrimSpace`, `strings.Split`, `base64.StdEncoding.DecodeString`, `strconv.Atoi`, a map-existence check, a map insert. That's ~100,256 lines for cl100k and ~199,998 for o200k.
2. `buildDecoderSlice` — a second full pass building a `[]string` of `maxID+1`.
3. `bpe.NewCoreBPE` → `ValidateTokenIDs` — builds *another* `map[int]string` of ~200k entries purely to check ID uniqueness, then throws it away.
4. `NewRankIndex` — zeroes a `[65536]int32` (256 KB) and a `[256]int32`, then iterates all ~200k tokens again.
5. `regexp2.Compile` of the main pattern and the special-token pattern.

Rough cost: **~200k allocations and 10–20 MB of transient garbage per call**, likely 50–200 ms. **[verify]** — benchmark `GetEncoding` directly; this is the single most valuable measurement to take.

Now trace the advertised gateway path:

```go
func CountForModel(model, text string, ...) (CountResult, error) {
    ...
    tok, err := GetEncoding(Encoding(res.TokenizerID))   // tokenizer.go:370  ← per call!
    count, err := tok.Count(text)
}
```

`CountForModel` — the highest-level, most convenient, most likely-to-be-used-in-a-request-handler function in the library — **rebuilds the entire 200k-entry vocabulary on every invocation**. A gateway calling this per request will spend >99.9% of its time in vocabulary construction and generate sustained multi-GB/s garbage. The README's benchmark table (`37,448 ns/op`) measures only the steady-state encode of an already-constructed tokenizer and does not surface this at all.

The same applies to `ForModel`. `test/ablation_test.go:13` hoists `GetEncoding` out of the benchmark loop, so the benchmark suite is structurally blind to it.

This is the **#1 finding of this audit.**

### 4.3 Encode lifecycle (`EncodeOrdinary`)

1. Preallocate `ret` with capacity `(len(text)+2)/3` (`encode.go:419`). For English this over-allocates by roughly 1.3× (real ratio ≈ 4 bytes/token); for CJK it under-allocates and forces regrowth.
2. `EncodeSubTextMatches` runs `Regex.FindAllStringIndex(subText, -1)` — full materialization of every match as a `[]int{start,end}`. For a 3.4 MB document that is ~800k pieces × (a 2-elem slice header + backing array) ≈ **tens of MB of index metadata alive simultaneously**.
3. If `numPieces >= 10000 && len(subText) >= 1000000`, fan out to ≤4 goroutines, each appending into its own buffer, then concatenate in worker order (`encode.go:339-377`). Deterministic ordering is correct here.
4. Otherwise, sequential loop calling `EncodePieceTo` per piece.
5. `EncodePieceTo` first tries a whole-piece `RankIndex.Lookup` (the common case for real words — a genuinely good optimization), falling back to `BytePairEncodeToWithIndex`.

### 4.4 Encode-with-specials lifecycle (`Encode`)

`Tokenizer.Encode(text)` calls `t.core.EncodeNative(text, nil)` — **`allowed` is hardcoded `nil`**. `FindNextAllowedSpecial` returns immediately when `len(allowed) == 0` (`encode.go:27`). So `Encode` is functionally identical to `EncodeOrdinary`. There is no exported way to reach the special-token path at all. See §8.2.

### 4.5 Batch lifecycle

`EncodeOrdinaryBatchNative` (`encode.go:203`) partitions by cumulative byte count into ≤4 contiguous ranges, one goroutine each, writing into disjoint indices of a preallocated `results [][]int`. The disjoint-index pattern is race-free and correct. Errors are captured via `sync.Once` + a plain `firstErr` variable — see §10.1 for the race there.

---

## 5. COMPONENT BREAKDOWN

### 5.1 `RankIndex` (`internal/bpe/bpe.go:7-121`)

Three-tier lookup structure, the core performance idea of the library:

| Token length | Structure | Cost |
|---|---|---|
| 1 byte | `short1 [256]int32` | direct index |
| 2 bytes | `short2 [1<<16]int32` (256 KB) | direct index |
| 3–7 bytes | `shortPacked map[uint64]int32` | integer-key hash |
| 8+ bytes | `ranks map[string]int` | string hash |

`packKey` (`bpe.go:14`) encodes length in byte 0 and the token bytes in bytes 1–7. Since length occupies a disjoint byte from all content bytes, **the encoding is injective and collision-free** — `TestPackedKeyCollisionCornerCases` (`bpe_test.go:26`) correctly probes the dangerous cases (leading/trailing NULs, all-`0xFF`). This is well-engineered and well-tested.

Two issues:

- **`RankIndex` is embedded by value in `CoreBPE`** (`core.go:22`), and `NewRankIndex` returns it by value (`bpe.go:62`). The struct is ~257 KB. That's at least one 257 KB memcpy on every construction, on top of the zeroing loops at `bpe.go:43-48`. Should be `*RankIndex`.
- `NewRankIndex` **silently skips** tokens with `rank > math.MaxInt32` (`bpe.go:50`). No error, no log. A custom vocabulary with large ranks would produce a tokenizer that silently mis-encodes. Given `validateAndCopyConfig` only rejects negative ranks, this is reachable from public API.

### 5.2 Byte-pair merge loop (`bpe.go:139-167`)

```go
func findMinRank(parts []part) (int, int) {
    minRank, minIdx := math.MaxInt, -1
    for i := 0; i < len(parts)-1; i++ { ... }   // O(n) linear scan
}

func bytePairMergeLoop(piece string, parts []part, idx *RankIndex) []part {
    for len(parts) > 1 {                        // O(n) merges
        minRank, minIdx := findMinRank(parts)
        ...
        parts = mergeMinPart(piece, parts, minIdx, idx)
    }
}
```

**O(n²) in piece length**, where n = bytes in the piece. Plus `mergeMinPart` does `append(parts[:i+1], parts[i+2:]...)` — an O(n) memmove per merge, so the constant factor is real. This matches upstream Rust `tiktoken`'s algorithm, so it is not a porting error, but it is a load-bearing assumption that pieces stay short. §7.3 and §8.1 explain when that assumption breaks.

`makePartsBuffer` (`bpe.go:169`) uses a stack-allocated `[33]part` for pieces ≤ 32 bytes — a good escape-analysis-friendly optimization covering the overwhelming majority of real pieces.

### 5.3 `ByteCursor` (`internal/bpe/byte_cursor.go`)

Bridges regexp2's **rune** indices to Go's **byte** offsets. Fast path: if the whole text is ASCII, rune index == byte index. Slow path: monotonic forward `utf8.DecodeRuneInString` walk, amortizing to O(n) total across a full scan rather than O(n) per match. Correct and well-motivated.

Correctness note: `AdvanceTo` returns `c.BytePos` after the loop, and the loop is bounded by `c.BytePos < len(text)`. If `targetRune` exceeds the rune count of `text`, it silently returns `len(text)` rather than erroring. Benign for the current caller but an easy trap.

### 5.4 `MatchIterator` (`encode.go:60-100`)

The streaming, constant-memory alternative to `FindAllStringIndex`. Correctly implemented. **Used only by `EncodeOrdinarySequential` and `CountOrdinarySequential`, and only when `len(text) >= 2048`.** Those two functions are reachable **only** from the batch APIs. The primary single-text path (`EncodeOrdinaryNative` → `EncodeSubTextMatches`) never touches it. See §7.1.

### 5.5 `FindNextAllowedSpecial` (`encode.go:26-58`)

Scans for special tokens, skipping ones not in `allowed`. Two problems:

- `ascii := isASCII(text)` is computed **inside the function**, and the function is called once per special token found in a loop in `EncodeNative` (`encode.go:394`). For a document with *k* special tokens this is *k* full O(n) scans of the text — **O(n·k)**. Hoist it to `EncodeNative` or into `CoreBPE`.
- `startRune` is passed to `FindStringMatchStartingAt` as a **rune** index, but the ASCII branch then treats `match.RuneIndex` as a byte index. This is only sound because the two coincide for ASCII, and `isASCII` guards it. Correct, but the mixing of rune and byte units in the same local variables (`byteStart`, `RuneEnd`, `ByteEnd` in `SpecialMatch`) is a latent bug magnet.

### 5.6 `Registry` (`registry.go`)

Holds `sync.RWMutex` + `exactModels` + `aliases` + `prefixes`. **There is no exported mutation method** — no `Register`, no `AddAlias`. The `aliases` map is populated by nothing and is always empty (`resolveExactOrAlias:91` is dead code). The mutex is therefore pure overhead on a structure that is immutable after `init()`. `registry_bench_test.go` exists to benchmark `ResolveModel`, measuring a lock that protects nothing.

`resolvePrefix` (`registry.go:99`) sets `CanonicalModel: model` — i.e. the *requested* name, not a canonical one. Callers reading `CountResult.CanonicalModel` to group metrics will get one bucket per fine-tune ID.

### 5.7 Dead subsystem inventory

Fully unreferenced or write-only symbols, all exported and all frozen by the v1.0.0 SemVer promise:

| Symbol | File | Status |
|---|---|---|
| `Engine` interface | `engine.go:19` | implemented only by `dummyEngine` in `engine_test.go` |
| `EngineMetadata`, `Algorithm` + 4 constants | `engine.go` | never constructed |
| `Capabilities` + 6 constants | `capabilities.go` | written into `ModelSpec`, never read |
| `Provider` — Anthropic/Qwen/Kimi/DeepSeek/Grok/Mistral/Gemini | `provider.go` | no implementations |
| `ModelSpec.Aliases/Prefixes/FallbackProfile/ChatTemplateID/IsExact` | `model.go` | `IsExact` set but never read; other 4 never set |
| `Calibration` struct | `count_result.go:26` | `CountResult.Calibration` is always `nil` |
| `AccuracyEstimatedCalibrated`, `AccuracyEstimatedHeuristic` | `count_result.go` | never assigned |
| `CountConfig.AllowCalibratedFallback/AllowHeuristicFallback` | `count_options.go` | never read |
| `WithCalibratedFallback`, `WithHeuristicFallback` | `count_options.go` | set fields nothing reads |
| `ErrDisallowedSpecial` | `errors.go:8` | **never returned anywhere** |
| `Resolution.UsedFallback` / `.Reason` | `resolution.go` | never set to true / non-empty |

Consequence: `WithExactOnly()` is a guarantee that cannot fail. Its check is `if cfg.ExactOnly && res.UsedFallback` (`tokenizer.go:366`), and `res.UsedFallback` is `false` on every code path. A caller who writes `CountForModel(m, t, WithExactOnly())` believing it will error on an inexact tokenizer gets a silent no-op. That is worse than not offering the option.

---

## 6. CODE QUALITY REVIEW

**Strong points, stated plainly:** naming is consistent and idiomatic; functions are small (the largest, `EncodeSubTextMatches`, is 55 lines); sentinel errors are wrapped with `%w` throughout so `errors.Is` works; the `internal/` boundary is respected; there are no global mutable variables outside `globalRegistry`; `gofmt` compliance appears clean.

### 6.1 Duplication: `EncodeOrdinaryBatchNative` vs `CountOrdinaryBatchNative`

`encode.go:203-266` and `encode.go:268-331` are **63 lines that differ only in element type and which sequential function they call.** Identical partitioning, identical worker spawn, identical `sync.Once` error capture, identical structure. Same for `EncodeOrdinarySequential` vs `CountOrdinarySequential` (`encode.go:105-172`). In Go 1.26 this is a two-line generic:

```go
func batchMap[T any](texts []string, fn func(string) (T, error), maxWorkers int) ([]T, error)
```

Why it matters here specifically: the two copies have already begun to drift conceptually (`Count` uses `CountPieceToWithIndex` which skips output allocation; `Encode` doesn't), and any fix to the error-race in §10.1 must be applied in two places or it will be applied in one.

### 6.2 Duplication: model tables

`openai.ModelToEncoding` (44 entries) and `Registry.exactModels` (16 entries) encode the same knowledge with different coverage. `openai.ModelPrefixToEncoding` (17 entries) and `Registry.prefixes` (10 entries) likewise. Four tables, two of everything, no cross-check test.

### 6.3 Inconsistent thresholds

The number `16384` appears as `Options.ParallelByteThreshold` default, `Options.BatchByteThreshold` default, and as a magic literal in `partitionByBytes:183`. `2048` appears twice as a magic literal. `4` appears as `maxParallelWorkers` and `maxBatchWorkers`. `10000` and `1000000` appear once each with no comment explaining derivation. None of these are connected to the `Options` struct that exists to hold them.

### 6.4 `lastLen` — a return value with two different meanings

`EncodeSubTextMatches` returns `(ret, lastLen, err)`. In the sequential branch (`encode.go:380-385`):

```go
lastLen := 0
for _, pair := range indices {
    before := len(ret)
    ret = bp.EncodePieceTo(...)
    lastLen = len(ret) - before      // tokens produced by the LAST PIECE
}
```

In the parallel branch (`encode.go:369-375`):

```go
lastLen := 0
for _, chunk := range chunks {
    ret = append(ret, chunk...)
    if len(chunk) > 0 { lastLen = len(chunk) }   // tokens in the LAST CHUNK (~200k)
}
```

Same variable, same return position, values differing by ~5 orders of magnitude depending on an input-size threshold. Currently harmless because every caller discards it (`EncodeNative` returns it, `Tokenizer.Encode` drops it with `_`). But it is a live landmine: this value is exactly the shape of thing a future "unstable last token" / streaming API would consume, and it would be wrong 1% of the time — only on large inputs, only in the parallel path. Either fix it or delete it.

### 6.5 Tests that cannot fail

Several tests assert only non-emptiness and would pass against a tokenizer that returns `[]int{1}` for all input:

- `parity_test.go:19` — `if len(tokens) == 0` (test is named *parity*)
- `regex_test.go:19`, `special_test.go:23`, `bpe_test.go` empty case
- `context_test.go:69` — `TestWithOptionsConfiguration`, discussed in §3.4
- `count_for_model_test.go` — checks `UsedFallback == false`, which is unconditionally true

Genuine exact-value assertions exist in exactly **two places**: `tokenizer_test.go:21` (`"hello world"` → `[24912, 2375]` for o200k) and `tokenizer_test.go:41` (`→ [15339, 1917]` for cl100k). That is **two token vectors, one two-word ASCII string, against a 200,000-entry vocabulary**, for a library whose entire value proposition is exact tiktoken parity.

Meanwhile `scripts/generate_fixtures.py` exists and generates a proper 10-case multilingual fixture including CJK, emoji, and combining marks — and **no Go file reads it**:

```
$ grep -rn "fixture\|golden\|testdata\|\.json" --include=*.go .
(no results)
```

The fixture infrastructure was built and then not wired up.

### 6.6 Other smells

- `EncodeSingleToken` (`tokenizer.go:254`) reaches directly into `t.core.Encoder` and `t.core.SpecialTokensEncoder` — exported fields of an internal struct. Works, but `CoreBPE` exports 7 mutable fields with no accessor discipline; a future refactor of `RankIndex` breaks the facade.
- `Decode` returns `string(bytes)` (`tokenizer.go:288`) with no note that a truncated token sequence yields invalid UTF-8. tiktoken distinguishes `decode` (lossy, with replacement chars) from `decode_bytes`. Here `DecodeBytes` exists but `Decode` is silently lossy-by-omission.
- `TokenByteValues()` (`tokenizer.go:305`) allocates ~200,000 fresh `[]byte` on every call, several MB. No caching, no doc comment warning.
- `validateAndCopyConfig` returns `(map, slice, map, error)` — a 4-value return that would read better as a small struct.
- Zero doc comments on exported symbols. `golint`/`revive` would flag every one; `pkg.go.dev` will render an API reference with no prose.

---

## 7. PERFORMANCE ANALYSIS

### 7.1 The streaming matcher is not used on the main path

CHANGELOG v0.6.1 claims: *"Hybrid Regex Matcher: Optimized matcher strategy using `FindAllStringIndex` for short text (<2048B) and streaming `MatchIterator` for long documents."*

That hybrid lives in `EncodeOrdinarySequential` (`encode.go:105`) and `CountOrdinarySequential` (`encode.go:143`). Call-graph:

- `EncodeOrdinarySequential` ← `EncodeOrdinaryBatchNative` only
- `CountOrdinarySequential` ← `CountOrdinaryBatchNative` only
- `Tokenizer.EncodeOrdinary` → `EncodeOrdinaryNative` → `EncodeSubTextMatches` → **`FindAllStringIndex(text, -1)`**
- `Tokenizer.Encode` → `EncodeNative` → `EncodeSubTextMatches` → **`FindAllStringIndex(text, -1)`**
- `Tokenizer.Count` → `EncodeOrdinary` → same

So the single-document path — including the README's headline "Full Document UDHR (3.4 MB)" benchmark — uses full materialization. Memory: ~800k pieces × `[]int` slice header (24 B) + 2-int backing array (16 B) ≈ **32 MB of index metadata**, live simultaneously, on top of the ~1M-element output `[]int`.

Fix: route `EncodeOrdinaryNative` through the same size check as `EncodeOrdinarySequential`, or better, delete `EncodeOrdinarySequential` and make `EncodeSubTextMatches` itself hybrid.

### 7.2 "Zero-Allocation Token Counting" is not zero-allocation on the single-text path

CHANGELOG v0.6.1 claims: *"Zero-Allocation Token Counting: Implemented `Count` and `CountOrdinaryBatch` reducing memory allocation by ~19.3%."*

```go
func (t *Tokenizer) Count(text string) (int, error) {
    tokens, err := t.EncodeOrdinary(text)   // tokenizer.go:272 — allocates the full []int
    if err != nil { return 0, err }
    return len(tokens), nil                 // then throws it away
}
```

`Count` **fully encodes and discards**. The genuinely allocation-free machinery — `CountPieceToWithIndex` (`bpe.go:214`), which runs the merge loop and returns `len(parts)-1` without ever building an output slice — is reachable **only** via `CountOrdinaryBatch`. `CountContext` inherits the same problem (`tokenizer.go:211`).

One-line fix:

```go
func (t *Tokenizer) Count(text string) (int, error) {
    return t.core.CountOrdinarySequential(text)
}
```

This also picks up the streaming matcher for free. Expect a large win on the count-only path — likely the dominant workload for a gateway. **[verify]**

### 7.3 O(n²) merge on unbounded pieces

Both patterns contain an unbounded letter run: `\p{L}+` (cl100k, `cl100k.go:3`) and `[\p{Ll}\p{Lm}\p{Lo}\p{M}]+` (o200k, `o200k.go:3`). Numbers are capped at `\p{N}{1,3}`, but letters are not.

A single 1 MB run of letters with no whitespace produces **one piece of 1,000,000 bytes**, fed into `bytePairMergeLoop`. With `findMinRank` scanning O(n) per merge and O(n) merges: **~10¹² operations**, plus O(n) memmove per merge from the `append` splice. `makePartsBuffer` heap-allocates a `[]part` of 1,000,001 × 16 bytes = **16 MB** for that single piece.

`security_test.go:88` tests `bytes.Repeat([]byte("a"), 100000)` — 100 KB, which at n² is ~10¹⁰ and probably already takes seconds. The test has no timeout and only asserts no panic, so a 30-second runtime passes silently. **[verify]** — time this test; then try 1 MB.

Mitigations, in order of preference: (a) a max-piece-length guard that splits oversized pieces at a safe boundary; (b) replace `findMinRank`'s linear scan with a min-heap or doubly-linked-list-with-priority structure to get O(n log n); (c) document a hard input-size limit.

### 7.4 Reconstruction cost per `GetEncoding` call

Covered in §4.2 — the dominant performance defect. Fix:

```go
var (
    encCache   = map[Encoding]*Tokenizer{}
    encCacheMu sync.Mutex
    encOnce    = map[Encoding]*sync.Once{}
)
```

or simply three package-level `sync.Once` + cached pointers. Since `Tokenizer` is documented immutable and thread-safe, returning a shared pointer is sound. `WithOptions` already returns a copy, so per-caller configuration still works once options are actually wired up.

### 7.5 Redundant validation pass at construction

`ValidateTokenIDs` (`core.go:37`) builds a throwaway `map[int]string` of ~200k entries to detect duplicate IDs. For the embedded path, `openai.ParseVocabulary` has *already* enforced token-uniqueness and non-negativity (`vocabulary.go:33-45`), and `buildDecoderSlice` has already produced a slice that would reveal collisions. Three passes over 200k entries where one would do. Once §7.4 is fixed this becomes a one-time cost and matters much less.

### 7.6 Capacity heuristics

- `encode.go:390,419`: `make([]int, 0, (len(text)+2)/3)` — assumes ~3 bytes/token. English is ~4; this over-allocates ~33%. CJK in UTF-8 is ~1.5–2 bytes/token; this under-allocates and forces regrowth precisely where texts are large.
- `decode.go:6`: `make([]byte, 0, len(tokens)*4)` — reasonable.
- `bpe.go:40`: `make(map[uint64]int32, len(ranks)/4)` — for o200k, tokens of length 3–7 are far more than 25% of the vocabulary, so this under-sizes and forces multiple map growths during construction.

### 7.7 Worker cap ignores the machine

`maxBatchWorkers = 4` (`encode.go:103`) caps batch parallelism at 4 goroutines regardless of `GOMAXPROCS`. On a 32-core server, batch throughput is capped at 12.5% of available parallelism. This is exactly what `Options.MaxWorkers` was meant to control (§3.4).

---

## 8. SECURITY REVIEW

This is a parsing library that ingests untrusted text by design. Injection (SQL/command/template) and authn/authz are out of scope — there is no I/O, no `os/exec`, no network, no deserialization of untrusted structured data. `NewFromFile` (`tokenizer.go:126`) takes a caller-supplied path but is an explicit, caller-driven operation. The real attack surface is **resource exhaustion** and **semantic divergence from the reference implementation**.

### 8.1 CRITICAL — Algorithmic complexity DoS via long unbroken letter runs

**Vector:** POST a prompt containing a single 1 MB run of letters with no whitespace or punctuation.
**Mechanism:** §7.3 — one piece of length n through an O(n²) merge loop, plus a 16 MB `[]part` allocation.
**Impact:** For a gateway calling `CountForModel` on inbound prompts, one request pins a core for a very long time and there is **no timeout, no size limit, and no context check** that can interrupt it (see 8.4). A handful of concurrent requests exhausts the worker pool. Classic amplification: kilobytes of request → seconds of CPU.
**Mitigation:** input length limit at the API boundary + a piece-length guard in `BytePairEncodeToWithIndex` + a better merge structure.

### 8.2 HIGH — `Encode` silently tokenizes special tokens as ordinary text

Reference `tiktoken`'s `Encoding.encode()` defaults to `disallowed_special="all"` and **raises** if the input contains `<|endoftext|>` or any other special token. This is a deliberate safety default: it prevents a user-supplied string from injecting a control token into a prompt.

Here:

```go
func (t *Tokenizer) Encode(text string) ([]int, error) {
    tokens, _, err := t.core.EncodeNative(text, nil)   // tokenizer.go:188 — allowed = nil
    return tokens, err
}
```

`allowed == nil` → `FindNextAllowedSpecial` returns `(SpecialMatch{}, false, nil)` immediately (`encode.go:27`) → the special-token scan never runs → `<|endoftext|>` is tokenized as ordinary text. `ErrDisallowedSpecial` is declared in `errors.go:8` and **never returned by any code path**.

Two consequences:

1. **No safety check.** A caller who reads `Encode` as "the safe one" (the tiktoken convention) gets no protection. There is also no exported way to opt *into* special-token handling — `EncodeNative`'s `allowed` parameter is unreachable from outside `internal/bpe`.
2. **Count divergence.** The upstream model server *will* tokenize `<|endoftext|>` as a single special token. This library counts it as ~6 ordinary tokens. A user can therefore make the gateway's count disagree with the provider's — a budget/rate-limit bypass, small per instance but deterministic and repeatable.

**Fix:** `Encode` should return `ErrDisallowedSpecial` when a special token appears, and add `EncodeAllowingSpecial(text string, allowed []string)` to expose the working machinery in `EncodeNative`. Note this is an API-breaking change against the frozen v1.0.0 signature — which is itself an argument that v1.0.0 was declared too early.

### 8.3 HIGH — Unbounded allocation from `Config.MergeableRanks`

`tokenizer.go:66`:

```go
decoderSlice := make([]string, maxRank+1)
```

`maxRank` is the maximum value in a caller-supplied `map[string]int`. Validation rejects negatives (`tokenizer.go:42`) and duplicates, but has **no upper bound**. A single entry `{"x": 2147483647}` in an otherwise valid config causes a `make([]string, 2147483648)` — **32 GB** on 64-bit (16 bytes per string header). Immediate OOM kill, not a recoverable error.

The identical pattern exists in `openai.buildDecoderSlice` (`vocabulary.go:47-58`), reachable from `NewFromVocabulary`/`NewFromFile` with any caller-supplied `io.Reader`. A 30-byte `.tiktoken` file whose single line has rank `2147483647` OOMs the process.

**Fix:** reject `maxRank > len(ranks) * someFactor` (a dense vocabulary should have max rank ≈ vocabulary size), or cap at a configurable ceiling, or build the decoder as a `map[int]string` when density is low.

### 8.4 HIGH — No `regexp2` match timeout, and context cancellation is decorative

`core.go:91` and `core.go:80` compile with `regexp2.None` and **never set `MatchTimeout`**:

```
$ grep -rn "MatchTimeout" --include=*.go .
(no results)
```

`regexp2` is a **backtracking** engine (that is why the project uses it — Go's RE2 lacks lookahead, needed for `\s+(?!\S)`). Backtracking engines have no inherent runtime bound; `regexp2` provides `MatchTimeout` precisely for this. Leaving it unset means a pathological pattern/input pair runs forever with no way to stop it.

For the two built-in patterns the generated engines (`regexp2_codegen_*.go`) are atomic-grouped and unlikely to blow up. But `New(Config)` and `NewFromVocabulary` accept an **arbitrary caller-supplied pattern** (`tokenizer.go:102`, `:114`) which is compiled with the same settings. A gateway that lets tenants define custom encodings has a direct ReDoS path.

Compounding this: the context APIs cannot rescue you.

```go
func (t *Tokenizer) EncodeContext(ctx context.Context, text string) ([]int, error) {
    if err := checkContext(ctx); err != nil { return nil, err }   // ONE check, then...
    return t.EncodeOrdinary(text)                                  // ...uninterruptible
}
```

`checkContext` is a single non-blocking `select` **before** work begins. Once `EncodeOrdinary` starts on a 3.4 MB document (or the 8.1 pathological input), the context is never consulted again. The README's "**Production Control**: Context cancellation APIs … for AI Gateways" describes a pre-flight deadline check, not cancellation. `EncodeOrdinaryBatchContext` is better — it polls every 10 items (`tokenizer.go:220`) — but still cannot interrupt a single long item.

**Fix:** set `MatchTimeout` on compile; thread `ctx` into `EncodeSubTextMatches` and poll every N pieces inside the piece loop.

### 8.5 MEDIUM — `EncodeOrdinaryBatchContext` is a silent 4× performance downgrade

```go
func (t *Tokenizer) EncodeOrdinaryBatchContext(ctx, texts) ([][]int, error) {
    res := make([][]int, len(texts))
    for i, txt := range texts {          // tokenizer.go:219 — strictly sequential
        ...
        tokens, err := t.EncodeOrdinary(txt)
    }
}
```

The non-context `EncodeOrdinaryBatch` fans out to 4 goroutines. The context variant does **not** — it reimplements the batch loop serially. A user following the README's "Production Control" advice and switching to the context API loses all batch parallelism with no indication. Same for `CountOrdinaryBatchContext`, which additionally loses the allocation-free count path by going through `Count` (§7.2).

### 8.6 MEDIUM — Nondeterministic tokenizer selection

§3.3. Beyond correctness, this is a security-relevant nondeterminism: two replicas of the same gateway binary, same config, same input, can enforce different token limits for `ft:gpt-4o*` models. Anything that depends on reproducible counting (audit logs, billing reconciliation, cache keys) breaks.

### 8.7 LOW — Silent rank truncation

`NewRankIndex` skips tokens with `rank > math.MaxInt32` without error (`bpe.go:50-52`). Those tokens become unencodable, silently. A custom vocabulary hitting this produces wrong output with no diagnostic.

### 8.8 LOW — Process/policy

- `SECURITY.md` instructs reporters to use **GitHub Issues** — i.e. public disclosure. Should be GitHub Private Vulnerability Reporting or a security email. It also promises "acknowledged within 24 hours" and "a patch will be released promptly," which are commitments most solo-maintained projects cannot keep.
- `LICENSE` contains the literal two words `MIT License` — **no license body, no copyright holder, no year**. This is not a valid MIT grant. Corporate license scanners (FOSSA, Snyk, Black Duck) will flag it as unlicensed/unknown, and legally the terms are unstated. `NOTICE` says `Copyright (c) 2026` with no holder named. **Paste the full MIT text with a named copyright holder** — this is a five-minute fix blocking enterprise adoption.

### 8.9 Not findings (checked, clean)

- No SQL, no `exec`, no `text/template`, no `net/http`, no `encoding/gob`/`json` unmarshalling of untrusted input.
- Map copying in `validateAndCopyConfig` (`tokenizer.go:40,72`) genuinely prevents post-construction mutation by the caller; `TestCustomEncodingSuccessAndImmutability` verifies it properly.
- Byte-coverage validation (0–255 required, `tokenizer.go:59-63`) correctly prevents unencodable-byte panics. `security_test.go` covers this well.
- `DecodeNative` bounds-checks token IDs before slice indexing (`decode.go:8`).
- Thread-safety claim holds: `CoreBPE` is written only during construction and read-only afterward; batch workers write to disjoint indices.

---

## 9. SCALABILITY ANALYSIS

**Concurrency model:** fork-join per call. No worker pool, no goroutine reuse, no backpressure. Each `EncodeOrdinaryBatch` spawns up to 4 fresh goroutines and joins on `sync.WaitGroup`.

**State:** `Tokenizer`/`CoreBPE` are immutable post-construction — genuinely stateless for horizontal scaling. `globalRegistry` is a process-global but is also immutable. Nothing prevents running N replicas.

### What breaks first at 10×

1. **Memory and GC, from missing encoder caching (§4.2).** This is the first and by far the largest wall. At 1,000 req/s through `CountForModel`, the process allocates ~200,000 objects × 1,000/s = 2×10⁸ allocations/sec and tens of GB/s of garbage. The GC will not keep up; the process will thrash or OOM long before CPU becomes the limit. **Everything else in this section is downstream of fixing this.**

2. **Fixed 4-worker cap (§7.7).** Once caching is fixed, batch throughput plateaus at 4 cores. Adding cores 5–32 does nothing. The knob to fix it (`Options.MaxWorkers`) exists but is disconnected.

3. **Peak memory on large documents (§7.1).** ~32 MB of transient index metadata per concurrent 3.4 MB document, all live at once because of `FindAllStringIndex`. 20 concurrent large documents ≈ 640 MB of pure index overhead. The streaming iterator that fixes this already exists and is unreachable from the main path.

4. **Uncontrolled goroutine growth.** With no pool, 1,000 concurrent batch calls spawn 4,000 goroutines. Each `EncodeOrdinarySequential` allocates its own `ret` slice. No admission control, no queue, no way for a caller to bound total concurrency.

5. **Adversarial input (§8.1).** A single crafted request occupies a core for seconds with no timeout. At any scale, this is a one-request denial of service.

**Not a bottleneck:** the `RankIndex` lookup design. The 3-tier structure is genuinely good and will not be the limiting factor.

**Missing entirely:** any observability. No metrics, no counters, no tracing hooks. At 10× you cannot tell *which* of the above is failing because the library emits nothing.

---

## 10. RELIABILITY & ERROR HANDLING

### 10.1 Data race in batch error capture — **[verify with `go test -race`]**

`encode.go:237-264` (and the identical copy at `:302-329`):

```go
var errOnce sync.Once
var firstErr error                       // plain variable

for w := 0; w < numPartitions; w++ {
    go func(start, end int) {
        ...
        errOnce.Do(func() { firstErr = err })   // written inside goroutine
    }(startIdx, endIdx)
}
wg.Wait()
if firstErr != nil { ... }                      // read after Wait
```

`sync.Once.Do` guarantees the function body runs exactly once and establishes happens-before between the `Do` call and subsequent `Do` calls — and `wg.Wait()` establishes happens-before with all `wg.Done()` calls. So the *read after `Wait`* is in fact correctly ordered, and `-race` will likely be clean. **But the pattern is fragile by construction**: it relies on the `Wait` barrier rather than on the variable being safely published. A single future refactor that reads `firstErr` before `Wait` — e.g. to implement early cancellation, which this code will want — introduces a real race silently. Use `atomic.Pointer[error]`, an error channel, or `golang.org/x/sync/errgroup` (which also gives you the context cancellation that §8.4 needs).

Note also: on error, a worker `return`s and leaves the remaining `results[i]` entries as nil. The caller gets `nil, firstErr`, so this is not observable today — but combined with the above, it's a second thing that only works by accident of the current control flow.

### 10.2 CI is broken

`.github/workflows/ci.yml:22`:

```yaml
- name: Run Fuzz Smoke Test
  run: go test -fuzz=FuzzEncodeDecode -fuzztime=5s .
```

The only fuzz target in the repository is `FuzzEncodeOrdinary` (`security_test.go:109`). There is no `FuzzEncodeDecode`:

```
$ grep -rn "func Fuzz" --include=*.go .
./security_test.go:109:func FuzzEncodeOrdinary(f *testing.F) {
```

`go test -fuzz=<no match>` errors with *"no fuzz tests matching pattern"* and exits non-zero. So either the CI has been red since this workflow was added, or it has never run on a PR. Either way **the fuzzing gate provides zero protection.** **[verify]** by checking Actions history.

Other CI gaps: no `go vet`, no `staticcheck`, no `gofmt -l` check, no build matrix despite README claiming Linux amd64/arm64 + macOS arm64/amd64 + Windows amd64 support (CI runs `ubuntu-latest` only), no coverage reporting, no benchmark regression gate despite performance being the project's headline claim.

### 10.3 Error handling quality

Good: consistent `%w` wrapping; `errors.Is` works against all sentinels; `parseVocabLine` and `validateAndInsert` return errors rather than panicking; `DecodeNative` bounds-checks.

Gaps:

- `internal/openai/vocabulary.go:63` — `bufio.Scanner` is created with the default 64 KB `MaxScanTokenSize` and **`scanner.Err()` is never checked** after the loop. A line longer than 64 KB (or any read error) causes the loop to stop silently, and `ParseVocabulary` returns a **partial vocabulary with `nil` error**. A truncated or corrupted `.tiktoken` file passed to `NewFromFile` yields a silently-wrong tokenizer. This is a real correctness bug with a two-line fix.
- `parseVocabLine:20` returns bare `fmt.Errorf("invalid line format")` — no line number, no content, no sentinel. Debugging a malformed vocab file means bisecting by hand.
- No retry/circuit-breaker logic anywhere — correctly so, there is nothing to retry.
- Zero logging, zero metrics, zero tracing hooks. For a library this is defensible; for one positioned at the center of an AI gateway it means operators have no visibility into which encoding was selected, how long counting took, or how often fallback fired.

### 10.4 Partial-failure resilience

None needed at the I/O level (no I/O). At the API level: a batch call that fails on item 3 of 8,000 discards all 8,000 results. For a gateway batching independent user requests, a per-item `[]error` (or `[]Result{Tokens, Err}`) would be far more useful than all-or-nothing.

---

## 11. DEPENDENCY ANALYSIS

The dependency graph is exactly one node deep. That is a genuine strength worth preserving.

### `github.com/dlclark/regexp2/v2 v2.5.2`

**Why:** required, not optional. Both tiktoken patterns use negative lookahead `\s+(?!\S)`, which Go's RE2-based `regexp` cannot express. Every Go tiktoken port reaches for `regexp2` for this reason.

**Quality:** the de-facto standard .NET-regex port for Go, actively maintained, widely used. v2 with `RegisterEngine`/codegen support is recent.

**Risk — this is the important part:** `regexp2` is a **backtracking** engine. That is a fundamentally different security model from RE2's linear-time guarantee. The library compensates via generated atomic-group engines for the two built-in patterns, but:

- `MatchTimeout` is never set (§8.4) — the mitigation the dependency itself provides is unused.
- Arbitrary caller patterns via `New(Config)` get no protection at all.
- The generated-engine optimization is keyed by **exact pattern string match** (`regexp2_codegen_cl100k.go:425`). Drift between `openai.PatternCL100K` and the registered literal silently degrades to the interpreter. No test guards this. Recommendation: add a test asserting the registration key equals `openai.PatternCL100K`, or generate the constant and the registration from one source.

**Version pinning:** `go.sum` pins v2.5.2 with hashes. Fine. No `dependabot.yml` or Renovate config, so security updates are manual. No `govulncheck` in CI.

### Standard library usage

All appropriate: `encoding/base64`, `bufio`, `strconv` for vocab parsing; `sync`/`runtime` for concurrency; `unicode/utf8` for the byte cursor; `embed` for vocabularies; `regexp` used *only* for `QuoteMeta` in `core.go:78` (correct — it does not compile anything).

### `go.mod` issue

```
go 1.26.5
```

README states *"Requires Go 1.21+ (for automatic Profile-Guided Optimization support)"*. The `go` directive says otherwise: since Go 1.21, this line is a **minimum requirement**, and a toolchain older than 1.26.5 will refuse to build (or auto-download a newer toolchain, if enabled). The README is wrong, and pinning to a specific *patch* release is unusually restrictive — `go 1.26` is the conventional form. If 1.21+ really is the intent, set `go 1.21`.

### Module distribution weight

The module ships 5.2 MB of embedded vocabulary (`o200k_base.tiktoken` 3.5 MB + `cl100k_base.tiktoken` 1.7 MB) plus three copies of `default.pgo` (20 KB each, in `/`, `internal/bpe/`, and `test/`). Only the root `default.pgo` is used by the Go toolchain; the other two are dead weight. Every `go get` downloads all of it. The 5.2 MB is the deliberate price of zero-CGO/zero-download, and is defensible — the duplicated PGO files are not.

(Confirmed clean: `cpu.pprof` and the 10 MB `test.test` binary in the working tree are **not** tracked by git — `.gitignore` handles them correctly. `.idea/` is likewise untracked.)

---

## 12. IMPROVEMENT SUGGESTIONS

Prioritized. P0 items should block any production use.

### P0 — Correctness and availability

| # | Change | Location | Why |
|---|---|---|---|
| 1 | **Cache constructed encodings** behind `sync.Once` (one per encoding) | `embed.go`, `tokenizer.go:135` | Removes ~200k allocations per `GetEncoding`/`ForModel`/`CountForModel` call. Single highest-value change in the repo (§4.2). |
| 2 | **Fix nondeterministic prefix matching** — sort prefixes longest-first into a slice instead of ranging a map | `tokenizer.go:160` | `ft:gpt-4o*` currently resolves to a different encoding across process restarts (§3.3). |
| 3 | **Unify the two model registries** — delete one, or make `EncodingForModel` delegate to `ResolveModel` | `tokenizer.go:156`, `registry.go` | `ForModel` and `CountForModel` currently disagree on ≥4 real model names (§3.2). |
| 4 | **Bound `maxRank`** before `make([]string, maxRank+1)` | `tokenizer.go:66`, `vocabulary.go:54` | Prevents a 32 GB allocation from one map entry (§8.3). |
| 5 | **Check `scanner.Err()`** after the parse loop | `vocabulary.go:76` | Truncated vocab files currently produce a silently-wrong tokenizer with nil error (§10.3). |
| 6 | **Fix the CI fuzz target name** (`FuzzEncodeDecode` → `FuzzEncodeOrdinary`) | `.github/workflows/ci.yml:22` | The fuzzing gate has never protected anything (§10.2). |
| 7 | **Write a real MIT LICENSE file** with body and named copyright holder | `LICENSE` | Currently not a valid license grant (§8.8). |

### P1 — Security hardening

| # | Change | Location |
|---|---|---|
| 8 | Set `regexp2.Regexp.MatchTimeout` after compile (e.g. 5s, configurable) | `core.go:91,80` |
| 9 | Guard piece length in `BytePairEncodeToWithIndex`; reject or split pieces above a threshold (e.g. 32 KB) | `bpe.go:188` |
| 10 | Make `Encode` return `ErrDisallowedSpecial` on special tokens; add `EncodeAllowingSpecial(text, allowed)` | `tokenizer.go:187` |
| 11 | Thread `ctx` into the piece loop; poll every ~1024 pieces | `encode.go:333`, `tokenizer.go:200` |
| 12 | Return an error instead of silently skipping ranks `> MaxInt32` | `bpe.go:50` |
| 13 | Switch `SECURITY.md` to private vulnerability reporting | `SECURITY.md` |
| 14 | Add `govulncheck` + `go vet` + `staticcheck` to CI | `ci.yml` |

### P2 — Performance

| # | Change | Location |
|---|---|---|
| 15 | `Count` → call `core.CountOrdinarySequential` directly instead of encode-and-discard | `tokenizer.go:271` |
| 16 | Route `EncodeOrdinaryNative` through the hybrid short/long matcher | `encode.go:418,333` |
| 17 | Wire `Options` through to `CoreBPE` — or delete `Options`/`WithOptions` entirely | `tokenizer.go:168`, `encode.go` |
| 18 | Make `EncodeOrdinaryBatchContext`/`CountOrdinaryBatchContext` use the parallel path (via `errgroup`) | `tokenizer.go:214,234` |
| 19 | `RankIndex` by pointer, not value, in `CoreBPE` | `core.go:22`, `bpe.go:38` |
| 20 | Size `shortPacked` from an actual count of 3–7-byte tokens, not `len(ranks)/4` | `bpe.go:40` |
| 21 | Replace `findMinRank`'s linear scan with a heap → O(n log n) merges | `bpe.go:139` |
| 22 | Delete the duplicate `default.pgo` copies in `internal/bpe/` and `test/` | repo root |

### P3 — Maintainability and testing

| # | Change |
|---|---|
| 23 | **Wire up `scripts/generate_fixtures.py` output as a golden-file test.** Commit `testdata/cl100k.json` + `testdata/o200k.json` and assert exact token vectors for all 10 cases × both encodings. This is the single most valuable *test* change — parity is the product. |
| 24 | Add a test asserting the codegen registration key `==` `openai.PatternCL100K` / `PatternO200K` |
| 25 | Deduplicate `EncodeOrdinaryBatchNative`/`CountOrdinaryBatchNative` and the two `*Sequential` functions with generics |
| 26 | Fix or delete `lastLen` (§6.4) |
| 27 | Delete or clearly mark as experimental: `Engine`, `EngineMetadata`, `Algorithm`, `Capabilities`, unimplemented `Provider`s, `Calibration`, `WithCalibratedFallback`, `WithHeuristicFallback`, unused `ModelSpec` fields |
| 28 | Return `ErrUnknownEncoding` with a helpful message for `p50k_base`/`r50k_base`/`gpt2` models, which are in `ModelToEncoding` but unsupported by `GetEncoding` |
| 29 | Add doc comments to all exported symbols |
| 30 | Correct README: Go version, "Resource Options", "zero-allocation Count", "context cancellation", and the untranslated *"Terhambat Python GIL"* cell |
| 31 | `sort.SliceStable` + lexical tiebreak for registry prefixes (`registry.go:66`) |
| 32 | Build matrix in CI matching the OS/arch support claimed in README |

---

## 13. REFACTORING EXAMPLES

### 13.1 Encoding cache — the highest-value change

**Current** (`embed.go:17-29`, `tokenizer.go:135-146`) — full reconstruction every call:

```go
func getEmbeddedCL100K() (*Tokenizer, error) {
	return NewFromVocabulary(strings.NewReader(cl100kVocab), openai.PatternCL100K, openai.SpecialTokensCL100K())
}

func GetEncoding(name Encoding) (*Tokenizer, error) {
	switch name {
	case CL100KBase:
		return getEmbeddedCL100K()
	case O200KBase:
		return getEmbeddedO200K()
	case O200KHarmony:
		return getEmbeddedO200KHarmony()
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnknownEncoding, name)
	}
}
```

**Refactored:**

```go
type cachedEncoding struct {
	once sync.Once
	tok  *Tokenizer
	err  error
}

var encodingCache = map[Encoding]*cachedEncoding{
	CL100KBase:   {},
	O200KBase:    {},
	O200KHarmony: {},
}

// build must not be a method on cachedEncoding so the switch stays in one place.
func buildEncoding(name Encoding) (*Tokenizer, error) {
	switch name {
	case CL100KBase:
		return NewFromVocabulary(strings.NewReader(cl100kVocab),
			openai.PatternCL100K, openai.SpecialTokensCL100K())
	case O200KBase:
		return NewFromVocabulary(strings.NewReader(o200kVocab),
			openai.PatternO200K, openai.SpecialTokensO200K())
	case O200KHarmony:
		return NewFromVocabulary(strings.NewReader(o200kVocab),
			openai.PatternO200K, openai.SpecialTokensO200KHarmony())
	}
	return nil, fmt.Errorf("%w: %s", ErrUnknownEncoding, name)
}

// GetEncoding returns a shared, immutable Tokenizer for the named encoding.
// The returned value is safe for concurrent use and is constructed at most once
// per encoding per process. Callers needing distinct Options should use WithOptions,
// which returns a copy sharing the same underlying vocabulary.
func GetEncoding(name Encoding) (*Tokenizer, error) {
	c, ok := encodingCache[name]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnknownEncoding, name)
	}
	c.once.Do(func() { c.tok, c.err = buildEncoding(name) })
	return c.tok, c.err
}
```

**Why it's better.** The map is written once at package init and only read afterward, so no mutex is needed on the map itself; `sync.Once` handles the per-entry construction race. Cost drops from ~200k allocations per call to ~200k allocations *per process*. The doc comment makes the shared-pointer contract explicit — which is honest, since `Tokenizer` was already documented as immutable. `WithOptions` still returns a copy, so per-caller configuration is unaffected. Note this makes construction errors sticky, which is correct for embedded assets that cannot change at runtime.

### 13.2 Deterministic prefix resolution

**Current** (`tokenizer.go:156-166`) — map iteration, nondeterministic:

```go
func EncodingForModel(model string) (Encoding, error) {
	if enc, ok := openai.ModelToEncoding[model]; ok {
		return Encoding(enc), nil
	}
	for prefix, enc := range openai.ModelPrefixToEncoding {   // RANDOM ORDER
		if strings.HasPrefix(model, prefix) {
			return Encoding(enc), nil
		}
	}
	return "", fmt.Errorf("%w: %s", ErrUnknownModel, model)
}
```

**Refactored** — precompute a longest-first ordering once:

```go
// in internal/openai/models.go

type prefixRule struct {
	prefix   string
	encoding string
}

// sortedPrefixes is ordered longest-prefix-first, then lexically, so that
// resolution is deterministic and "ft:gpt-4o" wins over "ft:gpt-4".
var sortedPrefixes = buildSortedPrefixes()

func buildSortedPrefixes() []prefixRule {
	rules := make([]prefixRule, 0, len(ModelPrefixToEncoding))
	for p, e := range ModelPrefixToEncoding {
		rules = append(rules, prefixRule{p, e})
	}
	sort.Slice(rules, func(i, j int) bool {
		if len(rules[i].prefix) != len(rules[j].prefix) {
			return len(rules[i].prefix) > len(rules[j].prefix)
		}
		return rules[i].prefix < rules[j].prefix // total order: no ties
	})
	return rules
}

func EncodingNameForModel(model string) (string, error) {
	if enc, ok := ModelToEncoding[model]; ok {
		return enc, nil
	}
	for _, r := range sortedPrefixes {
		if strings.HasPrefix(model, r.prefix) {
			return r.encoding, nil
		}
	}
	return "", fmt.Errorf("could not automatically map %s to a tokenizer", model)
}
```

**Why it's better.** `ft:gpt-4o:acme::abc123` now deterministically resolves to `o200k_base` on every run, because the 9-character prefix is tested before the 8-character one. The secondary lexical comparator makes the sort a **total order**, so `sort.Slice`'s instability cannot bite; the same fix should be applied to `registry.go:66`. Sorting happens once at package init rather than per call, and the 17-entry linear scan is faster than the map range it replaces.

### 13.3 Allocation-free `Count`

**Current** (`tokenizer.go:271-277`) — encode everything, then throw it away:

```go
func (t *Tokenizer) Count(text string) (int, error) {
	tokens, err := t.EncodeOrdinary(text)
	if err != nil {
		return 0, err
	}
	return len(tokens), nil
}
```

**Refactored:**

```go
// Count returns the number of tokens in text without materializing them.
func (t *Tokenizer) Count(text string) (int, error) {
	return t.core.CountOrdinarySequential(text)
}
```

**Why it's better.** `CountOrdinarySequential` (`encode.go:143`) already exists, already routes through `CountPieceToWithIndex` — which runs the same merge loop but returns `len(parts)-1` instead of appending to an output slice — and already uses the streaming `MatchIterator` for texts ≥ 2048 bytes. Three wins from one line: the `[]int` output allocation disappears, the `FindAllStringIndex` full materialization disappears for large inputs, and the CHANGELOG's "zero-allocation token counting" claim becomes true on the single-text path. `TestCountOrdinaryEquivalence` (`tokenizer_test.go:44`) already asserts `Count(x) == len(EncodeOrdinary(x))` across 9 multilingual cases, so this refactor is covered by an existing test — rename it, don't delete it.

---

## 14. TECHNICAL DEBT ANALYSIS

### 14.1 Debt taken deliberately, in visible order

The git history reads as a compressed sprint: 20+ commits all dated 2026-08-06, moving `v0.5.0 → v0.6.1 → v0.6.2 → v0.7.0 → v0.8.0 → v0.9.0 → v1.0.0` in a single day, then adding the entire multi-provider type surface (`Engine`, `Capabilities`, `Provider`, `Registry`, `Calibration`, `CountForModel`) in **six commits *after* the `v1.0.0 Stable Public API` commit**.

That ordering is the root of most of the debt in this report. Declaring "v1.0.0 freezes all public API signatures" (README) and *then* adding a large speculative API means the freeze now protects code that was never designed against a real second provider.

### 14.2 The compounding item: a frozen API that is wrong

Several P0/P1 fixes are **breaking changes against the stated SemVer promise**:

- Making `Encode` reject special tokens (§8.2) changes behavior for existing callers.
- Fixing `EncodingForModel` for `ft:gpt-4o*` changes returned token counts.
- Unifying the registries changes which models error.
- Deleting `Engine`/`Capabilities`/unimplemented `Provider`s removes exported symbols.
- Making `WithOptions` actually work changes runtime behavior.

Each week of adoption raises the cost of these. The debt compounds on a schedule set by other people's `go.mod` files. **Recommendation: do these now, ship as v2.0.0 (or retract v1.0.0 as premature and republish v0.x), before the module accumulates dependents.** The alternative — carrying `Encode`'s unsafe default and the `ft:gpt-4o` nondeterminism forever — is much worse.

### 14.3 Documentation debt with a security shape

Three README claims are not merely inaccurate; each one causes a reader to *skip a safety measure they would otherwise take*:

| Claim | Reality | What the reader wrongly concludes |
|---|---|---|
| "Context cancellation APIs … for AI Gateways" | one pre-flight `select` (§8.4) | "long encodes are interruptible" → no external timeout added |
| "Configurable worker pools and byte thresholds" | `WithOptions` is a no-op (§3.4) | "I've tuned resource use" → no capacity planning done |
| "Zero-Allocation Token Counting" for `Count` | `Count` allocates the full token slice (§7.2) | "counting is cheap" → counting in a hot loop |

Plus: "Requires Go 1.21+" contradicted by `go.mod`'s `go 1.26.5`; benchmarks reported to 5 significant figures from an unstated number of runs with no variance; a multi-core comparison against a GIL-bound Python binding described as a "3.63x Multi-core Speedup" (that measures CPython's GIL, not this library's advantage); and an untranslated Indonesian phrase — *"Terhambat Python GIL"* — in an otherwise-English table.

### 14.4 Areas that will be painful to change

1. **The codegen ↔ pattern coupling.** `regexp2_codegen_cl100k.go:425` registers a hand-copied 90-character escaped pattern literal that must byte-match `openai.PatternCL100K`. There is no test, no build check, and no comment explaining the regeneration procedure — nor is the generator tool referenced anywhere in the repo. Anyone touching a pattern will silently lose the fast path, and 1,185 lines of generated code is a wall for a new contributor. Add the regeneration command to a `//go:generate` directive and assert the key in a test.

2. **The duplicated batch machinery** (§6.1). Every concurrency fix must be applied twice, correctly, in code that looks identical but isn't.

3. **Rune/byte index mixing** in `FindNextAllowedSpecial` / `MatchIterator` / `ByteCursor`. Correct today, but the ASCII fast paths mean a bug introduced here manifests **only on non-ASCII input**, which the current test suite barely covers with exact assertions.

4. **`internal/bpe.CoreBPE`'s seven exported mutable fields**, reached directly from `tokenizer.go:255,296,306`. Any change to `RankIndex`'s representation ripples into the public package.

### 14.5 Long-term maintainability risks

- **Parity has no regression net.** Two hardcoded token vectors from one ASCII string. The fixture generator exists and is unused. Any future optimization to the merge loop or `RankIndex` can silently change token output for CJK, emoji, combining marks, or invalid UTF-8 — the exact inputs where BPE ports historically diverge — and every test will still pass. This is the highest-severity *process* risk in the repo.
- **Vocabulary drift.** OpenAI adds models continuously. Four hand-maintained tables (§6.2) with no sync test means new models get added to one and not the other.
- **Performance claims with no gate.** The README leads with beating Rust `tiktoken`. Nothing in CI measures it. The claim will silently become false.

---

## 15. FINAL ASSESSMENT

### Strengths

- **The `RankIndex` three-tier lookup is genuinely well-engineered.** The `packKey` scheme (length in byte 0, content in bytes 1–7) is provably injective, and `TestPackedKeyCollisionCornerCases` probes exactly the right adversarial cases. The whole-piece fast path in `EncodePieceTo` short-circuits the merge loop for common words — a real, well-chosen optimization.
- **`ByteCursor`'s ASCII fast path** is the correct answer to regexp2's rune-indexed API, and the amortized forward walk avoids the obvious O(n²) trap.
- **The zero-CGO, zero-runtime-download design is a real differentiator.** 5.2 MB of embedded vocabulary buys deployment simplicity that CGO-based competitors cannot match.
- **Immutability discipline in the constructors is honest.** `validateAndCopyConfig` genuinely copies caller maps, and `TestCustomEncodingSuccessAndImmutability` mutates the originals afterward to prove it. The 0–255 byte-coverage validation is exactly the right precondition, and `security_test.go` exercises all 256 values individually.
- **Batch worker partitioning is race-free by construction** — disjoint index writes into a preallocated slice, no shared mutable state.
- **The `internal/` boundary is respected**, and the dependency graph is one node deep.
- **Adversarial-input testing exists at all**, including native Go fuzzing and invalid-UTF-8 cases. Most tokenizer ports have none.

### Weaknesses

- **No encoder caching.** Every `GetEncoding`/`ForModel`/`CountForModel` call rebuilds a 200k-entry vocabulary from scratch (§4.2). This makes the library's most convenient API unusable in the workload it advertises.
- **Two divergent model registries plus map-order prefix matching** produce inconsistent and nondeterministic model resolution (§3.2, §3.3).
- **Three headline README features do not work as described**: `WithOptions` is a no-op, context "cancellation" is a pre-flight check, `Count` is not allocation-free (§14.3).
- **Parity — the entire product thesis — is verified by two token vectors** from one two-word ASCII string, while a proper fixture generator sits unused (§6.5).
- **~30 exported symbols implement nothing**, frozen into a v1.0.0 SemVer promise (§5.7).
- **CI's fuzz gate targets a function that does not exist**, and there is no `vet`, no `staticcheck`, no `govulncheck`, no OS matrix (§10.2).
- **`LICENSE` contains no license text** (§8.8).

### Key Risks — top 3 if deployed as-is

**1. Vocabulary reconstruction on every call → memory exhaustion under any real traffic.**
`CountForModel` and `ForModel` rebuild ~200k map entries, a 256 KB `RankIndex`, and two compiled regexes per invocation. At meaningful request rates the process will thrash the GC or OOM long before CPU saturates. This is not a tuning problem; it makes the advertised gateway use case non-functional. *(§4.2 — fix: P0-1)*

**2. Nondeterministic and inconsistent model→encoding resolution → silently wrong token counts.**
`ft:gpt-4o*` models resolve to a different encoding depending on Go's randomized map iteration order, varying **across process restarts of the same binary**. Separately, `ForModel` and `CountForModel` disagree on `gpt-4.1`, `o4-mini`, `chatgpt-4o-*`, and `gpt-35-turbo`. Any system using this for billing, rate limiting, or context-window enforcement will produce counts that are wrong, irreproducible, and very hard to debug. *(§3.2, §3.3 — fix: P0-2, P0-3)*

**3. Uninterruptible O(n²) work on untrusted input → single-request denial of service.**
A 1 MB unbroken letter run becomes one piece in an O(n²) merge loop with a 16 MB `[]part` allocation, no `regexp2.MatchTimeout`, no piece-length guard, and no in-flight context check to abort it. Kilobytes of request buy seconds of pinned CPU, and the "context cancellation" API cannot stop it. *(§7.3, §8.1, §8.4 — fix: P1-8, P1-9, P1-11)*

### Scores

| Dimension | Score | Rationale |
|---|---|---|
| Architecture Quality | **5/10** | Clean `internal/` layering and a stateless, immutable facade — undercut by two competing registries, a disconnected `Options` struct, and a large speculative abstraction layer with no implementations behind it. |
| Code Quality | **6/10** | Idiomatic, small functions, consistent `%w` wrapping, good naming. Loses points for 63 lines of copy-pasted batch logic, a return value with two incompatible meanings, zero doc comments, and several tests that cannot fail. |
| Security | **4/10** | Excellent input-validation hygiene at construction (byte coverage, duplicate ranks, immutability) sits next to unbounded allocation from a single map entry, no regex timeout on a backtracking engine, an O(n²) DoS with no interrupt, and an `Encode` that silently drops tiktoken's special-token safety default. |
| Scalability | **5/10** | Genuinely stateless and horizontally scalable, with sound race-free batch partitioning — but capped at 4 workers regardless of core count, holding ~32 MB of transient index metadata per large document, and gated behind the caching defect that dominates everything else. |
| **Overall Engineering Quality** | **5/10** | **A competent, fast BPE core wrapped in a public API that over-promises.** The tokenizer engine is the strong part; the packaging around it — caching, model resolution, configuration, documentation, and parity testing — is where the risk concentrates. Fix the seven P0 items and this becomes a 7–8. |

---

### Suggested order of work

1. P0-1 (caching) — largest single improvement, non-breaking, ~30 lines.
2. P0-6 (CI fuzz name) and P0-7 (LICENSE) — five minutes each, unblock everything else.
3. P3-23 (golden-file parity tests from the existing fixture generator) — **do this before any other behavioral change**, so subsequent fixes are provably safe.
4. P0-2, P0-3 (model resolution) — breaking; batch with the v2 decision in §14.2.
5. P0-4, P0-5 (unbounded allocation, scanner error) — small, non-breaking, high severity.
6. P1 security items, then P2 performance.
7. Rewrite the README against what the code actually does.

**Before acting on the performance findings, run `go build ./... && go vet ./... && go test -race ./...` and benchmark `GetEncoding`, `Count`, and a 1 MB single-word input.** No Go toolchain was available in this analysis environment, so every performance claim above is derived from reading the call graph rather than measurement — the reasoning is stated explicitly in each case so it can be checked.

---

## 16. MULTI-PROVIDER SUBSYSTEM — OFFLINE-ONLY DESIGN WITH NEAREST-TOKENIZER FALLBACK

> **Status: implemented** (2026-08-10). Tasks 1–4, 6, 7, and 9 from §16.9 have shipped; see the file list at the end of this section. Task 5 (real calibration data) is scaffolded (`scripts/calibrate.py`) but not run — no network access was available to call Anthropic's or Google's counting APIs, and calibration must never be fabricated (§16.4). The two shipped profiles are honest, uncalibrated identity placeholders (`sample_count: 0`), which is why they report `AccuracyEstimatedHeuristic` rather than `AccuracyEstimatedCalibrated` and are rejected by `CountForModel`'s default config unless the caller opts in with `WithHeuristicFallback()`. Task 8 (Tier B: Qwen/DeepSeek/Kimi/Mistral sibling modules) is out of scope — it requires downloading and parsing real HF byte-BPE/Tekken vocabularies, which the offline-focused ask that prompted this section did not require. All new code passed `go build ./... && go test ./... && go test -race ./...` with a real Go 1.26.5 toolchain (`/tmp/go/bin`, `GOCACHE`/`GOPATH` in `/tmp`, not the mounted repo — the mounted path rejected the module cache's internal file removals). The rest of this section is kept as originally written as the design record; the status notes below each numbered task in §16.9 point to what actually landed.
>
> This section supersedes the framing in §5.7. Reading `docs/superpowers/plans/2026-08-06-multi-provider-tokenizer.md` shows the ~30 unreferenced symbols are **deliberate Phase 1 scaffolding**, not accidental dead code. The problem is not that they are unimplemented — it is that the design decision they depend on has never been made. This section makes it.

### 16.1 The constraint that blocks everything, and how to resolve it

The plan states two requirements that cannot both hold:

- *"No external API requests (OpenAI, Anthropic, Google, xAI, etc.). No network calls."*
- Eight providers in `provider.go`, including Anthropic and Gemini.

Neither Anthropic (since Claude 3) nor Google (Gemini) publishes a tokenizer artifact. Both expose token counting **only** through a `count_tokens` API endpoint. Under a strict no-network constraint, exact counting for these providers is not merely unimplemented — it is impossible, and no amount of Phase 2 work changes that.

**The resolution is to split the constraint by lifetime:**

| | Network allowed? |
|---|---|
| **Library runtime** (what users import) | **Never.** Absolute. This is the product's differentiator. |
| **Maintainer calibration** (a script you run occasionally) | **Yes, once per profile.** Output is a committed JSON file. |

This is not a loophole — it is how every offline estimator works. The user's process makes no network calls; a maintainer measured the ratios ahead of time and shipped them as data. `Calibration.CorpusSHA256` already exists in `count_result.go:32`, which means the original design anticipated exactly this and then stopped short of saying so.

Everything below assumes this split.

### 16.2 Provider tiering

Replace the flat `Provider` list with an explicit accuracy tier per provider. This is the single most important documentation and API change in the subsystem.

| Tier | Providers | Mechanism | `Accuracy` | Ships where |
|---|---|---|---|---|
| **A — Exact, embedded** | OpenAI (`cl100k_base`, `o200k_base`, `o200k_harmony`) | tiktoken BPE, `//go:embed` | `AccuracyExactLocal` | base module (current 5.2 MB) |
| **B — Exact, optional module** | Qwen, DeepSeek, Kimi (HF byte-BPE); Mistral (Tekken) | public `tokenizer.json` / `tekken.json` | `AccuracyExactLocal` | **separate Go modules** — see 16.2.1 |
| **C — Calibrated estimate** | Anthropic, Gemini | nearest embedded tokenizer × committed ratio | `AccuracyEstimatedCalibrated` | base module (profile JSON is ~2 KB) |
| **D — Heuristic** | unknown / unregistered models | bytes-per-token heuristic | `AccuracyEstimatedHeuristic` | base module, **opt-in only** |

Tier D must stay opt-in. `DefaultCountConfig()` currently sets `AllowHeuristicFallback: false` (`count_options.go:15`) — that default is correct and should not change.

#### 16.2.1 Why Tier B must be a separate module

`tokenizer.json` for Qwen 2.5 is ~11 MB; Mistral's `tekken.json` is comparable. Embedding four of them takes the module from 5.2 MB to ~50 MB, downloaded by every `go get` even for users who only want GPT-4o. Ship them as sibling modules:

```
github.com/xorvus/tokenizer-go                 (base, OpenAI + Tier C profiles)
github.com/xorvus/tokenizer-go/providers/qwen  (own go.mod, own embed)
github.com/xorvus/tokenizer-go/providers/mistral
```

Each sibling registers itself into the base registry from its own `init()` — which requires the public `Registry.Register` API that does not currently exist (§5.6). This is the concrete reason the missing registration API matters: **without it, Tier B cannot be built at all** without editing `registry.go` for every provider.

Tier B also needs a second `Algorithm` implementation (HF byte-BPE uses a byte-level pre-tokenizer and an explicit merges list, not tiktoken's regex-split + rank-map). `AlgorithmHFByteBPE` and `AlgorithmTekken` already exist as constants in `engine.go:7-9`; they now have a reason to.

### 16.3 Resolution chain with fallback

`Registry.Resolve` currently has three steps and can only succeed exactly or fail (`registry.go:74-85`). Add a fourth step that consults a per-provider fallback profile, and make every step set `Accuracy`, `UsedFallback`, and `Reason` honestly:

```go
// registry.go

type ModelSpec struct {
	CanonicalName   string
	Provider        Provider
	TokenizerID     string   // "" when no exact tokenizer exists
	FallbackProfile string   // profile ID, e.g. "anthropic-claude-v1"; "" when exact
	Capabilities    Capabilities
	IsExact         bool
	// Aliases/Prefixes/ChatTemplateID removed — see §5.7, they duplicate
	// the registry's own aliases/prefixes maps and were never populated.
}

func (r *Registry) Resolve(model string) (Resolution, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// 1-3: exact, alias, prefix — unchanged, all AccuracyExactLocal.
	if res, ok := r.resolveExactOrAlias(model); ok {
		return res, nil
	}
	if res, ok := r.resolvePrefix(model); ok {
		return res, nil
	}

	// 4: NEW — provider-level fallback for registered but inexact models.
	if spec, ok := r.fallbackModels[model]; ok {
		prof, ok := calib.Lookup(spec.FallbackProfile)
		if !ok {
			return Resolution{}, fmt.Errorf("%w: profile %q missing for %s",
				ErrExactTokenizerUnavailable, spec.FallbackProfile, model)
		}
		return Resolution{
			RequestedModel: model,
			CanonicalModel: spec.CanonicalName,
			Provider:       spec.Provider,
			TokenizerID:    prof.BaseTokenizer,   // e.g. "o200k_base"
			Accuracy:       AccuracyEstimatedCalibrated,
			UsedFallback:   true,
			Reason: fmt.Sprintf("no public tokenizer for %s; estimated via %s (profile %s)",
				spec.Provider, prof.BaseTokenizer, prof.ProfileID),
		}, nil
	}

	return Resolution{}, fmt.Errorf("%w: %s", ErrUnknownModel, model)
}
```

**Base-tokenizer choice for Tier C.** Use `o200k_base` for both Anthropic and Gemini. Its ~200k vocabulary is structurally much closer to Claude 3+'s and Gemini's large-vocabulary tokenizers than `cl100k_base`'s ~100k, so the correction ratio stays near 1.0 and the residual error is smaller. **[measure]** — confirm against both `cl100k_base` and `o200k_base` as base and keep whichever gives lower P95 error; the profile format below records the choice, so it is data, not a hardcoded assumption.

Note what this unlocks: **step 4 is the only thing standing between the current code and a live `UsedFallback`, `Reason`, `Calibration`, `AccuracyEstimatedCalibrated`, `ErrExactTokenizerUnavailable`, `FallbackProfile`, `WithExactOnly`, and `WithCalibratedFallback`.** One resolution branch turns the entire dormant API surface on at once — and makes `WithExactOnly()` a real guarantee instead of a no-op (§5.7).

### 16.4 Calibration profile — format and provenance

Committed data, embedded, never fetched at runtime:

```go
// internal/calib/profile.go

type Bucket string

const (
	BucketLatin Bucket = "latin" // Latin-script prose
	BucketCJK   Bucket = "cjk"   // Chinese / Japanese / Korean
	BucketCode  Bucket = "code"  // source code, JSON, markup
	BucketOther Bucket = "other" // Cyrillic, Arabic, Devanagari, emoji-heavy, mixed
)

type Ratio struct {
	Mean            float64 `json:"mean"`               // target_tokens / base_tokens
	P95             float64 `json:"p95"`
	Max             float64 `json:"max"`
	MeanAbsErrorPct float64 `json:"mean_abs_error_pct"`
	P95AbsErrorPct  float64 `json:"p95_abs_error_pct"`
}

type Profile struct {
	ProfileID     string            `json:"profile_id"`     // "anthropic-claude-v1"
	TargetModel   string            `json:"target_model"`   // "claude-sonnet-4"
	BaseTokenizer string            `json:"base_tokenizer"` // "o200k_base"
	Ratios        map[Bucket]Ratio  `json:"ratios"`
	SampleCount   int               `json:"sample_count"`
	CorpusSHA256  string            `json:"corpus_sha256"`
	GeneratedAt   string            `json:"generated_at"`
	ToolVersion   string            `json:"tool_version"`
}

//go:embed profiles/*.json
var profileFS embed.FS
```

**Provenance — `scripts/calibrate.py`** (maintainer-only, mirrors the existing `scripts/generate_fixtures.py` pattern):

1. Read a committed corpus (`testdata/corpus/*.txt`), bucketed by script class.
2. For each sample: count with the local base tokenizer, and get ground truth from the provider's `count_tokens` endpoint.
3. Emit mean / P95 / max of `target/base` per bucket, plus the corpus SHA-256.
4. Commit the corpus **and** the resulting `internal/calib/profiles/*.json` together.

`CorpusSHA256` makes each profile reproducible and auditable — anyone can re-run the script against the same corpus and check the numbers. Add a CI job that fails if a profile is older than N months, so ratios do not silently rot as providers update tokenizers.

**Do not invent the ratios.** Every number in a shipped profile must come from a measurement run, and `SampleCount` must be large enough that P95 is meaningful (target ≥ 500 samples per bucket). A profile with fabricated constants is worse than no fallback, because it launders a guess through a struct that looks authoritative.

### 16.5 Why a single scalar ratio is wrong

The token-count ratio between two tokenizers is **not** a constant — it varies by more between scripts than most estimators' entire error budget. A tokenizer with a large CJK-heavy vocabulary and one without differ by a small factor on English prose and a large one on Chinese. A single global multiplier tuned on English will be badly wrong on CJK, and vice versa.

Hence the per-bucket `Ratios` map. Classification is one cheap pass, and the codebase already has the primitive (`isASCII`, `encode.go:10`):

```go
// internal/calib/bucket.go

// Classify returns the script bucket for text using a single pass.
// It samples at most maxSample bytes so cost stays O(1) for large inputs.
func Classify(text string) Bucket {
	const maxSample = 4096
	s := text
	if len(s) > maxSample {
		s = s[:maxSample] // may split a rune; DecodeRuneInString handles it
	}

	var cjk, letters, punct, total int
	for _, r := range s {
		total++
		switch {
		case unicode.Is(unicode.Han, r), unicode.Is(unicode.Hiragana, r),
			unicode.Is(unicode.Katakana, r), unicode.Is(unicode.Hangul, r):
			cjk++
		case unicode.IsLetter(r):
			letters++
		case r < 0x80 && (unicode.IsPunct(r) || unicode.IsSymbol(r)):
			punct++
		}
	}
	if total == 0 {
		return BucketLatin
	}
	switch {
	case cjk*100/total >= 20:
		return BucketCJK
	case punct*100/total >= 15: // brace/semicolon/quote density ⇒ code or markup
		return BucketCode
	case letters*100/total >= 40 && isMostlyLatin(s):
		return BucketLatin
	default:
		return BucketOther
	}
}
```

Thresholds are starting points — **[measure]** them against the calibration corpus and tune for lowest P95 error. The 4 KB sampling cap keeps this O(1) rather than O(n), which matters because it runs on every `CountForModel` call for Tier C/D models.

### 16.6 Applying the fallback in `CountForModel`

```go
func CountForModel(model, text string, opts ...CountOption) (CountResult, error) {
	cfg := DefaultCountConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	res, err := ResolveModel(model)
	if err != nil {
		return CountResult{}, err
	}

	// Gate before doing any work.
	switch res.Accuracy {
	case AccuracyEstimatedCalibrated:
		if cfg.ExactOnly || !cfg.AllowCalibratedFallback {
			return CountResult{}, fmt.Errorf("%w: %s resolves via calibrated estimate",
				ErrExactTokenizerUnavailable, model)
		}
	case AccuracyEstimatedHeuristic:
		if !cfg.AllowHeuristicFallback {
			return CountResult{}, fmt.Errorf("%w: %s has no calibration profile",
				ErrExactTokenizerUnavailable, model)
		}
	}

	tok, err := GetEncoding(Encoding(res.TokenizerID)) // §13.1: now cached
	if err != nil {
		return CountResult{}, err
	}
	base, err := tok.Count(text) // §13.3: now allocation-free
	if err != nil {
		return CountResult{}, err
	}

	out := CountResult{
		Tokens:         base,
		RequestedModel: res.RequestedModel,
		CanonicalModel: res.CanonicalModel,
		TokenizerID:    res.TokenizerID,
		Provider:       res.Provider,
		Accuracy:       res.Accuracy,
		UsedFallback:   res.UsedFallback,
		FallbackReason: res.Reason,
	}
	if !res.UsedFallback {
		return out, nil // exact path: never scale, never allocate a Calibration
	}

	prof, _ := calib.Lookup(res.ProfileID)
	b := calib.Classify(text)
	ratio, ok := prof.Ratios[b]
	if !ok {
		ratio = prof.Ratios[BucketOther] // profiles must always define "other"
	}
	out.Tokens = int(math.Ceil(float64(base) * ratio.Mean))
	out.Calibration = &Calibration{
		ProfileID:       prof.ProfileID,
		SampleCount:     prof.SampleCount,
		MeanAbsErrorPct: ratio.MeanAbsErrorPct,
		P95AbsErrorPct:  ratio.P95AbsErrorPct,
		CorpusSHA256:    prof.CorpusSHA256,
	}
	return out, nil
}
```

Two properties worth stating explicitly: the exact path never touches calibration and never allocates a `*Calibration`, so Tier A performance is untouched; and the gate runs **before** tokenization, so `WithExactOnly()` fails fast rather than after doing the work.

### 16.7 Safety semantics — an estimate must never silently under-count

For a gateway enforcing a context window or a spend limit, under-counting is the failure that costs money and truncates prompts; over-counting merely wastes headroom. `CountResult.Tokens` should carry the **mean** estimate (the honest central value), and callers enforcing a limit should use an upper bound. `WithSafetyMargin` (`count_result.go:47`) is already the right shape — add its calibration-aware sibling:

```go
// UpperBound returns a conservative token count suitable for budget enforcement.
// For exact results it returns Tokens unchanged. For estimates it inflates
// Tokens by the profile's P95 error, so the true count is below this ~95% of the time.
func (r CountResult) UpperBound() int {
	if r.Accuracy == AccuracyExactLocal || r.Calibration == nil {
		return r.Tokens
	}
	return r.WithSafetyMargin(r.Calibration.P95AbsErrorPct)
}
```

Document the contract plainly in the README: **exact for OpenAI (and Tier B), estimated within a stated P95 for Anthropic and Gemini, and never a network call.** An estimate honestly labelled and bounded is a legitimate product. An estimate presented as exact is not.

### 16.8 Prerequisite: fix the `Accuracy` zero value first

```go
const ( AccuracyExactLocal Accuracy = iota   // == 0  ← wrong
```

Every `return CountResult{}, err` in `CountForModel` yields a struct claiming `Accuracy: ExactLocal, Tokens: 0`. A caller that ignores the error reads the most reassuring label in the enum. This is fail-open on the exact field this whole subsystem exists to make trustworthy, and it must be fixed **before** any fallback lands, or the fallback inherits it:

```go
const (
	AccuracyUnknown Accuracy = iota // zero value: nothing was resolved
	AccuracyExactLocal
	AccuracyEstimatedCalibrated
	AccuracyEstimatedHeuristic
)
```

This is a breaking change to the numeric values of exported constants — another entry for the v2 batch in §14.2. While there, add `MarshalJSON`/`UnmarshalJSON` for `Accuracy` and `Capabilities`: every `CountResult` field carries a JSON tag, so wire output is clearly intended, yet `Accuracy` currently serialises as `0`/`1`/`2` despite `String()` existing (`count_result.go:14`).

### 16.9 Task order

| # | Task | Breaking? | Blocks | Status |
|---|---|---|---|---|
| 1 | `AccuracyUnknown` as zero value; `MarshalJSON` for `Accuracy`/`Capabilities` | yes | everything below | **Done** — `count_result.go`, `capabilities.go`. Also added `UnmarshalJSON` for both and `Capabilities.Has()`. |
| 2 | Unify the two model registries (§12 P0-3) so there is one resolution path to extend | yes | 3 | **Done** — `registry.go` now builds `exactModels`/`prefixes` from `internal/openai`'s tables directly (single source of truth); `EncodingForModel`/`ForModel` route through `ResolveModel` instead of reading `openai.ModelToEncoding` independently. Prefix ordering is `sort.SliceStable` with an explicit lexical tiebreak (§3.3's nondeterminism fix), covered by `TestPrefixResolutionIsDeterministicUnderOverlap`. |
| 3 | `Registry.Register(ModelSpec)` + `Registry.Lookup(model) (ModelSpec, bool)` | additive | Tier B and C | **Done**, plus `RegisterAlias`, `RegisterPrefix`, and `RegisterFallbackPrefix` (not originally scoped, but `Register` alone couldn't add prefix-based or fallback-based rules, only exact ones). Package-level wrappers (`RegisterModel`, `LookupModel`, etc.) added since `globalRegistry` itself was never exported and users have no other way to reach it. |
| 4 | `internal/calib`: `Profile`, `Ratio`, `Bucket`, `Classify`, `Lookup`, embedded `profiles/*.json` | additive | 5, 6 | **Done** — `internal/calib/{profile,bucket}.go`. `Classify`'s thresholds are as originally sketched, now with a real test (`internal/calib/calib_test.go`) covering Latin/CJK(zh,ja,ko)/code/JSON. Also exported at the top level (`calibration.go`: `Bucket`, `CalibrationProfile`, `CalibrationRatio`, `RegisterCalibrationProfile`, `LookupCalibrationProfile`) — the original design left `internal/calib` unreachable from outside the module, which would have made `RegisterFallbackPrefix`'s "add another provider without forking" claim false for any external caller. |
| 5 | `scripts/calibrate.py` + committed corpus; generate `anthropic-claude-v1`, `gemini-v1` | — | 6 | **Scaffolded, not run.** `scripts/calibrate.py` + `scripts/count_helper.go` (a `go run` bridge so the "base" count in calibration comes from this module's own `Count`, not a Python reimplementation that could drift) are written and syntax-checked, but no calibration run happened — no network access to Anthropic's/Google's counting APIs in this environment, and per §16.4, numbers must never be hand-invented. `internal/calib/profiles/{anthropic-claude-v1,gemini-v1}.json` ship as declared-uncalibrated identity profiles (`sample_count: 0`, all ratios `1.0`, a `notes` field stating this explicitly). This is the one task in this table that remains genuinely open. |
| 6 | Resolution step 4 + `CountForModel` scaling + `UpperBound()` | additive | — | **Done** — `registry.go`'s `resolveFallback`, `tokenizer.go`'s `CountForModel`, `count_result.go`'s `UpperBound()`. One deviation from the original sketch: `resolveFallback` derives `Accuracy` from `prof.SampleCount` (`IsCalibrated()`) rather than a fixed `AccuracyEstimatedCalibrated`, so the identity placeholders correctly report `AccuracyEstimatedHeuristic` — see the callout below this table. `UpperBound()` also applies a fixed 25% margin when `Calibration` is present but `SampleCount == 0`, rather than trusting a `P95AbsErrorPct` of 0 that was never measured. |
| 7 | Split `Engine` into `Counter` (`Count`) and `Codec` (`Encode`/`Decode`); add `var _ Counter = (*Tokenizer)(nil)` | yes | Tier B | **Done** — `engine.go`, assertion in `tokenizer.go` (as `var _ Engine`, since `*Tokenizer` implements both `Counter` and `Codec`; a hypothetical Tier C `Counter`-only implementation would assert `var _ Counter`, not `Engine`). |
| 8 | Tier B sibling modules for Qwen / DeepSeek / Kimi / Mistral | additive | — | **Not started** — out of scope for the offline-fallback ask that prompted this implementation pass; needs real HF byte-BPE/Tekken vocab files and a second BPE algorithm, neither available here. `RegisterModelFallbackPrefix`/`RegisterCalibrationProfile` demonstrate the extension point a Tier B module (or a caller) would use (see `registry_fallback_test.go`'s `TestPublicRegistryExtensionAPI`, and the README's Mistral example), but no such module exists yet. |
| 9 | Golden test: every registered model resolves; every `FallbackProfile` ID exists in the embedded profiles | — | — | **Done, in a stronger form** — `resolveFallback` itself refuses to resolve a fallback prefix whose profile isn't registered (skips to the next rule / `ErrUnknownModel` instead of returning a broken `Resolution`), and `RegisterFallbackPrefix`/`RegisterModelFallbackPrefix` reject registration up front if the named profile doesn't exist yet (`TestRegisterFallbackPrefixRejectsUnknownProfile`), so the invariant is enforced structurally rather than only checked after the fact by a test. |

Task 7 deserves a note. The current `Engine` (`engine.go:19`) demands `Encode`, `Decode`, **and** `Count` from every implementation. For a Tier C provider there is no meaningful `Encode` — the library cannot produce Claude token IDs, only estimate how many there would be. Forcing Tier C to implement `Encode` guarantees either a panic or a lie. Splitting the interface lets Tier C implement `Counter` only, which is also what `Capabilities` was designed to express (`CapabilityCountText` without `CapabilityEncode`). Add `Registry.Lookup` so callers can actually read that bitmask — today it is written and unreadable (§5.7).

**A design refinement made during implementation, not in the original §16.6 sketch:** the sketch in §16.6 always set `Accuracy: AccuracyEstimatedCalibrated` for any fallback resolution. The shipped code instead computes it from the matched profile's `SampleCount` (`Profile.IsCalibrated()` in `internal/calib/profile.go`): `SampleCount == 0` → `AccuracyEstimatedHeuristic`, `SampleCount > 0` → `AccuracyEstimatedCalibrated`. This matters because it is what makes the currently-shipped `claude-*`/`gemini-*` fallbacks honest — they run through the identical code path §16.6 describes, but since no real calibration data exists yet (task 5), they correctly report the lower-confidence label and are rejected by `CountForModel`'s default config. Running `scripts/calibrate.py` and committing a profile with `sample_count > 0` upgrades the label automatically, with no code change to `registry.go` or `tokenizer.go`.

Verification performed: `go build ./...`, `go test ./...`, `go test -race ./...` (including a dedicated concurrent-registration stress test, `TestConcurrentRegisterAndResolve`, exercising `Register`/`Resolve`/`CountForModel` from multiple goroutines against the shared global registry), and `go vet ./...` (clean except two pre-existing warnings in `custom_test.go`/`security_test.go`, unrelated to this section, present before this work started). New/changed files: `count_result.go`, `capabilities.go`, `calibration.go` (new), `model.go`, `resolution.go`, `registry.go`, `engine.go`, `tokenizer.go`, `internal/calib/{profile,bucket,calib_test}.go` (new), `internal/calib/profiles/*.json` (new), `scripts/{calibrate.py,count_helper.go}` (new), `registry_fallback_test.go` (new), `capabilities_test.go` (new), `count_result_test.go`, `README.md`.

### 16.10 What this design does not solve

Stated plainly, so nobody expects more than it delivers:

- **Chat-message overhead.** Every provider adds per-message and per-role framing tokens. `CapabilityCountMessages` and `ModelSpec.ChatTemplateID` gesture at it; nothing implements it. Raw text counting will under-count a multi-turn conversation regardless of how good the ratio is — and under-counting is the dangerous direction (16.7). This needs its own design.
- **Tool/function-call schemas.** `CapabilityTools` exists; JSON-schema serialisation overhead is provider-specific and undocumented. Currently unestimable.
- **Multimodal.** `CapabilityMultimodalEstimate` exists; image token cost is a function of resolution and provider tiling rules. Out of scope for a text tokenizer, and the capability flag should say so or be removed.
- **Model-name ambiguity across providers.** The registry is keyed on a bare model-name string. Real gateways see the same name served by several providers. `ResolveModel` will eventually need a provider hint (`ResolveModelFor(provider, model)`), and adding it later is breaking.
- **Profile staleness.** Providers change tokenizers without announcement. A committed ratio is a snapshot; `GeneratedAt` plus a CI staleness check is mitigation, not a fix.
