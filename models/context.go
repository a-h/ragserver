package models

type ContextPostRequest struct {
	Text string `json:"text"`
}

type ContextPostResponse struct {
	Results []ContextDocument `json:"results"`
}

type ContextDocument struct {
	Text      string    `json:"text"`
	Embedding []float32 `json:"embedding"`
	Distance  float64   `json:"distance"`
	URL       string    `json:"url"`
	Title     string    `json:"title"`
	Summary   string    `json:"summary"`
}
