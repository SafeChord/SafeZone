package event

type CovidEvent struct {
	EventType string `json:"event_type"`
	EventTime int64  `json:"event_time"`
	TraceID   string `json:"trace_id"`
	Payload   struct {
		Date   string `json:"date"`
		City   string `json:"city"`
		Region string `json:"region"`
		Cases  int    `json:"cases"`
	} `json:"payload"`
	Version string `json:"version"`
}
