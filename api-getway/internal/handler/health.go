package handler

import (
	"net/http"
	"time"
)

type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Service   string    `json:"service"`
	Version   string    `json:"version"`
}

type ReadyResponse struct {
	Status    string          `json:"status"`
	Timestamp time.Time       `json:"timestamp"`
	Services  []ServiceStatus `json:"services,omitempty"`
}

type ServiceStatus struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	URL    string `json:"url,omitempty"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

// NewErrorResponse создает стандартизированный ответ об ошибке
func NewErrorResponse(status int, message, code string) (int, ErrorResponse) {
	return status, ErrorResponse{
		Error:   http.StatusText(status),
		Message: message,
		Code:    code,
	}
}

// ServiceUnavailableResponse создает ответ "Сервис недоступен"
func ServiceUnavailableResponse(serviceName string) (int, ErrorResponse) {
	return http.StatusServiceUnavailable, ErrorResponse{
		Error:   "Service Unavailable",
		Message: serviceName + " is temporarily unavailable",
		Code:    "SERVICE_UNAVAILABLE",
	}
}
