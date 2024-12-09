package models

type QueryPostRequest struct {
	// Text of the query.
	Text string `json:"text"`

	// NoContext indicates context should not be used to populate
	// chat models.
	NoContext bool `json:"no-context"`
}
