package main

import (
	"time"

	"gorm.io/gorm"
)

type File struct {
	ID          uint           `gorm:"primarykey;serializer:json"`
	CreatedAt   time.Time      `gorm:"serializer:json"`
	UpdatedAt   time.Time      `gorm:"serializer:json"`
	DeletedAt   gorm.DeletedAt `gorm:"index;serializer:json"`
	Name        string         `gorm:"serializer:json"`
	Description string         `gorm:"serializer:json"`
	FilePath    string         `gorm:"index;serializer:json"`
	Tags        []Tag          `gorm:"many2many:user_tags;serializer:json"`
	UserId      uint8          `gorm:"serializer:json"`
}

type Tag struct {
	ID        uint           `gorm:"primarykey;serializer:json"`
	CreatedAt time.Time      `gorm:"serializer:json"`
	UpdatedAt time.Time      `gorm:"serializer:json"`
	DeletedAt gorm.DeletedAt `gorm:"index;serializer:json"`
	Name      string         `gorm:"serializer:json"`
	Files     []File         `gorm:"many2many:user_tags;serializer:json"`
}

type User struct {
	ID        uint           `gorm:"primarykey;serializer:json"`
	CreatedAt time.Time      `gorm:"serializer:json"`
	UpdatedAt time.Time      `gorm:"serializer:json"`
	DeletedAt gorm.DeletedAt `gorm:"index;serializer:json"`
	Email     string         `gorm:"serializer:json"`
	Password  string         `gorm:"uniqueIndex;serializer:json"`

	Files []File
}
