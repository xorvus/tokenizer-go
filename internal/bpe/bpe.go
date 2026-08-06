package bpe

import "math"

type part struct {
	start int
	rank  int
}

func getPartRank(piece string, parts []part, i int, ranks map[string]int) int {
	if i+2 < len(parts) {
		if r, ok := ranks[piece[parts[i].start:parts[i+2].start]]; ok {
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

func mergeMinPart(piece string, parts []part, i int, ranks map[string]int) []part {
	parts = append(parts[:i+1], parts[i+2:]...)
	parts[i].rank = getPartRank(piece, parts, i, ranks)
	if i > 0 {
		parts[i-1].rank = getPartRank(piece, parts, i-1, ranks)
	}
	return parts
}

func bytePairMergeLoop(piece string, parts []part, ranks map[string]int) []part {
	for len(parts) > 1 {
		minRank, minIdx := findMinRank(parts)
		if minRank == math.MaxInt {
			break
		}
		parts = mergeMinPart(piece, parts, minIdx, ranks)
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

func BytePairEncodeTo(piece string, ranks map[string]int, out []int) []int {
	if len(piece) == 0 {
		return out
	}
	if len(piece) == 1 {
		return append(out, ranks[piece])
	}
	var local [33]part
	parts := makePartsBuffer(len(piece)+1, local[:])
	for i := 0; i < len(parts)-2; i++ {
		parts[i].rank = getPartRank(piece, parts, i, ranks)
	}
	parts = bytePairMergeLoop(piece, parts, ranks)
	for i := 0; i < len(parts)-1; i++ {
		out = append(out, ranks[piece[parts[i].start:parts[i+1].start]])
	}
	return out
}

func BytePairEncode(piece string, ranks map[string]int) []int {
	return BytePairEncodeTo(piece, ranks, nil)
}
