package main

import (
	"context"
	"hash/fnv"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"source-first-search/internal/pipeline"

	"github.com/qdrant/go-client/qdrant"
)

type LocalFile struct {
	Title   string
	URL     string
	Content string
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// instantiate DB conn + embedding generator
	client, err := qdrant.NewClient(&qdrant.Config{
		Host: "localhost",
		Port: 6334,
	})
	if err != nil {
		log.Fatalf("failed to connect to qdrant container: %v", err)
	}
	defer client.Close()

	embedEngine := pipeline.NewEmbeddingsEngine()
	collectionName := "sources"
	sourceDir := "./data/sources"

	files, err := os.ReadDir(sourceDir)
	if err != nil {
		log.Fatalf("failed to read data directory: %v", err)
	}

	var documents []LocalFile
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".txt") {
			path := filepath.Join(sourceDir, f.Name())
			content, err := os.ReadFile(path)
			if err != nil {
				log.Printf("failed to read file %s: %v", path, err)
				continue
			}

			// extract structural metadata from raw filename variations
			title := strings.TrimSuffix(f.Name(), ".txt")
			title = strings.ReplaceAll(title, "-", " ")

			documents = append(documents, LocalFile{
				Title:   title,
				URL:     "local://" + f.Name(),
				Content: string(content),
			})
		}
	}

	if len(documents) == 0 {
		log.Printf("no .txt files located in %s. populate data directory to run ingestion.", sourceDir)
		return
	}

	log.Printf("vectorizing and ingesting %d files into qdrant index...", len(documents))

	var points []*qdrant.PointStruct
	for _, doc := range documents {
		// convert textual body copy to deep semantic vectors
		vector, err := embedEngine.GetVector(doc.Content)
		if err != nil {
			log.Printf("failed to generate vector embedding for %s: %v", doc.Title, err)
			continue
		}

		// generate id from doc string footprint
		hasher := fnv.New64a()
		hasher.Write([]byte(doc.URL))
		docID := hasher.Sum64()

		// build structural payload record 
		payload := map[string]*qdrant.Value{
			"title":   qdrant.NewValueString(doc.Title),
			"url":     qdrant.NewValueString(doc.URL),
			"content": qdrant.NewValueString(doc.Content),
		}

		points = append(points, &qdrant.PointStruct{
			Id:      qdrant.NewIDNum(docID),
			Vectors: qdrant.NewVectors(vector...),
			Payload: payload,
		})
	}

	// stream items directly to cluster
	if len(points) > 0 {
		_, err = client.Upsert(ctx, &qdrant.UpsertPoints{
			CollectionName: collectionName,
			Points:         points,
		})
		if err != nil {
			log.Fatalf("database ingestion matrix payload rejected: %v", err)
		}
		log.Printf("successfully populated %d high-fidelity vectors to index.", len(points))
	}
}
