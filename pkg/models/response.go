package models

type AnalysisRequest struct {
	Symbol string `json:"symbol" binding:"required"`
	Market string `json:"market" binding:"required"`
}

type AnalysisResponse struct {
	TaskID              string `json:"task_id"`
	Status              string `json:"status"`
	CreatedAt           string `json:"created_at"`
	EstimatedCompletion string `json:"estimated_completion,omitempty"`
	ResultURL           string `json:"result_url,omitempty"`
}

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}
