package repository

import (
	"easyoffer/question/internal/domain"
	"errors"

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
	err := r.db.Create(question).Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return ErrQuestionAlreadyExists
	}
	return err
}

func (r *questionRepository) Update(question *domain.Question) error {
	return r.db.Save(question).Error
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

func (r *questionRepository) GetMyQuestions(userID, status, category string, limit, offset int) ([]*domain.QuestionWithReview, int64, error) {
	rows := make([]domain.QuestionWithReview, 0)

	countQuery := r.db.Table("question_reviews AS qr").
		Joins("JOIN questions AS q ON q.id = qr.question_id").
		Where("qr.user_id = ?", userID)

	if status != "" {
		countQuery = countQuery.Where("qr.status = ?", status)
	}
	if category != "" {
		countQuery = countQuery.Where("q.category = ?", category)
	}

	total := int64(0)
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	dataQuery := r.db.Table("question_reviews AS qr").
		Select(`
            q.id,
            q.title,
            q.content,
            q.category,
            q.answer_format,
            q.language,
            q.starter_code,
            q.author_id,
            q.created_at,
            qr.status AS review_status,
            qr.reviewed_at
        `).
		Joins("JOIN questions AS q ON q.id = qr.question_id").
		Where("qr.user_id = ?", userID)

	if status != "" {
		dataQuery = dataQuery.Where("qr.status = ?", status)
	}
	if category != "" {
		dataQuery = dataQuery.Where("q.category = ?", category)
	}

	err := dataQuery.
		Order("qr.reviewed_at DESC").
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

		return nil
	})
}
