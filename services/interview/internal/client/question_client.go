package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"easyoffer/interview/internal/domain"
)

var ErrQuestionServiceUnexpectedStatus = errors.New("question service returned unexpected status")

type QuestionClient interface {
	ListQuestions(ctx context.Context, params ListQuestionsParams) ([]domain.QuestionSnapshot, error)
}

type httpQuestionClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewHTTPQuestionClient(baseURL string, timeout time.Duration) QuestionClient {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	return &httpQuestionClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *httpQuestionClient) ListQuestions(ctx context.Context, params ListQuestionsParams) ([]domain.QuestionSnapshot, error) {
	endpoint, err := url.Parse(c.baseURL + "/questions")
	if err != nil {
		return nil, err
	}

	query := endpoint.Query()
	if params.Category != "" {
		query.Set("category", params.Category)
	}
	if params.AnswerFormat != "" {
		query.Set("answer_format", params.AnswerFormat)
	}
	if params.Language != "" {
		query.Set("language", params.Language)
	}
	if params.Limit > 0 {
		query.Set("limit", strconv.Itoa(params.Limit))
	}
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, err
	}
	if userID := strings.TrimSpace(params.UserID); userID != "" {
		req.Header.Set("X-User-ID", userID)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("%w: status=%d body=%s", ErrQuestionServiceUnexpectedStatus, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload struct {
		Questions []domain.QuestionSnapshot `json:"questions"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	return payload.Questions, nil
}

