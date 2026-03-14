package repository

import (
    "gorm.io/gorm"
    "easyoffer/auth/internal/domain"
)

type UserRepository interface{
	Create(user *domain.User) error
	GetByEmail(email string) (*domain.User, error)
}

type userRepository struct{
	db *gorm.DB	
}

func NewUserRepository(db *gorm.DB) UserRepository{
	return &userRepository{db: db}
}

func (r *userRepository) Create(user *domain.User) error{
	return r.db.Create(user).Error
}

func (r *userRepository) GetByEmail(email string) (*domain.User, error){
	var user domain.User
	err := r.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}