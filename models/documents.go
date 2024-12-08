package models

type DocumentsPostRequest struct {
	Document Document `json:"document"`
}

type Document struct {
	URL     string `json:"url"`
	Title   string `json:"title"`
	Text    string `json:"text"`
	Summary string `json:"summary"`
}

type DocumentsPostResponse struct {
	ID int64 `json:"id"`
}
