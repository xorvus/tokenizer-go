package bpe

import "math"

const missingShortRank int32 = -1

type RankIndex struct {
	short1      [256]int32
	short2      [1 << 16]int32
	shortPacked map[uint64]int32
	ranks       map[string]int
}

func packKey(token string) (uint64, bool) {
	l := len(token)
	if l < 3 || l > 7 {
		return 0, false
	}
	key := uint64(l)
	for i := 0; i < l; i++ {
		key |= uint64(token[i]) << ((i + 1) * 8)
	}
	return key, true
}

func packKeyBytes(piece string, start, end int) (uint64, bool) {
	l := end - start
	if l < 3 || l > 7 {
		return 0, false
	}
	key := uint64(l)
	for i := 0; i < l; i++ {
		key |= uint64(piece[start+i]) << ((i + 1) * 8)
	}
	return key, true
}

func NewRankIndex(ranks map[string]int) RankIndex {
	idx := RankIndex{
		shortPacked: make(map[uint64]int32, len(ranks)/4),
		ranks:       ranks,
	}
	for i := range idx.short1 {
		idx.short1[i] = missingShortRank
	}
	for i := range idx.short2 {
		idx.short2[i] = missingShortRank
	}
	for token, rank := range ranks {
		if rank > math.MaxInt32 {
			continue
		}
		if len(token) == 1 {
			idx.short1[token[0]] = int32(rank)
		} else if len(token) == 2 {
			key := uint16(token[0])<<8 | uint16(token[1])
			idx.short2[key] = int32(rank)
		} else if key, ok := packKey(token); ok {
			idx.shortPacked[key] = int32(rank)
		}
	}
	return idx
}

func (idx *RankIndex) Lookup(token string) (int, bool) {
	l := len(token)
	if l == 1 {
		rank := idx.short1[token[0]]
		if rank != missingShortRank {
			return int(rank), true
		}
		return 0, false
	}
	if l == 2 {
		key := uint16(token[0])<<8 | uint16(token[1])
		rank := idx.short2[key]
		if rank != missingShortRank {
			return int(rank), true
		}
		return 0, false
	}
	if key, ok := packKey(token); ok {
		if rank, ok := idx.shortPacked[key]; ok {
			return int(rank), true
		}
		return 0, false
	}
	rank, ok := idx.ranks[token]
	return rank, ok
}

func (idx *RankIndex) Lookup2(b0, b1 byte) (int, bool) {
	key := uint16(b0)<<8 | uint16(b1)
	rank := idx.short2[key]
	if rank != missingShortRank {
		return int(rank), true
	}
	return 0, false
}

func (idx *RankIndex) LookupSlice(piece string, start, end int) (int, bool) {
	l := end - start
	if l == 1 {
		rank := idx.short1[piece[start]]
		if rank != missingShortRank {
			return int(rank), true
		}
		return 0, false
	}
	if l == 2 {
		return idx.Lookup2(piece[start], piece[start+1])
	}
	if key, ok := packKeyBytes(piece, start, end); ok {
		if rank, ok := idx.shortPacked[key]; ok {
			return int(rank), true
		}
		return 0, false
	}
	rank, ok := idx.ranks[piece[start:end]]
	return rank, ok
}

type part struct {
	start int
	rank  int
}

func getPartRank(piece string, parts []part, i int, idx *RankIndex) int {
	if i+2 < len(parts) {
		start := parts[i].start
		end := parts[i+2].start
		if r, ok := idx.LookupSlice(piece, start, end); ok {
			return r
		}
	}
	return math.MaxInt
}

func findMinRank(parts []part) (int, int) {
	minRank, minIdx := math.MaxInt, -1
	for i := 0; i < len(parts)-1; i++ {
		if parts[i].rank < minRank {
			minRank, minIdx = parts[i].rank, i
		}
	}
	return minRank, minIdx
}

func mergeMinPart(piece string, parts []part, i int, idx *RankIndex) []part {
	parts = append(parts[:i+1], parts[i+2:]...)
	parts[i].rank = getPartRank(piece, parts, i, idx)
	if i > 0 {
		parts[i-1].rank = getPartRank(piece, parts, i-1, idx)
	}
	return parts
}

func bytePairMergeLoop(piece string, parts []part, idx *RankIndex) []part {
	for len(parts) > 1 {
		minRank, minIdx := findMinRank(parts)
		if minRank == math.MaxInt {
			break
		}
		parts = mergeMinPart(piece, parts, minIdx, idx)
	}
	return parts
}

func makePartsBuffer(n int, local []part) []part {
	var parts []part
	if n <= len(local) {
		parts = local[:n]
	} else {
		parts = make([]part, n)
	}
	for i := 0; i < n; i++ {
		parts[i].start = i
	}
	if n > 0 {
		parts[n-1].rank = math.MaxInt
	}
	if n > 1 {
		parts[n-2].rank = math.MaxInt
	}
	return parts
}

func BytePairEncodeToWithIndex(piece string, idx *RankIndex, out []int) []int {
	if len(piece) == 0 {
		return out
	}
	if len(piece) == 1 {
		rank := idx.short1[piece[0]]
		if rank != missingShortRank {
			return append(out, int(rank))
		}
		return append(out, 0)
	}
	var local [33]part
	parts := makePartsBuffer(len(piece)+1, local[:])
	for i := 0; i < len(parts)-2; i++ {
		parts[i].rank = getPartRank(piece, parts, i, idx)
	}
	parts = bytePairMergeLoop(piece, parts, idx)
	for i := 0; i < len(parts)-1; i++ {
		start := parts[i].start
		end := parts[i+1].start
		rank, _ := idx.LookupSlice(piece, start, end)
		out = append(out, rank)
	}
	return out
}

func CountPieceToWithIndex(piece string, idx *RankIndex) int {
	if len(piece) == 0 {
		return 0
	}
	if len(piece) == 1 {
		return 1
	}
	var local [33]part
	parts := makePartsBuffer(len(piece)+1, local[:])
	for i := 0; i < len(parts)-2; i++ {
		parts[i].rank = getPartRank(piece, parts, i, idx)
	}
	parts = bytePairMergeLoop(piece, parts, idx)
	return len(parts) - 1
}

func BytePairEncodeTo(piece string, ranks map[string]int, out []int) []int {
	idx := NewRankIndex(ranks)
	return BytePairEncodeToWithIndex(piece, &idx, out)
}

func BytePairEncode(piece string, ranks map[string]int) []int {
	return BytePairEncodeTo(piece, ranks, nil)
}
