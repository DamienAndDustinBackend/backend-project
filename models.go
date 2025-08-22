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
}

type Tag struct {
	gorm.Model
	Name  string
	Files []File `gorm:"many2many:user_tags;"`
}
