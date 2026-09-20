package pipeline

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/qdrant/go-client/qdrant"
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
	qdrantClient *qdrant.Client
	embedEngine  *EmbeddingsEngine
	collection   string
}

// instantiates isolated data repo with diverse entries
func NewHybridRetriever() *HybridRetriever {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// establish gRPC connection to local Dockerized Qdrant instance
	client, err := qdrant.NewClient(&qdrant.Config{
		Host: "localhost",
		Port: 6334,
	})
	if err != nil {
		log.Fatalf("failed to connect to qdrant: %v", err)
	}

	collectionName := "sources"
	embedEngine := NewEmbeddingsEngine()

	// verify collection space exists OR new build configured for all-minilm dimensions
	exists, err := client.CollectionExists(ctx, collectionName)
	if err == nil && !exists {
		err = client.CreateCollection(ctx, &qdrant.CreateCollection{
			CollectionName: collectionName,
			VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
				Size:     384,
				Distance: qdrant.Distance_Cosine,
			}),
		})
		if err != nil {
			log.Fatalf("failed to create vector collection: %v", err)
		}
	}

	return &HybridRetriever{
		qdrantClient: client,
		embedEngine:  embedEngine,
		collection:   collectionName,
	}
}

// search scours DB against expanded subqueries, return ranked docs
func (hr *HybridRetriever) Search(subQueries []string) []SourceDocument {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	type docMatch struct {
		doc   SourceDocument
		score float64
	}
	matchMap := make(map[int]*docMatch)

	// query database with vectors generated from concept clusters
	for _, query := range subQueries {
		if strings.TrimSpace(query) == "" {
			continue
		}

		// generate vector array from local ollama engine
		vector, err := hr.embedEngine.GetVector(query)
		if err != nil {
			log.Printf("embedding conversion failed for query chunk: %v", err)
			continue
		}

		// fetch closest multi-dimensional math alignments from vector index
		searchResult, err := hr.qdrantClient.Query(ctx, &qdrant.QueryPoints{
			CollectionName: hr.collection,
			Query:          qdrant.NewQuery(vector...),
			Limit:          qdrant.PtrOf(uint64(3)),
			WithPayload:    qdrant.NewWithPayload(true),
		})
		if err != nil {
			log.Printf("qdrant search operation failed: %v", err)
			continue
		}

		// unpack data objects back into structural application records
		for _, point := range searchResult {
			payload := point.Payload
			
			titleAttr := payload["title"].GetStringValue()
			urlAttr := payload["url"].GetStringValue()
			contentAttr := payload["content"].GetStringValue()
			idAttr := int(point.Id.GetNum())
			vectorScore := float64(point.Score)

			// calculate secondary keyword intersection bonus for hybrid search verification
			tokens := strings.Fields(strings.ToLower(query))
			contentLower := strings.ToLower(contentAttr)
			var keywordBonus float64
			for _, token := range tokens {
				token = strings.Trim(token, `.,"?!()[]{}:;`)
				if len(token) > 3 && strings.Contains(contentLower, token) {
					keywordBonus += 0.05
				}
			}

			finalCombinedScore := vectorScore + keywordBonus

			if existing, found := matchMap[idAttr]; found {
				if finalCombinedScore > existing.score {
					existing.score = finalCombinedScore
				}
			} else {
				matchMap[idAttr] = &docMatch{
					doc: SourceDocument{
						ID:      idAttr,
						Title:   titleAttr,
						URL:     urlAttr,
						Content: contentAttr,
					},
					score: finalCombinedScore,
				}
			}
		}
	}

	// unpack matching structures into output slice
	var finalResults []SourceDocument
	for _, match := range matchMap {
		match.doc.DensityScore = match.score
		finalResults = append(finalResults, match.doc)
	}

	// simple bubble sort to order by matching score, desc
	for i := 0; i < len(finalResults); i++ {
		for j := i + 1; j < len(finalResults); j++ {
			if finalResults[i].DensityScore < finalResults[j].DensityScore {
				finalResults[i], finalResults[j] = finalResults[j], finalResults[i]
			}
		}
	}

	return finalResults
}

// close cleanly handles long-lived database connection exit handshakes
func (hr *HybridRetriever) Close() {
	if hr.qdrantClient != nil {
		hr.qdrantClient.Close()
	}
}
