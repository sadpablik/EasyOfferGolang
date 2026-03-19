package repository

import (
	"easyoffer/question/internal/domain"
	"easyoffer/question/internal/events"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var ErrQuestionAlreadyExists = errors.New("question already exists")

type QuestionRepository interface {
	Create(question *domain.Question) error
	Update(question *domain.Question) error
	Delete(questionID string) error
	UpsertReview(review *domain.QuestionReview) error
	GetReviewsByUser(userID, status string) ([]*domain.QuestionReview, error)
	GetAll(userID, category, status, answerFormat, language, searchQuery string, unreviewed bool, limit, offset int, sortBy, order string) ([]*domain.Question, int64, error)
	GetByID(id string) (*domain.Question, error)
	GetReviewByUserAndQuestion(userID, questionID string) (*domain.QuestionReview, error)
	GetMyQuestions(userID, status, category string, limit, offset int) ([]*domain.QuestionWithReview, int64, error)
}

type questionRepository struct {
	db *gorm.DB
}

func NewQuestionRepository(db *gorm.DB) QuestionRepository {
	return &questionRepository{db: db}
}

func (r *questionRepository) Create(question *domain.Question) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(question).Error; err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return ErrQuestionAlreadyExists
			}
			return err
		}

		// Пишем outbox в той же транзакции, что и бизнес-данные.
		outbox, err := newOutboxEvent(events.EventQuestionCreated, question, "")
		if err != nil {
			return err
		}
		return tx.Create(outbox).Error
	})
}

func (r *questionRepository) Update(question *domain.Question) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(question).Error; err != nil {
			return err
		}

		// Обновление вопроса и событие фиксируются атомарно.
		outbox, err := newOutboxEvent(events.EventQuestionUpdated, question, "")
		if err != nil {
			return err
		}
		return tx.Create(outbox).Error
	})
}

func (r *questionRepository) UpsertReview(review *domain.QuestionReview) error {
	var existing domain.QuestionReview

	err := r.db.Where("user_id = ? AND question_id = ?", review.UserID, review.QuestionID).First(&existing).Error
	if err == nil {
		existing.Status = review.Status
		existing.UserAnswer = review.UserAnswer
		existing.Note = review.Note
		existing.ReviewedAt = review.ReviewedAt
		return r.db.Save(&existing).Error
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return r.db.Create(review).Error
	}

	return err
}

func (r *questionRepository) GetReviewsByUser(userID, status string) ([]*domain.QuestionReview, error) {
	var reviews []*domain.QuestionReview
	if status != "" {
		reviews = make([]*domain.QuestionReview, 0)
		err := r.db.Where("user_id = ? AND status = ?", userID, status).Find(&reviews).Error
		return reviews, err
	}
	err := r.db.Where("user_id = ?", userID).Find(&reviews).Error
	return reviews, err
}

func (r *questionRepository) GetAll(userID, category, status, answerFormat, language, searchQuery string, unreviewed bool, limit, offset int, sortBy, order string) ([]*domain.Question, int64, error) {
	questions := make([]*domain.Question, 0)
	query := r.db.Model(&domain.Question{})

	if category != "" {
		query = query.Where("questions.category = ?", category)
	}

	if status != "" {
		query = query.
			Joins("JOIN question_reviews qr ON qr.question_id = questions.id AND qr.user_id = ?", userID).
			Where("qr.status = ?", status)
	}

	if unreviewed {
		query = query.
			Joins("LEFT JOIN question_reviews qru ON qru.question_id = questions.id AND qru.user_id = ?", userID).
			Where("qru.id IS NULL")
	}

	if answerFormat != "" {
		query = query.Where("questions.answer_format = ?", answerFormat)
	}

	if language != "" {
		query = query.Where("LOWER(questions.language) = LOWER(?)", language)
	}

	if searchQuery != "" {
		pattern := "%" + searchQuery + "%"
		query = query.Where("(questions.title ILIKE ? OR questions.content ILIKE ?)", pattern, pattern)
	}

	total := int64(0)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	sortColumn := "questions.created_at"
	if sortBy == "title" {
		sortColumn = "questions.title"
	}

	sortOrder := "desc"
	if order == "asc" {
		sortOrder = "asc"
	}

	err := query.
		Order(sortColumn + " " + sortOrder).
		Limit(limit).
		Offset(offset).
		Find(&questions).Error

	return questions, total, err
}

func (r *questionRepository) GetByID(id string) (*domain.Question, error) {
	var question domain.Question
	err := r.db.Where("id = ?", id).First(&question).Error
	if err != nil {
		return nil, err
	}
	return &question, nil
}

func (r *questionRepository) GetReviewByUserAndQuestion(userID, questionID string) (*domain.QuestionReview, error) {
	var review domain.QuestionReview
	err := r.db.Where("user_id = ? AND question_id = ?", userID, questionID).First(&review).Error
	if err != nil {
		return nil, err
	}
	return &review, nil
}

// GetMyQuestions returns questions created by the user (author_id = userID),
// with optional left join to the user's review (review_status, reviewed_at).
// status filter: only questions the author has reviewed with that status.
// category filter: question category.
func (r *questionRepository) GetMyQuestions(userID, status, category string, limit, offset int) ([]*domain.QuestionWithReview, int64, error) {
	uid := strings.TrimSpace(userID)
	if uid == "" {
		return nil, 0, errors.New("user id is required for my questions")
	}

	rows := make([]domain.QuestionWithReview, 0)

	// Base: questions where author_id = user; join own review for review_status/reviewed_at.
	countQuery := r.db.Table("questions").
		Joins("LEFT JOIN question_reviews qr ON qr.question_id = questions.id AND qr.user_id = ?", uid).
		Where("questions.author_id = ?", uid)

	if status != "" {
		countQuery = countQuery.Where("qr.status = ?", status)
	}
	if category != "" {
		countQuery = countQuery.Where("questions.category = ?", category)
	}

	total := int64(0)
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	dataQuery := r.db.Table("questions").
		Select(`
            questions.id,
            questions.title,
            questions.content,
            questions.category,
            questions.answer_format,
            questions.language,
            questions.starter_code,
            questions.author_id,
            questions.created_at,
            qr.status AS review_status,
            qr.reviewed_at
        `).
		Joins("LEFT JOIN question_reviews qr ON qr.question_id = questions.id AND qr.user_id = ?", uid).
		Where("questions.author_id = ?", uid)

	if status != "" {
		dataQuery = dataQuery.Where("qr.status = ?", status)
	}
	if category != "" {
		dataQuery = dataQuery.Where("questions.category = ?", category)
	}

	err := dataQuery.
		Order("questions.created_at DESC").
		Limit(limit).
		Offset(offset).
		Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}

	questions := make([]*domain.QuestionWithReview, 0, len(rows))
	for i := range rows {
		questions = append(questions, &rows[i])
	}

	return questions, total, nil
}

func (r *questionRepository) Delete(questionID string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("question_id = ?", questionID).Delete(&domain.QuestionReview{}).Error; err != nil {
			return err
		}

		result := tx.Where("id = ?", questionID).Delete(&domain.Question{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}

		// Событие удаления пишем только если удаление реально произошло.
		outbox, err := newOutboxEvent(events.EventQuestionDeleted, nil, questionID)
		if err != nil {
			return err
		}
		return tx.Create(outbox).Error
	})
}

func newOutboxEvent(eventType string, q *domain.Question, questionID string) (*domain.OutboxEvent, error) {
	id := strings.TrimSpace(questionID)
	if q != nil && id == "" {
		id = strings.TrimSpace(q.ID)
	}
	if id == "" {
		return nil, errors.New("question id is required for outbox")
	}

	payload := events.QuestionPayload{
		QuestionID: id,
	}
	if q != nil {
		payload.AuthorID = q.AuthorID
		payload.Title = q.Title
		payload.Content = q.Content
		payload.Category = q.Category
		payload.AnswerFormat = q.AnswerFormat
		payload.Language = q.Language
		payload.StarterCode = q.StarterCode
		payload.CreatedAt = q.CreatedAt
	}

	occurredAt := time.Now().UTC()
	eventID := uuid.NewString()

	wireEvent := events.QuestionEvent{
		EventID:    eventID,
		EventType:  eventType,
		OccurredAt: occurredAt,
		Version:    1,
		Payload:    payload,
	}

	raw, err := json.Marshal(wireEvent)
	if err != nil {
		return nil, err
	}

	return &domain.OutboxEvent{
		ID:            eventID,
		AggregateType: "question",
		AggregateID:   id,
		EventType:     eventType,
		Payload:       string(raw),
		Status:        domain.OutboxStatusPending,
		Attempts:      0,
		NextRetryAt:   occurredAt,
		CreatedAt:     occurredAt,
	}, nil
}

func (r *questionRepository) ListOutboxForDispatch(limit int, now time.Time) ([]*domain.OutboxEvent, error) {
	if limit <= 0 {
		limit = 50
	}

	rows := make([]domain.OutboxEvent, 0, limit)
	err := r.db.
		Where("status IN ? AND next_retry_at <= ?", []domain.OutboxStatus{
			domain.OutboxStatusPending,
			domain.OutboxStatusFailed,
		}, now).
		Order("created_at ASC").
		Limit(limit).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make([]*domain.OutboxEvent, 0, len(rows))
	for i := range rows {
		result = append(result, &rows[i])
	}
	return result, nil
}

func (r *questionRepository) MarkOutboxSent(eventID string, sentAt time.Time) error {
	return r.db.Model(&domain.OutboxEvent{}).
		Where("id = ?", strings.TrimSpace(eventID)).
		Updates(map[string]interface{}{
			"status":     domain.OutboxStatusSent,
			"sent_at":    sentAt,
			"last_error": "",
		}).Error
}

func (r *questionRepository) MarkOutboxRetry(eventID string, nextRetryAt time.Time, lastError string) error {
	errText := strings.TrimSpace(lastError)
	if len(errText) > 1024 {
		errText = errText[:1024]
	}

	return r.db.Model(&domain.OutboxEvent{}).
		Where("id = ?", strings.TrimSpace(eventID)).
		Updates(map[string]interface{}{
			"status":        domain.OutboxStatusFailed,
			"attempts":      gorm.Expr("attempts + 1"),
			"next_retry_at": nextRetryAt,
			"last_error":    errText,
		}).Error
}

func (r *questionRepository) CountOutboxByStatus() (map[domain.OutboxStatus]int64, error) {
	type statusCountRow struct {
		Status domain.OutboxStatus
		Count  int64
	}

	rows := make([]statusCountRow, 0, 3)
	if err := r.db.Model(&domain.OutboxEvent{}).
		Select("status, COUNT(*) AS count").
		Group("status").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	counts := map[domain.OutboxStatus]int64{
		domain.OutboxStatusPending: 0,
		domain.OutboxStatusFailed:  0,
		domain.OutboxStatusSent:    0,
	}
	for _, row := range rows {
		counts[row.Status] = row.Count
	}

	return counts, nil
}
