package utils

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
)

func WriteJSON(w http.ResponseWriter, status int, data interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(data)
}

func ReadJSON(r *http.Request, dst interface{}) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(dst)
}

func ErrorResponse(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, map[string]string{
		"error": message,
	})
}

func SuccessResponse(w http.ResponseWriter, data interface{}) {
	response := map[string]interface{}{
		"success": true,
		"data":    data,
	}
	WriteJSON(w, http.StatusOK, response)
}

func GenerateUUID() string {
	return uuid.New().String()
}

func CalculateHash(data []byte, algorithm string) (string, error) {
	switch algorithm {
	case "sha256":
		hash := sha256.Sum256(data)
		return hex.EncodeToString(hash[:]), nil
	case "md5":
		hash := md5.Sum(data)
		return hex.EncodeToString(hash[:]), nil
	default:
		return "", fmt.Errorf("unsupported hash algorithm: %s", algorithm)
	}
}

func ValidateUUID(u string) bool {
	_, err := uuid.Parse(u)
	return err == nil
}
