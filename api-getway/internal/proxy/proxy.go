package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

type Proxy struct {
	target  *url.URL
	proxy   *httputil.ReverseProxy
	logger  zerolog.Logger
	timeout time.Duration
	retries int
}

type ProxyOption func(*Proxy)

func NewProxy(targetURL string, logger zerolog.Logger, options ...ProxyOption) (*Proxy, error) {
	target, err := url.Parse(targetURL)
	if err != nil {
		return nil, err
	}

	p := &Proxy{
		target:  target,
		logger:  logger,
		timeout: 30 * time.Second,
		retries: 3,
	}

	// Применяем опции
	for _, option := range options {
		option(p)
	}

	// Создаем reverse proxy
	p.proxy = httputil.NewSingleHostReverseProxy(target)

	// Настраиваем директор
	p.proxy.Director = p.director

	// Настраиваем транспорт
	transport := &http.Transport{
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	p.proxy.Transport = transport

	// Настраиваем обработчик ошибок
	p.proxy.ErrorHandler = p.errorHandler

	// Настраиваем модификатор ответа
	p.proxy.ModifyResponse = p.modifyResponse

	return p, nil
}

func WithTimeout(timeout time.Duration) ProxyOption {
	return func(p *Proxy) {
		p.timeout = timeout
	}
}

func WithRetries(retries int) ProxyOption {
	return func(p *Proxy) {
		p.retries = retries
	}
}

func (p *Proxy) director(req *http.Request) {
	// Сохраняем оригинальный путь
	originalPath := req.URL.Path

	// Устанавливаем целевой URL
	req.URL.Scheme = p.target.Scheme
	req.URL.Host = p.target.Host
	req.Host = p.target.Host

	// Логируем запрос
	p.logger.Debug().
		Str("method", req.Method).
		Str("original_path", originalPath).
		Str("target_path", req.URL.Path).
		Str("target", p.target.String()).
		Msg("Proxying request")

	// Добавляем заголовки
	req.Header.Set("X-Forwarded-Host", req.Host)
	req.Header.Set("X-Forwarded-For", req.RemoteAddr)
	req.Header.Set("X-Forwarded-Proto", req.URL.Scheme)

	// Если это multipart/form-data, нужно сохранить boundary
	if strings.HasPrefix(req.Header.Get("Content-Type"), "multipart/form-data") {
		// Не изменяем Content-Type для multipart
	} else {
		req.Header.Set("Content-Type", "application/json")
	}
}

func (p *Proxy) errorHandler(w http.ResponseWriter, r *http.Request, err error) {
	p.logger.Error().
		Err(err).
		Str("method", r.Method).
		Str("path", r.URL.Path).
		Str("target", p.target.String()).
		Msg("Proxy error")

	// Стандартизированный ответ об ошибке
	errorResponse := map[string]interface{}{
		"error":     "Service unavailable",
		"message":   "The requested service is temporarily unavailable. Please try again later.",
		"code":      "SERVICE_UNAVAILABLE",
		"path":      r.URL.Path,
		"timestamp": time.Now().UTC(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusServiceUnavailable)

	// В реальной реализации здесь должен быть json.NewEncoder
	encoder := json.NewEncoder(w)
	encoder.Encode(errorResponse)
}

func (p *Proxy) modifyResponse(resp *http.Response) error {
	// Логируем ответ
	p.logger.Debug().
		Str("method", resp.Request.Method).
		Str("path", resp.Request.URL.Path).
		Int("status", resp.StatusCode).
		Str("target", p.target.String()).
		Msg("Proxy response")

	// Добавляем заголовки CORS если их нет
	if resp.Header.Get("Access-Control-Allow-Origin") == "" {
		resp.Header.Set("Access-Control-Allow-Origin", "*")
	}

	// Добавляем заголовок с именем сервиса
	resp.Header.Set("X-Service-Name", p.target.Hostname())

	return nil
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Копируем тело запроса для возможности повторных попыток
	var bodyBytes []byte
	var err error

	if r.Body != nil {
		bodyBytes, err = io.ReadAll(r.Body)
		if err != nil {
			p.errorHandler(w, r, err)
			return
		}
		r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	}

	// Пытаемся выполнить запрос с retry
	for attempt := 0; attempt < p.retries; attempt++ {
		if attempt > 0 {
			// Восстанавливаем тело запроса для повторной попытки
			if bodyBytes != nil {
				r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			}

			// Exponential backoff
			waitTime := time.Duration(attempt) * 100 * time.Millisecond
			time.Sleep(waitTime)

			p.logger.Info().
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Int("attempt", attempt+1).
				Msg("Retrying request")
		}

		// Создаем канал для результата
		resultChan := make(chan error, 1)
		var recorder *httptest.ResponseRecorder
		// Запускаем запрос в горутине с таймаутом
		go func() {
			// Создаем ResponseRecorder для захвата ответа
			recorder = httptest.NewRecorder()
			p.proxy.ServeHTTP(recorder, r)

			// Копируем ответ в оригинальный ResponseWriter
			for k, v := range recorder.Header() {
				w.Header()[k] = v
			}
			w.WriteHeader(recorder.Code)
			w.Write(recorder.Body.Bytes())

			resultChan <- nil
		}()

		// Ждем с таймаутом
		select {
		case <-resultChan:
			// Успешно
			if recorder.Code < 500 {
				return // Успех
			}
			// Если статус 5xx - пробуем снова
		case <-time.After(p.timeout):
			// Таймаут
			if attempt == p.retries-1 {
				p.errorHandler(w, r, context.DeadlineExceeded)
				return
			}
		}
	}

	// Если все попытки исчерпаны
	p.errorHandler(w, r, fmt.Errorf("all %d attempts failed", p.retries))
}
