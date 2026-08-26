# Changelog

All notable changes to `tokenizer-go` will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v1.1.0] - 2026-08-26

### Added
- **Multi-Provider Fallback Profiles**: Added calibration profiles and registry fallback prefixes for DeepSeek, Qwen, Kimi, Grok, and Mistral.
- **Go 1.27 Upgrade**: Upgraded module and CI workflow to Go 1.27.

### Optimized
- **Singleton Tokenizer Caching**: Cached embedded tokenizers (`cl100k_base`, `o200k_base`, `o200k_harmony`) via `sync.OnceValues` to eliminate repetitive vocabulary parsing.
- **Zero-Allocation Counting**: Routed `Count` directly to `CountOrdinarySequential` without intermediate token slice allocations.
- **Deterministic Prefix Matching**: Sorted prefix mappings by length descending for deterministic model resolution.

## [v1.0.0] - 2026-08-06

### Added
- **Stable Public API**: Freeze public API signatures with guaranteed thread-safety and backward compatibility.
- **Production Hardening**: Full validation for vocabulary completeness (0-255 bytes), non-negative ranks, and immutable configuration maps.
- **Cross-Platform Support**: Tested across Linux (amd64/arm64), macOS (arm64/amd64), and Windows (amd64).

## [v0.9.0] - 2026-08-06

### Added
- **Context Cancellation APIs**: Added `EncodeContext`, `CountContext`, `EncodeOrdinaryBatchContext`, and `CountOrdinaryBatchContext` for AI Gateways with zero hot-path CPU overhead.
- **Resource Control Options**: Added `Options` struct and `WithOptions` method to customize worker pools and byte thresholds.

## [v0.8.0] - 2026-08-06

### Added
- **Custom Encoding Support**: Added `Config` struct and `New(config)` constructor to safely create custom BPE tokenizers.
- **Constructor Validation**: Implemented strict immutability checks, duplicate rank detection, missing single-byte token checks, and special token conflict validation.

## [v0.7.0] - 2026-08-06

### Added
- **OpenAI Parity Methods**: Added `EncodeSingleToken`, `DecodeSingleTokenBytes`, `TokenByteValues`, `DecodeWithOffsets`, `DecodeBatch`, and `DecodeBytesBatch`.
- **OpenAI Harmony & Model Mapping**: Added `o200k_harmony` encoding support and model mappings for `gpt-oss-*`, `gpt-5`, `gpt-4.5`, `o1`, and `o3`.

## [v0.6.2] - 2026-08-06

### Added
- **Security Hardening**: Added single-byte `0-255` coverage validation, adversarial inputs testing, and native Go fuzzing (`FuzzEncodeOrdinary`).
- **Benchmark Suite**: Added PGO ON vs PGO OFF benchmarks confirming ~4% compiler inlining speedup.

## [v0.6.1] - 2026-08-06

### Added
- **Profile-Guided Optimization (PGO)**: Integrated `default.pgo` in package root, achieving `37.448 ns/op` on `o200k_base` (passing OpenAI Rust `tiktoken` at `37.975 ns/op`).
- **Zero-Allocation Token Counting**: Implemented `Count` and `CountOrdinaryBatch` reducing memory allocation by ~19.3%.
- **Packed `uint64` RankIndex**: Fixed-width integer map lookups for 3-7 byte subwords.
- **Hybrid Regex Matcher**: Optimized matcher strategy using `FindAllStringIndex` for short text (<2048B) and streaming `MatchIterator` for long documents.
