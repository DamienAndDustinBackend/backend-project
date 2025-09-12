package main

import (
	"time"

	"gorm.io/gorm"
)

type File struct {
	gorm.Model
	ID          uint           `gorm:"primarykey"`
	CreatedAt   time.Time      ``
	UpdatedAt   time.Time      ``
	DeletedAt   gorm.DeletedAt `gorm:"index"`
	Name        string         ``
	Description string         ``
	FilePath    string         `gorm:"index"`
	Tags        []Tag          `gorm:"many2many:user_tags"`
	UserId      uint8          ``
}

type Tag struct {
	ID        uint           `gorm:"primarykey"`
	CreatedAt time.Time      ``
	UpdatedAt time.Time      ``
	DeletedAt gorm.DeletedAt `gorm:"index"`
	Name      string         ``

	Files []File `gorm:"many2many:user_tags"`
}

type User struct {
	ID        uint           `gorm:"primarykey"`
	CreatedAt time.Time      ``
	UpdatedAt time.Time      ``
	DeletedAt gorm.DeletedAt `gorm:"index"`
	Email     string         `gorm:"uniqueIndex" json:"email"`
	Password  string         ``

	Files []File
}
