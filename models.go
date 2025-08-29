package main

import (
	"gorm.io/gorm"
)

type File struct {
	gorm.Model
	Name        string
	Description string
	FilePath    string
	Tags        []Tag `gorm:"many2many:user_tags;"`
	UserId      uint8
}

type Tag struct {
	gorm.Model
	Name  string
	Files []File `gorm:"many2many:user_tags;"`
}

type User struct {
	gorm.Model
	Email    string `gorm:"uniqueIndex"`
	Password string

	Files []File
}
