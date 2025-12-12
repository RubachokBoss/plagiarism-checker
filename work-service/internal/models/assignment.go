package models

import (
	"time"
)

type Assignment struct {
	ID          string    `json:"id" db:"id"`
	Title       string    `json:"title" db:"title"`
	Description string    `json:"description" db:"description"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

type AssignmentWithStats struct {
	Assignment
	TotalWorks    int `json:"total_works" db:"total_works"`
	AnalyzedWorks int `json:"analyzed_works" db:"analyzed_works"`
	PendingWorks  int `json:"pending_works" db:"pending_works"`
}
