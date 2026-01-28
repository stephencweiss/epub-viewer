package web

import (
	"encoding/json"
	"math"
	"net/http"
	"strconv"

	"epub-reader/pkg/storage"
)

// MetricsResponse contains normalized metrics for a single author.
type MetricsResponse struct {
	Labels []string  `json:"labels"`
	Data   []float64 `json:"data"`
}

// CompareResponse contains data for comparing two authors.
type CompareResponse struct {
	Labels  []string      `json:"labels"`
	Author1 AuthorDataset `json:"author1"`
	Author2 AuthorDataset `json:"author2"`
}

// AuthorDataset contains chart data for one author.
type AuthorDataset struct {
	Label string    `json:"label"`
	Data  []float64 `json:"data"`
}

// apiAuthorMetrics returns JSON metrics for a single author's corpus.
func (s *Server) apiAuthorMetrics(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		jsonError(w, "Invalid author ID", http.StatusBadRequest)
		return
	}

	corpus, err := s.store.GetCorpusAnalysis(id)
	if err != nil {
		jsonError(w, "Author not found", http.StatusNotFound)
		return
	}

	resp := MetricsResponse{
		Labels: []string{"Vocabulary", "Readability", "Sentence Len", "Word Len", "Dialogue"},
		Data:   normalizeMetrics(corpus),
	}

	jsonResponse(w, resp)
}

// apiCompare returns JSON data for comparing two authors.
func (s *Server) apiCompare(w http.ResponseWriter, r *http.Request) {
	id1, err := strconv.ParseInt(r.URL.Query().Get("author1"), 10, 64)
	if err != nil {
		jsonError(w, "Invalid author1 ID", http.StatusBadRequest)
		return
	}

	id2, err := strconv.ParseInt(r.URL.Query().Get("author2"), 10, 64)
	if err != nil {
		jsonError(w, "Invalid author2 ID", http.StatusBadRequest)
		return
	}

	c1, err := s.store.GetCorpusAnalysis(id1)
	if err != nil {
		jsonError(w, "Author 1 not found", http.StatusNotFound)
		return
	}

	c2, err := s.store.GetCorpusAnalysis(id2)
	if err != nil {
		jsonError(w, "Author 2 not found", http.StatusNotFound)
		return
	}

	resp := CompareResponse{
		Labels: []string{"Vocabulary", "Readability", "Sentence Len", "Word Len", "Dialogue"},
		Author1: AuthorDataset{
			Label: c1.Author.Name,
			Data:  normalizeMetrics(c1),
		},
		Author2: AuthorDataset{
			Label: c2.Author.Name,
			Data:  normalizeMetrics(c2),
		},
	}

	jsonResponse(w, resp)
}

// normalizeMetrics scales corpus metrics to 0-100 for radar chart display.
func normalizeMetrics(c *storage.CorpusAnalysis) []float64 {
	return []float64{
		c.AvgVocabularyRich * 100,         // 0-1 → 0-100
		c.AvgReadability,                   // Already 0-100 scale
		math.Min(c.AvgSentenceLen*4, 100), // ~25 words → 100
		c.AvgWordLen * 20,                  // ~5 chars → 100
		c.AvgDialogueRatio * 100,          // 0-1 → 0-100
	}
}

// jsonResponse writes a JSON response.
func jsonResponse(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// jsonError writes a JSON error response.
func jsonError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
