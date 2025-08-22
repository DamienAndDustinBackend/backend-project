package main

import (
	"context"
	"fmt"
	"os"

	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	environment := os.Getenv("ENVIRONMENT")
	if environment == "" {
		environment = "PRODUCTION"
	}
	fmt.Printf("Running in %s\n", environment)

	var db *gorm.DB
	var err error
	if environment == "TEST" {
		fmt.Println("Using SQLite.")
		db, err = gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
		if err != nil {
			panic("failed to connect database")
		}
	} else {
		fmt.Println("Using MySQL.")
		dsn := os.Getenv("DSN")
		if dsn == "" {
			panic("DSN environment variable not set.")
		}
		db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
		if err != nil {
			panic("failed to connect database")
		}
	}

	ctx := context.Background()

	// Migrate the schema
	err = db.AutoMigrate(&File{}, &Tag{}, &User{})
	if err != nil {
		panic("failed to run database migrations")
	}

	tag := Tag{Name: "test-tag"}
	err = gorm.G[Tag](db).Create(
		ctx,
		&tag,
	)

	err = gorm.G[File](db).Create(
		ctx,
		&File{Name: "name", Description: "description", FilePath: "/tmp/test.txt", Tags: []Tag{tag}},
	)

	fmt.Println("Hello World")
}
