package main

import (
	"html/template"
	"log"
	"net/http"
	"source-first-search/internal/pipeline"
)

// bind query tracing steps and results into HTML templates
type PageData struct {
	Query      string
	SubQueries []string
	Results    []pipeline.SourceDocument
}

func main() {
	// parse template on startup for runtime performance
	tmpl, err := template.ParseFiles("templates/index.html")
	if err != nil {
		log.Fatalf("Critical Error loading templates: %v", err)
	}

	// init pipeline engines
	expander := pipeline.NewQueryExpander()
	retriever := pipeline.NewHybridRetriever()
	reranker := pipeline.NewEngineReranker()

	// primary server-rendering entrypoint
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("q")
		data := PageData{Query: query}

		if query != "" {
			log.Printf("Processing incoming full-syntax query: %s", query)

			// semantic fan-out
			subQueries, err := expander.FanOut(query)
			if err != nil {
				log.Printf("Expansion engine connection error: %v. Triggering fallback configuration.", err)
				subQueries = []string{query}
			}
			data.SubQueries = subQueries
			log.Printf("Generated Concept Clusters: %v", subQueries)

			// search hybrid keyword memory index
			rawResults := retriever.Search(subQueries)
			log.Printf("Database retrieval phase completed. Found %d raw source candidates.", len(rawResults))

			finalResults := reranker.Rerank(query, rawResults)
			data.Results = finalResults
			log.Printf("Finished reranking. Served %d deeply relevant sources.", len(finalResults))
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		tmpl.Execute(w, data)
	})

	log.Println("!!! Search Engine running live at http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
