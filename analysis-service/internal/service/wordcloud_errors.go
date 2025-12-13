package service

import "errors"

// Типизированные ошибки для корректного маппинга на HTTP-коды в delivery-слое.
var (
	// Ошибки валидации/состояния домена.
	ErrInvalidWorkID    = errors.New("invalid work_id")
	ErrReportNotFound   = errors.New("report not found for this work")
	ErrFileIDEmpty      = errors.New("file_id is empty for this work")
	ErrFileContentEmpty = errors.New("file content is empty")

	// Ошибки внешних зависимостей.
	ErrFileServiceError = errors.New("file service error")
	ErrQuickChartError  = errors.New("quickchart error")
)
