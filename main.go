package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"
	openai "github.com/sashabaranov/go-openai"
	"github.com/xyproto/ollamaclient"
)

type chapter struct {
	Number     int
	Title      string
	QuickBrief string
	Content    string
}

type chapterSummary struct {
	Number  int
	Content string
}

func buildNewChapter(rawContent string) *chapter {
	lines := strings.Split(rawContent, "\n")

	chapterNumber := lines[0]
	chapterTitle := lines[1]
	summary := lines[2]
	content := strings.Join(lines[3:], "\n")

	i, err := strconv.Atoi(chapterNumber)
	if err != nil {
		// ... handle error
		panic(err)
	}

	newChapter := chapter{Number: i, Title: chapterTitle, QuickBrief: summary, Content: content}
	return &newChapter
}

func main() {
	start := time.Now()

	arg1 := flag.String("input", "TheArtOfThinkingClearly.txt", "input file")
	arg2 := flag.String("output", "TheArtOfThinkingClearly_Summary.txt", "output file")
	arg3 := flag.String("key", "", "OpenAI Key")

	// Parse command-line arguments
	flag.Parse()

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

	for i, chapter := range chapters {
		log.Printf("Adding worker %d. Title: %s\n", i, chapter.Title)
		go worker(&wg, i, chapter, &summaries[i], client)
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
	write_err := SaveToFile(directory, savedFile, content)
	if write_err != nil {
		println("Error:", err)
		return
	}

	duration := time.Since(start)
	log.Printf("Executed successfully in %s", duration)

}

func worker(wg *sync.WaitGroup, id int, chapter chapter, chapterSummary *chapterSummary, client *openai.Client) {
	defer wg.Done()

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
}

func SaveToFile(directory, filename string, content []byte) error {
	// Create the directory if it doesn't exist
	if err := os.MkdirAll(directory, 0755); err != nil {
		return err
	}

	// Join the directory path and filename
	filePath := filepath.Join(directory, filename)

	// Create or open the file for writing
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write the content to the file
	_, err = file.Write(content)
	if err != nil {
		return err
	}

	return nil
}

func splitIntoChapterList(content string) []chapter {
	re := regexp.MustCompile(`(?m)^\d+\n(?:[^\n]+\n)+`)
	matches := re.FindAllStringIndex(content, -1)
	var chapters []chapter
	for i, match := range matches {
		start := match[0]
		end := len(content)
		if i+1 < len(matches) {
			end = matches[i+1][0]
		}

		chapters = append(chapters, *buildNewChapter(content[start:end]))
	}

	return chapters
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
