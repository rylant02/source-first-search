package pipeline

import (
	"math"
	"strings"
)

type EngineReranker struct{}

func NewEngineReranker() *EngineReranker {
	return &EngineReranker{}
}

// rerank evaluates raw documents against original query, score on depth and density
func (er *EngineReranker) Rerank(originalQuery string, retrievedDocs []SourceDocument) []SourceDocument {
	if len(retrievedDocs) == 0 {
		return retrievedDocs
	}

	queryTokens := strings.Fields(strings.ToLower(originalQuery))

	for i := range retrievedDocs {
		contentLower := strings.ToLower(retrievedDocs[i].Content)
		words := strings.Fields(contentLower)
		totalWords := float64(len(words))

		if totalWords == 0 {
			retrievedDocs[i].DensityScore = 0
			continue
		}

		// protect against SEO spam
		uniqueWords := make(map[string]bool)
		for _, w := range words {
			if len(w) > 3 {
				uniqueWords[w] = true
			}
		}
		vocabularyRichness := float64(len(uniqueWords)) / totalWords

		// count specific keywords
		var intentMatches float64
		for _, qToken := range queryTokens {
			if len(qToken) > 3 && strings.Contains(contentLower, qToken) {
				intentMatches += 1.0
			}
		}
		intentAlignment := intentMatches / float64(len(queryTokens))

		// favor long-form documentation over quick answers.
		lengthBonus := math.Log10(totalWords) / 3.0 
		if lengthBonus > 1.0 {
			lengthBonus = 1.0
		}

		// combine components into weighted Informational Density metric
		finalDensity := (intentAlignment * 0.5) + (vocabularyRichness * 0.3) + (lengthBonus * 0.2)
		retrievedDocs[i].DensityScore = finalDensity
	}

	// sort documents cleanly by finalized density scores, desc
	for i := 0; i < len(retrievedDocs); i++ {
		for j := i + 1; j < len(retrievedDocs); j++ {
			if retrievedDocs[i].DensityScore < retrievedDocs[j].DensityScore {
				retrievedDocs[i], retrievedDocs[j] = retrievedDocs[j], retrievedDocs[i]
			}
		}
	}

	return retrievedDocs
}
