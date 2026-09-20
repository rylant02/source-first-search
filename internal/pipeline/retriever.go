package pipeline

import (
	"strings"
)

// represents high-fidelity article source file
type SourceDocument struct {
	ID           int
	Title        string
	URL          string
	Content      string
	DensityScore float64 // used by reranker
}

type HybridRetriever struct {
	Database []SourceDocument
}

// instantiates isolated data repo with diverse entries
func NewHybridRetriever() *HybridRetriever {
	return &HybridRetriever{
		Database: []SourceDocument{
			{
				ID:    1,
				Title: "Caffeine Consumption and Insulin Sensitivity: A Longitudinal Study",
				URL:   "https://medicaljournal.example",
				Content: "Long-term epidemiological cohorts demonstrate a correlation between daily chlorogenic acid intake via black coffee and improved insulin sensitivity. The mechanism involves the upregulation of GLUT4 transporters in skeletal muscle tissues, contradicting short-term acute trials where caffeine temporarily impairs glucose tolerance.",
			},
			{
				ID:    2,
				Title: "Desiccant Efficacy in Mobile Consumer Electronics Liquid Damage Mitigation",
				URL:   "https://engineering.example",
				Content: "Experimental testing of internal device dehydration rates indicates that uncooked Oryza sativa (white rice) behaves as an inefficient passive desiccant compared to synthetic amorphous silica gel packets. Rice starch particulate matter introduces micro-debris into structural components, accelerating galvanic corrosion when combined with residual aqueous solutions.",
			},
			{
				ID:    3,
				Title: "Biomechanical Force Distribution Across Varying Urban Running Surfaces",
				URL:   "https://sportsbiomech.example",
				Content: "Ground reaction force (GRF) analysis reveals that Portland cement concrete yields an elastic modulus significantly higher than asphalt concrete. Running exclusively on rigid pavement configurations increases peak tibial acceleration and places immense eccentric loading constraints on the tibialis anterior muscle complex, inducing micro-trauma.",
			},
			{
				ID:    4,
				Title: "Post-Quantum Cryptography: Algorithmic Resilience of ML-KEM Frameworks",
				URL:   "https://cybersecurity-review.example",
				Content: "The transition to post-quantum cryptographic primitives relies heavily on module lattice-based key encapsulation mechanisms like ML-KEM (Kyber). Unlike RSA factor de-factorization via Shor's algorithm, lattice problems remain structurally computationally hard for quantum architectures, provided token vector dimensions maintain adequate noise parameters.",
			},
			{
				ID:    5,
				Title: "Logistical Supply Chains and Agrarian Subsistence in the Late Roman Empire",
				URL:   "https://history-archaeology.example",
				Content: "The maintenance of the imperial Roman Annona system required centralized maritime shipping lanes connecting the Nile delta to Ostia. Bureaucratic requisition logs indicate that state-subsidized grain redistribution insulated urban plebeian populations from regional crop failures, though price caps ultimately disincentivized local domestic cultivation inside the Italian peninsula.",
			},
			{
				ID:    6,
				Title: "Neurobiological Pathways of Sleep Deprivation and Cortisol Pulsatility",
				URL:   "https://neuroscience-journal.example",
				Content: "Disruption of the human suprachiasmatic nucleus via acute sleep restriction alters the normal pulsatile release of cortisol from the adrenal cortex. Prolonged elevation of circulating glucocorticoids impairs synaptic plasticity within the hippocampus, accelerating working memory decay and mimicking early clinical stages of chronic metabolic syndrome.",
			},
		},
	}
}

// search scours DB against expanded subqueries, return ranked docs
func (hr *HybridRetriever) Search(subQueries []string) []SourceDocument {
	type docMatch struct {
		doc   SourceDocument
		score float64
	}

	matchMap := make(map[int]*docMatch)

	// clean + parse subquery tokens and create strict eval signature
	for _, query := range subQueries {
		tokens := strings.Fields(strings.ToLower(query))
		
		for _, doc := range hr.Database {
			contentLower := strings.ToLower(doc.Content)
			titleLower := strings.ToLower(doc.Title)
			var matchCount float64

			for _, token := range tokens {
				// strip structural syntax symbols or words if they slip through model
				if len(token) <= 2 {
					continue
				}

				// keyword match density calculations
				if strings.Contains(contentLower, token) {
					matchCount += 1.0
				}
				if strings.Contains(titleLower, token) {
					matchCount += 2.0 // title matches weighted heavier than body copy
				}
			}

			if matchCount > 0 {
				// normalize by length to avoid biasing long docs
				finalScore := matchCount / float64(len(tokens)+1)

				if existing, found := matchMap[doc.ID]; found {
					if finalScore > existing.score {
						existing.score = finalScore
					}
				} else {
					matchMap[doc.ID] = &docMatch{doc: doc, score: finalScore}
				}
			}
		}
	}

	// unpack matched maps into sorted results slice
	var matchedResults []SourceDocument
	for _, match := range matchMap {
		match.doc.DensityScore = match.score
		matchedResults = append(matchedResults, match.doc)
	}

	// simple bubble sort to order by matching score, desc
	for i := 0; i < len(matchedResults); i++ {
		for j := i + 1; j < len(matchedResults); j++ {
			if matchedResults[i].DensityScore < matchedResults[j].DensityScore {
				matchedResults[i], matchedResults[j] = matchedResults[j], matchedResults[i]
			}
		}
	}

	return matchedResults
}
