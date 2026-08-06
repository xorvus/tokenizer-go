package openai

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type Vocabulary struct {
	Encoder map[string]int
	Decoder []string
}

func parseVocabLine(line string) (string, int, error) {
	parts := strings.Split(line, " ")
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("invalid line format")
	}
	tokenBytes, err := base64.StdEncoding.DecodeString(parts[0])
	if err != nil {
		return "", 0, err
	}
	rank, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", 0, err
	}
	return string(tokenBytes), rank, nil
}

func validateAndInsert(encoder map[string]int, token string, rank int) error {
	if token == "" {
		return fmt.Errorf("empty token")
	}
	if rank < 0 {
		return fmt.Errorf("negative rank: %d", rank)
	}
	if _, exists := encoder[token]; exists {
		return fmt.Errorf("duplicate token: %q", token)
	}
	encoder[token] = rank
	return nil
}

func buildDecoderSlice(encoder map[string]int) []string {
	maxID := 0
	for _, id := range encoder {
		if id > maxID {
			maxID = id
		}
	}
	decoder := make([]string, maxID+1)
	for token, id := range encoder {
		decoder[id] = token
	}
	return decoder
}

func ParseVocabulary(r io.Reader) (*Vocabulary, error) {
	encoder := make(map[string]int)
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		token, rank, err := parseVocabLine(line)
		if err != nil {
			return nil, err
		}
		if err := validateAndInsert(encoder, token, rank); err != nil {
			return nil, err
		}
	}
	return &Vocabulary{Encoder: encoder, Decoder: buildDecoderSlice(encoder)}, nil
}
