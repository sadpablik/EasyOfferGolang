package service

import (
	"context"
	"easyoffer/question/internal/domain"
	"easyoffer/question/internal/events"
	"easyoffer/question/internal/repository"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrInvalidCategory         = errors.New("invalid category")
	ErrInvalidAnswerFormat     = errors.New("invalid answer format")
	ErrQuestionAlreadyExists   = errors.New("question already exists")
	ErrInvalidReviewStatus     = errors.New("invalid review status")
	ErrInvalidQuestionFilter   = errors.New("invalid question filter")
	ErrMissingUserID           = errors.New("missing user id")
	ErrQuestionNotFound        = errors.New("question not found")
	ErrReviewNotFound          = errors.New("review not found")
	ErrForbiddenQuestionUpdate = errors.New("forbidden question update")
	ErrInvalidQuestionPayload  = errors.New("invalid question payload")
	ErrForbiddenQuestionDelete = errors.New("forbidden question delete")
)

type QuestionService interface {
	CreateQuestion(title, content, authorID, category, answerFormat, language, starterCode string) (*domain.Question, error)
	PatchQuestion(questionID, userID string, title, content, category, answerFormat, language, starterCode *string) (*domain.Question, error)
	Delete(questionID, userID string) error
	ReviewQuestion(userID, questionID, status, userAnswer, note string) (*domain.QuestionReview, error)
	GetQuestions(userID, category, status, answerFormat, language, searchQuery string, unreviewed bool, limit, offset int, sortBy, order string) ([]*domain.Question, int64, error)
	GetQuestionByID(id string) (*domain.Question, error)
	GetMyReviews(userID, status string) ([]*domain.QuestionReview, error)
	GetMyQuestionReview(userID, questionID string) (*domain.QuestionReview, error)
	GetMyQuestions(userID, status, category string, limit, offset int) ([]*domain.QuestionWithReview, int64, error)
}

type questionService struct {
	repo      repository.QuestionRepository
	publisher events.Publisher
}

func NewQuestionService(repo repository.QuestionRepository, publisher events.Publisher) QuestionService {
	return &questionService{
		repo:      repo,
		publisher: publisher,
	}
}

func (s *questionService) CreateQuestion(title, content, authorID, category, answerFormat, language, starterCode string) (*domain.Question, error) {
	cat := domain.QuestionCategory(category)
	switch cat {
	case domain.CategoryResume, domain.CategoryTheory, domain.CategoryPractice:
	default:
		return nil, ErrInvalidCategory
	}

	af := domain.AnswerFormat(answerFormat)
	switch af {
	case domain.AnswerFormatText, domain.AnswerFormatCode:
	default:
		return nil, ErrInvalidAnswerFormat
	}

	q := &domain.Question{
		ID:           uuid.NewString(),
		Title:        strings.TrimSpace(title),
		Content:      strings.TrimSpace(content),
		Category:     cat,
		AnswerFormat: af,
		Language:     strings.TrimSpace(language),
		StarterCode:  strings.TrimSpace(starterCode),
		AuthorID:     strings.TrimSpace(authorID),
		CreatedAt:    time.Now().UTC(),
	}

	if err := s.repo.Create(q); err != nil {
		if errors.Is(err, repository.ErrQuestionAlreadyExists) {
			return nil, ErrQuestionAlreadyExists
		}
		return nil, err
	}

	_ = s.publisher.PublishQuestionCreated(context.Background(), q)
	return q, nil
}

func (s *questionService) PatchQuestion(questionID, userID string, title, content, category, answerFormat, language, starterCode *string) (*domain.Question, error) {
	q, err := s.repo.GetByID(strings.TrimSpace(questionID))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrQuestionNotFound
	}
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(q.AuthorID) != strings.TrimSpace(userID) {
		return nil, ErrForbiddenQuestionUpdate
	}

	if title != nil {
		newTitle := strings.TrimSpace(*title)
		if newTitle == "" {
			return nil, ErrInvalidQuestionPayload
		}
		q.Title = newTitle
	}

	if content != nil {
		newContent := strings.TrimSpace(*content)
		if newContent == "" {
			return nil, ErrInvalidQuestionPayload
		}
		q.Content = newContent
	}

	if category != nil {
		cat := domain.QuestionCategory(strings.TrimSpace(*category))
		switch cat {
		case domain.CategoryResume, domain.CategoryTheory, domain.CategoryPractice:
			q.Category = cat
		default:
			return nil, ErrInvalidCategory
		}
	}

	if answerFormat != nil {
		af := domain.AnswerFormat(strings.TrimSpace(*answerFormat))
		switch af {
		case domain.AnswerFormatText, domain.AnswerFormatCode:
			q.AnswerFormat = af
		default:
			return nil, ErrInvalidAnswerFormat
		}
	}

	if language != nil {
		q.Language = strings.TrimSpace(*language)
	}

	if starterCode != nil {
		q.StarterCode = strings.TrimSpace(*starterCode)
	}

	if err := s.repo.Update(q); err != nil {
		return nil, err
	}

	_ = s.publisher.PublishQuestionUpdated(context.Background(), q)

	return q, nil
}

func (s *questionService) ReviewQuestion(userID, questionID, status, userAnswer, note string) (*domain.QuestionReview, error) {
	st := domain.ReviewStatus(status)
	switch st {
	case domain.StatusKnow, domain.StatusDontKnow, domain.StatusRepeat:
	default:
		return nil, ErrInvalidReviewStatus
	}

	review := &domain.QuestionReview{
		ID:         uuid.NewString(),
		UserID:     strings.TrimSpace(userID),
		QuestionID: strings.TrimSpace(questionID),
		Status:     st,
		UserAnswer: strings.TrimSpace(userAnswer),
		Note:       strings.TrimSpace(note),
		ReviewedAt: time.Now().UTC(),
	}

	if err := s.repo.UpsertReview(review); err != nil {
		return nil, err
	}
	return review, nil
}

func (s *questionService) GetQuestions(userID, category, status, answerFormat, language, searchQuery string, unreviewed bool, limit, offset int, sortBy, order string) ([]*domain.Question, int64, error) {
	uid := strings.TrimSpace(userID)

	cat := strings.TrimSpace(category)
	if cat != "" {
		switch domain.QuestionCategory(cat) {
		case domain.CategoryResume, domain.CategoryTheory, domain.CategoryPractice:
		default:
			return nil, 0, ErrInvalidCategory
		}
	}

	st := strings.TrimSpace(status)
	if st != "" {
		if uid == "" {
			return nil, 0, ErrMissingUserID
		}
		switch domain.ReviewStatus(st) {
		case domain.StatusKnow, domain.StatusDontKnow, domain.StatusRepeat:
		default:
			return nil, 0, ErrInvalidReviewStatus
		}
	}

	if unreviewed {
		if uid == "" {
			return nil, 0, ErrMissingUserID
		}
		if st != "" {
			return nil, 0, ErrInvalidQuestionFilter
		}
	}

	af := strings.TrimSpace(answerFormat)
	if af != "" {
		switch domain.AnswerFormat(af) {
		case domain.AnswerFormatText, domain.AnswerFormatCode:
		default:
			return nil, 0, ErrInvalidAnswerFormat
		}
	}

	lang := strings.TrimSpace(language)
	search := strings.TrimSpace(searchQuery)

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	sb := strings.TrimSpace(sortBy)
	if sb != "title" {
		sb = "created_at"
	}

	ord := strings.TrimSpace(order)
	if ord != "asc" {
		ord = "desc"
	}

	return s.repo.GetAll(uid, cat, st, af, lang, search, unreviewed, limit, offset, sb, ord)

}

func (s *questionService) GetQuestionByID(id string) (*domain.Question, error) {
	q, err := s.repo.GetByID(strings.TrimSpace(id))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrQuestionNotFound
	}
	return q, err
}

func (s *questionService) GetMyReviews(userID, status string) ([]*domain.QuestionReview, error) {
	st := strings.TrimSpace(status)
	if st != "" {
		switch domain.ReviewStatus(st) {
		case domain.StatusKnow, domain.StatusDontKnow, domain.StatusRepeat:
		default:
			return nil, ErrInvalidReviewStatus
		}
	}
	return s.repo.GetReviewsByUser(strings.TrimSpace(userID), st)
}

func (s *questionService) GetMyQuestionReview(userID, questionID string) (*domain.QuestionReview, error) {
	review, err := s.repo.GetReviewByUserAndQuestion(strings.TrimSpace(userID), strings.TrimSpace(questionID))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrReviewNotFound
	}
	return review, err
}

func (s *questionService) GetMyQuestions(userID, status, category string, limit, offset int) ([]*domain.QuestionWithReview, int64, error) {
	st := strings.TrimSpace(status)
	if st != "" {
		switch domain.ReviewStatus(st) {
		case domain.StatusKnow, domain.StatusDontKnow, domain.StatusRepeat:
		default:
			return nil, 0, ErrInvalidReviewStatus
		}
	}

	cat := strings.TrimSpace(category)
	if cat != "" {
		switch domain.QuestionCategory(cat) {
		case domain.CategoryResume, domain.CategoryTheory, domain.CategoryPractice:
		default:
			return nil, 0, ErrInvalidCategory
		}
	}

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	return s.repo.GetMyQuestions(strings.TrimSpace(userID), st, cat, limit, offset)
}

func (s *questionService) Delete(questionID, userID string) error {
	q, err := s.repo.GetByID(strings.TrimSpace(questionID))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrQuestionNotFound
	}
	if err != nil {
		return err
	}

	if strings.TrimSpace(q.AuthorID) != strings.TrimSpace(userID) {
		return ErrForbiddenQuestionDelete
	}

	err = s.repo.Delete(strings.TrimSpace(questionID))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrQuestionNotFound
	}
	if err == nil {
		_ = s.publisher.PublishQuestionDeleted(context.Background(), q.ID)
	}
	return err
}
