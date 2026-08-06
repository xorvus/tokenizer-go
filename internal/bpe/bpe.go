package bpe

import "math"

const missingShortRank int32 = -1

type RankIndex struct {
	short2 [1 << 16]int32
	ranks  map[string]int
}

func NewRankIndex(ranks map[string]int) RankIndex {
	idx := RankIndex{ranks: ranks}
	for i := range idx.short2 {
		idx.short2[i] = missingShortRank
	}
	for token, rank := range ranks {
		if len(token) == 2 && rank <= math.MaxInt32 {
			key := uint16(token[0])<<8 | uint16(token[1])
			idx.short2[key] = int32(rank)
		}
	}
	return idx
}

func (idx *RankIndex) Lookup(token string) (int, bool) {
	if len(token) == 2 {
		key := uint16(token[0])<<8 | uint16(token[1])
		rank := idx.short2[key]
		if rank != missingShortRank {
			return int(rank), true
		}
		return 0, false
	}
	rank, ok := idx.ranks[token]
	return rank, ok
}

type part struct {
	start int
	rank  int
}

func getPartRank(piece string, parts []part, i int, idx *RankIndex) int {
	if i+2 < len(parts) {
		if r, ok := idx.Lookup(piece[parts[i].start : parts[i+2].start]); ok {
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
	for i := range parts {
		parts[i] = part{start: i, rank: math.MaxInt}
	}
	return parts
}

func BytePairEncodeToWithIndex(piece string, idx *RankIndex, out []int) []int {
	if len(piece) == 0 {
		return out
	}
	if len(piece) == 1 {
		rank, _ := idx.Lookup(piece)
		return append(out, rank)
	}
	var local [33]part
	parts := makePartsBuffer(len(piece)+1, local[:])
	for i := 0; i < len(parts)-2; i++ {
		parts[i].rank = getPartRank(piece, parts, i, idx)
	}
	parts = bytePairMergeLoop(piece, parts, idx)
	for i := 0; i < len(parts)-1; i++ {
		rank, _ := idx.Lookup(piece[parts[i].start : parts[i+1].start])
		out = append(out, rank)
	}
	return out
}

func BytePairEncodeTo(piece string, ranks map[string]int, out []int) []int {
	idx := NewRankIndex(ranks)
	return BytePairEncodeToWithIndex(piece, &idx, out)
}

func BytePairEncode(piece string, ranks map[string]int) []int {
	return BytePairEncodeTo(piece, ranks, nil)
}
