package domain

import "time"

type Question struct {
	ID        string    `gorm:"type:uuid;primaryKey" json:"id"`
	Title     string    `gorm:"type:varchar(255)" json:"title"`
	Content   string    `gorm:"type:text;not null" json:"content"`
	AuthorID  string    `gorm:"type:varchar(255); not null;index" json:"author_id"`
	CreatedAt time.Time `gorm:"type:timestamp" json:"created_at"`
}
