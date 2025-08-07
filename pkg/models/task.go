package models

import (
	"encoding/json"
	"time"
)

type TaskStatus struct {
	TaskID     string  `json:"task_id"`
	Status     string  `json:"status"` // pending, processing, completed, failed
	Message    string  `json:"message"`
	Percentage int     `json:"percentage"`
	Result     *Result `json:"result,omitempty"`
	Error      string  `json:"error,omitempty"`
	CreatedAt  string  `json:"created_at"`
}

type Result struct {
	DailyChart  string `json:"daily_chart"`
	HourlyChart string `json:"hourly_chart"`
}

func (ts *TaskStatus) ToJSON() string {
	data, _ := json.Marshal(ts)
	return string(data)
}

type Task struct {
	ID          string
	Symbol      string
	Market      string
	Status      string
	Message     string
	Percentage  int
	Result      *Result
	Error       string
	CreatedAt   time.Time
	subscribers map[chan *TaskStatus]bool
}

func (t *Task) ToStatus() *TaskStatus {
	return &TaskStatus{
		TaskID:     t.ID,
		Status:     t.Status,
		Message:    t.Message,
		Percentage: t.Percentage,
		Result:     t.Result,
		Error:      t.Error,
		CreatedAt:  t.CreatedAt.Format(time.RFC3339),
	}
}
