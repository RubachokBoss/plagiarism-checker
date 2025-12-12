package models

import (
	"time"
)

type Student struct {
	ID        string    `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	Email     string    `json:"email" db:"email"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type StudentWithStats struct {
	Student
	TotalWorks    int `json:"total_works" db:"total_works"`
	AnalyzedWorks int `json:"analyzed_works" db:"analyzed_works"`
	PendingWorks  int `json:"pending_works" db:"pending_works"`
}
