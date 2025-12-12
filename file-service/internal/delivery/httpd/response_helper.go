package httpd

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

// Вспомогательные функции для работы с запросами
func getIntQueryParam(r *http.Request, key string, defaultValue int) int {
	value := r.URL.Query().Get(key)
	if value == "" {
		return defaultValue
	}

	intValue, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}

	return intValue
}

func getInt64QueryParam(r *http.Request, key string, defaultValue int64) int64 {
	value := r.URL.Query().Get(key)
	if value == "" {
		return defaultValue
	}

	intValue, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return defaultValue
	}

	return intValue
}

func getBoolQueryParam(r *http.Request, key string, defaultValue bool) bool {
	value := r.URL.Query().Get(key)
	if value == "" {
		return defaultValue
	}

	boolValue, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}

	return boolValue
}

// Функции для отправки ответов
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	response := map[string]interface{}{
		"error": map[string]interface{}{
			"code":    status,
			"message": message,
			"type":    http.StatusText(status),
		},
		"success":   false,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	writeJSON(w, status, response)
}

func writeSuccess(w http.ResponseWriter, data interface{}) {
	response := map[string]interface{}{
		"success":   true,
		"data":      data,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	writeJSON(w, http.StatusOK, response)
}

// Функция для проверки строки на подстроку (замена рекурсивной contains)
func stringContains(s, substr string) bool {
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
