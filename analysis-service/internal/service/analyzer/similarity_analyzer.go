package analyzer

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/RubachokBoss/plagiarism-checker/analysis-service/internal/models"
	"github.com/RubachokBoss/plagiarism-checker/analysis-service/internal/service/integration"
	"github.com/rs/zerolog"
)

type SimilarityAnalyzer interface {
	AnalyzeContent(ctx context.Context, file1, file2 []byte) (float64, error)
	ExtractText(content []byte) (string, error)
	CalculateSimilarity(text1, text2 string) float64
	FindSimilarSections(text1, text2 string, minLength int) []SimilarSection
}

type SimilarSection struct {
	Text1Start int     `json:"text1_start"`
	Text1End   int     `json:"text1_end"`
	Text2Start int     `json:"text2_start"`
	Text2End   int     `json:"text2_end"`
	Similarity float64 `json:"similarity"`
	Text       string  `json:"text"`
}

type similarityAnalyzer struct {
	fileClient integration.FileClient
	logger     zerolog.Logger
}

func NewSimilarityAnalyzer(fileClient integration.FileClient, logger zerolog.Logger) SimilarityAnalyzer {
	return &similarityAnalyzer{
		fileClient: fileClient,
		logger:     logger,
	}
}

func (a *similarityAnalyzer) AnalyzeContent(ctx context.Context, file1, file2 []byte) (float64, error) {
	startTime := time.Now()

	// Extract text from files
	text1, err := a.ExtractText(file1)
	if err != nil {
		return 0, fmt.Errorf("failed to extract text from first file: %w", err)
	}

	text2, err := a.ExtractText(file2)
	if err != nil {
		return 0, fmt.Errorf("failed to extract text from second file: %w", err)
	}

	// Calculate similarity
	similarity := a.CalculateSimilarity(text1, text2)

	a.logger.Debug().
		Int("text1_length", len(text1)).
		Int("text2_length", len(text2)).
		Float64("similarity", similarity).
		Dur("processing_time", time.Since(startTime)).
		Msg("Content analysis completed")

	return similarity, nil
}

func (a *similarityAnalyzer) ExtractText(content []byte) (string, error) {
	// Simple text extraction - in real implementation you would:
	// 1. Detect file type (PDF, DOC, TXT, etc.)
	// 2. Use appropriate library to extract text
	// 3. Clean and normalize text

	// For now, just convert to string and clean
	text := string(content)

	// Remove extra whitespace
	text = strings.Join(strings.Fields(text), " ")

	// Convert to lowercase for comparison
	text = strings.ToLower(text)

	return text, nil
}

func (a *similarityAnalyzer) CalculateSimilarity(text1, text2 string) float64 {
	// Simple similarity calculation using Jaccard similarity
	// In real implementation, you might use:
	// - Cosine similarity with TF-IDF
	// - Levenshtein distance
	// - More sophisticated NLP techniques

	if text1 == "" || text2 == "" {
		return 0.0
	}

	// Tokenize texts
	tokens1 := strings.Fields(text1)
	tokens2 := strings.Fields(text2)

	// Create sets
	set1 := make(map[string]bool)
	for _, token := range tokens1 {
		set1[token] = true
	}

	set2 := make(map[string]bool)
	for _, token := range tokens2 {
		set2[token] = true
	}

	// Calculate intersection and union
	intersection := 0
	for token := range set1 {
		if set2[token] {
			intersection++
		}
	}

	union := len(set1) + len(set2) - intersection

	if union == 0 {
		return 0.0
	}

	return float64(intersection) / float64(union)
}

func (a *similarityAnalyzer) FindSimilarSections(text1, text2 string, minLength int) []SimilarSection {
	var sections []SimilarSection

	// Simple algorithm to find similar sections
	// In real implementation, use more sophisticated algorithms like:
	// - Longest Common Substring
	// - Smith-Waterman algorithm for local alignment

	words1 := strings.Fields(text1)
	words2 := strings.Fields(text2)

	// Create n-grams
	ngrams1 := a.createNGrams(words1, minLength)
	ngrams2 := a.createNGrams(words2, minLength)

	// Find matching n-grams
	for i, gram1 := range ngrams1 {
		for j, gram2 := range ngrams2 {
			similarity := a.CalculateSimilarity(gram1, gram2)
			if similarity > 0.8 { // High similarity threshold
				section := SimilarSection{
					Text1Start: i,
					Text1End:   i + minLength - 1,
					Text2Start: j,
					Text2End:   j + minLength - 1,
					Similarity: similarity,
					Text:       gram1,
				}
				sections = append(sections, section)
			}
		}
	}

	return sections
}

func (a *similarityAnalyzer) createNGrams(words []string, n int) []string {
	if n > len(words) {
		return []string{}
	}

	ngrams := make([]string, len(words)-n+1)
	for i := 0; i <= len(words)-n; i++ {
		ngrams[i] = strings.Join(words[i:i+n], " ")
	}
	return ngrams
}

// TextSimilarityCalculator provides text-based similarity analysis
type TextSimilarityCalculator struct {
	analyzer SimilarityAnalyzer
}

func NewTextSimilarityCalculator(analyzer SimilarityAnalyzer) *TextSimilarityCalculator {
	return &TextSimilarityCalculator{
		analyzer: analyzer,
	}
}

func (c *TextSimilarityCalculator) CompareFiles(ctx context.Context, file1ID, file2ID string) (float64, error) {
	// Get file contents
	// In real implementation, you would download files from storage
	// For now, return placeholder
	return 0.0, nil
}

func (c *TextSimilarityCalculator) GenerateReport(text1, text2 string, similarity float64) models.ReportDetails {
	// Find similar sections
	sections := c.analyzer.FindSimilarSections(text1, text2, 10)

	// Create report
	details := models.ReportDetails{
		AnalysisMetadata: models.AnalysisMetadata{
			AlgorithmUsed:    "text_similarity",
			SimilarityMethod: "jaccard_similarity",
			AnalysisVersion:  "1.0",
			Threshold:        80,
			StartedAt:        time.Now(),
			CompletedAt:      time.Now(),
		},
	}

	// Convert sections to comparison results
	for i, section := range sections {
		details.ComparisonResults = append(details.ComparisonResults, models.ComparisonResult{
			ComparedWorkID:  fmt.Sprintf("section_%d", i),
			MatchPercentage: int(section.Similarity * 100),
			ComparedAt:      time.Now().Format(time.RFC3339),
		})
	}

	return details
}
