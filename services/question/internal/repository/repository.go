package repository

import (
    "easyoffer/question/internal/domain"

    "gorm.io/gorm"
)

type QuestionRepository interface {
    Create(question *domain.Question) error
}

type questionRepository struct {
    db *gorm.DB
}

func NewQuestionRepository(db *gorm.DB) QuestionRepository {
    return &questionRepository{db: db}
}

func (r *questionRepository) Create(question *domain.Question) error {
    return r.db.Create(question).Error
}