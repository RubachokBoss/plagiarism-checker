package analyzer

import (
	"fmt"
	"strings"
)

type HashComparator interface {
	CompareHashes(hash1, hash2 string) (int, error)
	CompareMultiple(hashes []string, targetHash string) (map[string]int, error)
	GetAlgorithm() string
}

type hashComparator struct {
	algorithm string
}

func NewHashComparator(algorithm string) HashComparator {
	return &hashComparator{
		algorithm: strings.ToLower(algorithm),
	}
}

func (c *hashComparator) CompareHashes(hash1, hash2 string) (int, error) {
	hash1 = strings.ToLower(strings.TrimSpace(hash1))
	hash2 = strings.ToLower(strings.TrimSpace(hash2))

	if len(hash1) != len(hash2) {
		return 0, fmt.Errorf("hash lengths don't match: %d vs %d", len(hash1), len(hash2))
	}

	if hash1 == hash2 {
		return 100, nil
	}

	matchingChars := 0
	for i := 0; i < len(hash1); i++ {
		if hash1[i] == hash2[i] {
			matchingChars++
		}
	}

	percentage := (matchingChars * 100) / len(hash1)
	return percentage, nil
}

func (c *hashComparator) CompareMultiple(hashes []string, targetHash string) (map[string]int, error) {
	results := make(map[string]int)

	for _, hash := range hashes {
		percentage, err := c.CompareHashes(targetHash, hash)
		if err != nil {
			return nil, err
		}
		results[hash] = percentage
	}

	return results, nil
}

func (c *hashComparator) GetAlgorithm() string {
	return c.algorithm
}

type AdvancedHashComparator struct {
	hashComparator HashComparator
	threshold      int
}

func NewAdvancedHashComparator(algorithm string, threshold int) *AdvancedHashComparator {
	return &AdvancedHashComparator{
		hashComparator: NewHashComparator(algorithm),
		threshold:      threshold,
	}
}

func (c *AdvancedHashComparator) CompareHashes(hash1, hash2 string) (int, error) {
	return c.hashComparator.CompareHashes(hash1, hash2)
}

func (c *AdvancedHashComparator) CompareMultiple(hashes []string, targetHash string) (map[string]int, error) {
	return c.hashComparator.CompareMultiple(hashes, targetHash)
}

func (c *AdvancedHashComparator) GetAlgorithm() string {
	return c.hashComparator.GetAlgorithm()
}

func (c *AdvancedHashComparator) FindMatches(targetHash string, candidateHashes []string) ([]string, error) {
	var matches []string

	for _, candidate := range candidateHashes {
		percentage, err := c.CompareHashes(targetHash, candidate)
		if err != nil {
			return nil, err
		}

		if percentage >= c.threshold {
			matches = append(matches, candidate)
		}
	}

	return matches, nil
}

func (c *AdvancedHashComparator) GetSimilarityScore(hash1, hash2 string) (float64, error) {
	percentage, err := c.CompareHashes(hash1, hash2)
	if err != nil {
		return 0, err
	}

	return float64(percentage) / 100.0, nil
}
