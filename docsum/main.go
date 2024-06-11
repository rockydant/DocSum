package main

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	r := gin.Default()
	r.GET("/books/:id", func(c *gin.Context) {
		id := c.Param("id")
		db, err := connectToDatabase()
		if err != nil {
			fmt.Println("Error:", err)
			return
		}

		doc, err := getDocumentByID(db, id)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}

		c.String(200, string(doc.Content)+string(doc.Summary))
	})
	// Start the server on port 8080
	r.Run(":8080")
}

func connectToDatabase() (*gorm.DB, error) {
	// Replace the following variables with your actual database connection details
	username := "docsum"
	password := "12345678"
	hostname := "localhost"
	dbname := "doc_sum"

	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", username, password, hostname, dbname)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return db, nil
}

type Book struct {
	ID           uint      `gorm:"primaryKey"`
	Title        string    `gorm:"type:varchar(255)"`
	Content      []byte    `gorm:"type:mediumblob"`
	Summary      []byte    `gorm:"type:mediumblob"`
	Publish_Time time.Time `gorm:"type:datetime"`
}

func getDocumentByID(db *gorm.DB, id string) (*Book, error) {
	var doc Book
	if err := db.First(&doc, id).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve document: %w", err)
	}
	return &doc, nil
}
