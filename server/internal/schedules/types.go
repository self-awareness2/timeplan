package schedules

type Item struct {
	ID               int64  `json:"id"`
	Title            string `json:"title"`
	Description      string `json:"description"`
	Date             string `json:"date"`
	StartTime        string `json:"startTime"`
	EndTime          string `json:"endTime"`
	Repeat           string `json:"repeat"`
	Priority         string `json:"priority"`
	Status           string `json:"status"`
	ExecutionStatus  string `json:"executionStatus"`
	ActualStartAt    string `json:"actualStartAt"`
	ActualEndAt      string `json:"actualEndAt"`
	ExecutionMinutes int    `json:"executionMinutes"`
	FailureReason    string `json:"failureReason"`
	Category         string `json:"category"`
	CreatedAt        string `json:"createdAt"`
	UpdatedAt        string `json:"updatedAt"`
	HasTime          bool   `json:"hasTime"`
}

type Draft struct {
	Title           string `json:"title"`
	Description     string `json:"description"`
	Date            string `json:"date"`
	StartTime       string `json:"startTime"`
	EndTime         string `json:"endTime"`
	Repeat          string `json:"repeat"`
	Priority        string `json:"priority"`
	Status          string `json:"status"`
	ExecutionStatus string `json:"executionStatus"`
	FailureReason   string `json:"failureReason"`
	Category        string `json:"category"`
}

type ActionRequest struct {
	Action string         `json:"action"`
	Params map[string]any `json:"params"`
}
