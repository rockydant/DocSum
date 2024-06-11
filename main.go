package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"
	openai "github.com/sashabaranov/go-openai"
	"github.com/xyproto/ollamaclient"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type Book struct {
	ID           uint      `gorm:"primaryKey"`
	Title        string    `gorm:"type:varchar(255)"`
	Content      []byte    `gorm:"type:mediumblob"`
	Summary      []byte    `gorm:"type:mediumblob"`
	Publish_Time time.Time `gorm:"type:datetime"`
}

func main() {
	start := time.Now()

	// Define command-line arguments
	arg0 := flag.Int("max", 3, "max threads allowed")
	arg1 := flag.String("input", "TheArtOfThinkingClearly.txt", "input file")
	arg2 := flag.String("output", "TheArtOfThinkingClearly_Summary.txt", "output file")
	arg3 := flag.String("key", "", "OpenAI Key")

	// Parse command-line arguments
	flag.Parse()

	max_concurrency := *arg0
	fileName := *arg1
	savedFile := *arg2
	inputKey := *arg3

	env := os.Getenv("DOCSUM_ENV")
	if "" == env {
		env = "development"
	}

	godotenv.Load(".env." + env)
	godotenv.Load() // The Original .env

	secretKey := ""
	if "" == inputKey {
		secretKey = os.Getenv("SECRET_KEY")
	} else {
		secretKey = inputKey
	}

	if "" == secretKey {
		return
	}

	directory := "bin"

	if fileName == "" {
		log.Printf("No Key Found")
		return
	}
	//filename := "TheArtOfThinkingClearly_Summary.txt"

	// Read the document
	content, err := os.ReadFile(fileName)
	if err != nil {
		log.Fatalf("Failed to read document: %v", err)
	}

	// Split the document into chapters using regex
	chapters := splitIntoChapterList(string(content))

	log.Printf("------------ Chapter count: %d ------------\n", len(chapters))

	// Initialize the OpenAI client
	client := openai.NewClient(secretKey)

	// Summarize each chapter
	summaries := make([]chapterSummary, len(chapters))
	var wg sync.WaitGroup
	wg.Add(len(chapters))

	semaphore := make(chan struct{}, max_concurrency)

	for i, chapter := range chapters {
		log.Printf("Adding worker %d. Title: %s\n", i, chapter.Title)
		go worker(&wg, i, chapter, &summaries[i], semaphore, client)
	}

	log.Printf("Waiting for %d workers to finish\n", len(chapters))
	wg.Wait()
	log.Printf("All Workers Completed")

	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].Number < summaries[j].Number
	})

	// Combine summaries
	var tempSummary []string
	for _, summary := range summaries {
		tempSummary = append(tempSummary, summary.Content)
	}

	finalSummary := strings.Join(tempSummary, "\n\n")
	log.Printf("Final Summary:\n", finalSummary)
	outputPath, write_err := SaveToFile(directory, savedFile, []byte(finalSummary))
	if write_err != nil {
		println("Error:", err)
		return
	}

	fmt.Println(max_concurrency)
	// save files to database
	db, err := connectToDatabase()
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	if err := createDocument(db, fileName, fileName, outputPath); err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println("New record created successfully!")

	duration := time.Since(start)
	log.Printf("Executed successfully in %s", duration)

}

func worker(wg *sync.WaitGroup, id int, chapter chapter, chapterSummary *chapterSummary, semaphore chan struct{}, client *openai.Client) {
	defer wg.Done()

	// Acquire semaphore
	semaphore <- struct{}{}

	fmt.Printf("Worker %v (%s): Started\n", id, chapter.Title)
	//summary, err := summarizeChapter_ollama(chapter)
	summary, err := summarizeChapter_openai(client, chapter)
	if err != nil {
		log.Printf("Failed to summarize chapter %d: %v", chapter.Number, err)
	}

	chapterSummary.Number = chapter.Number //chapter.Number
	chapterSummary.Content = summary       //summary
	time.Sleep(time.Second)
	fmt.Printf("Worker %v (%s): Finished\n", id, chapter.Title)

	// Release semaphore
	<-semaphore
}

func SaveToFile(directory, filename string, content []byte) (string, error) {
	// Create the directory if it doesn't exist
	if err := os.MkdirAll(directory, 0755); err != nil {
		return "", err
	}

	// Join the directory path and filename
	filePath := filepath.Join(directory, filename)

	// Create or open the file for writing
	file, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Write the content to the file
	_, err = file.Write(content)
	if err != nil {
		return "", err
	}

	return filePath, nil
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

func createDocument(db *gorm.DB, title string, inputFilePath string, outputFilePath string) error {
	// Read file content
	inputFileContent, err := os.ReadFile(inputFilePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	outputFileContent, err := os.ReadFile(outputFilePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Create a new document record
	doc := Book{
		Title:        title,
		Content:      inputFileContent,
		Summary:      outputFileContent,
		Publish_Time: time.Now(),
	}

	// Insert the record into the database
	if err := db.Create(&doc).Error; err != nil {
		return fmt.Errorf("failed to create record: %w", err)
	}

	return nil
}

func summarizeChapter_ollama(chapter chapter) (string, error) {
	oc := ollamaclient.NewWithModel("mistral:latest")

	oc.Verbose = false

	if err := oc.PullIfNeeded(); err != nil {
		log.Printf("Error:", err)
		return "Error", err
	}

	prompt := fmt.Sprintf("Summarize this chapter with title %s and brief %s and content %s", chapter.Title, chapter.QuickBrief, chapter.Content)
	output, err := oc.GetOutput(prompt)
	if err != nil {
		log.Printf("Error:", err)
		return "Error", err
	}
	//fmt.Printf("\n------ Summary of Chapter %d:\n%s", chapter.Number, strings.TrimSpace(output))

	response := fmt.Sprintf("\n------ Summary of Chapter %d:\n\n%s", chapter.Number, strings.TrimSpace(output))

	return response, nil
}

func summarizeChapter_openai(client *openai.Client, chapter chapter) (string, error) {
	req := openai.CompletionRequest{
		Model:     "gpt-3.5-turbo-instruct",
		Prompt:    fmt.Sprintf("Summarize this chapter with title %s and brief %s and content %s", chapter.Title, chapter.QuickBrief, chapter.Content),
		MaxTokens: 200, // Adjust this based on your needs
	}

	resp, err := client.CreateCompletion(context.Background(), req)
	if err != nil {
		return "", err
	}

	response := fmt.Sprintf("\n------ Summary of Chapter %d:\n\n%s", chapter.Number, strings.TrimSpace(resp.Choices[0].Text))

	return response, nil
}
