# ML Result Ranking Engine w/ no chatbot
### Semantic source discovery to maximize depth with no summaries or hallucination.

## Philosophy
Due to the rise of AI chatbots and their integrations into search engines, the average user is less accustomed to search query optimization, and more likely to ask questions in full sentences, better-suited for the integrated AI summary than getting optimal search results due to filler words ("does", "would", "is").
My goal with this project is to utilize machine learning under the hood to accomodate long-syntax queries and return meaningfully relevant results through Retrieval-Augmented Generation (RAG). The idea here is to reap the benefits of the retrieval engine for maximally relevant results, without summarizing a shortlist of results as fact, with the potential of hallucination as well.

# Pipeline architecture
[Conversational query]\
↓
1. Query expansion (fan-out): generates 3-5 technical subqueries.\
↓
2. Hybrid retrieval (BM25 + vector): scours DB for semantic + keyword matches.\
↓
3. Cross-encoder reranking: evaluates source depth/density.\
↓
4. Raw source presentation: displays untouched docs with UX streaming interval pipeline to keep user engaged.\

## Local prod testing + verification

1. Boot background vector DB container:
   ```bash
   docker compose up -d
   ```
2. Ingest raw text data files into the vector space:
   ```bash
   go run cmd/ingest/main.go
   ```
3. Start web server binary:
   ```bash
   go run cmd/server/main.go
   ```
4. Load `http://localhost:8080` in browser + execute this sample conversational query:
   > **"Will quantum computers break encryption or are there algorithms that can stop them?"**
