package model

type KafkaEmailEvent struct {
	To          string `json:"to"`
	Subject     string `json:"subject"`
	ContentType string `json:"content_type"` // e.g., "text/html"
	Body        string `json:"body"`
}
