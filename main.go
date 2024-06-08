package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	openai "github.com/sashabaranov/go-openai"
	"github.com/xyproto/ollamaclient"
)

type chapter struct {
	Number     int
	Title      string
	QuickBrief string
	Content    string
}

func newChapter(number int, title string, quickBrief string, content string) *chapter {
	newChapter := chapter{Number: number, Title: title, QuickBrief: quickBrief, Content: content}
	return &newChapter
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

const apiKey = "OPEN_AI_KEY"

func main() {
	// Read the document
	content, err := os.ReadFile("TheArtOfThinkingClearly.txt")
	if err != nil {
		log.Fatalf("Failed to read document: %v", err)
	}

	// Split the document into chapters using regex
	//chapters := splitIntoChapters(string(content))
	chapters := splitIntoChapterList(string(content))

	log.Printf("------------ Chapter count: %d ------------\n", len(*chapters))

	// Initialize the OpenAI client
	//client := openai.NewClient(apiKey)

	// Summarize each chapter
	var summaries []string
	summaries = append(summaries, fmt.Sprintf("------------ Total chapters: %d ------------\n", len(*chapters)))
	for i, chapter := range *chapters {
		// summary, err := summarizeChapter(client, chapter)
		summary, err := summarizeChapter_ollama(&chapter)
		if err != nil {
			log.Printf("Failed to summarize chapter %d: %v", i+1, err)
			continue
		}
		summaries = append(summaries, fmt.Sprintf("--- Summary of Chapter %d:\n%s\n", i+1, summary))
	}

	// Combine summaries
	finalSummary := strings.Join(summaries, "\n\n")
	fmt.Println("Final Summary:\n", finalSummary)
	write_err := os.WriteFile("TheArtOfThinkingClearly_Summary.txt", []byte(finalSummary), 0644)
	if write_err != nil {
		fmt.Println("Error writing to file:", err)
		return
	}

	fmt.Println("File written successfully")
}

func splitIntoChapterList(content string) *[]chapter {
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

	return &chapters
}

func splitIntoChapters(content string) []string {
	re := regexp.MustCompile(`(?m)^\d+\n(?:[^\n]+\n)+`)
	// Find all matches and split the content accordingly
	matches := re.FindAllStringIndex(content, -1)

	var chapters []string
	for i, match := range matches {
		start := match[0]
		end := len(content)
		if i+1 < len(matches) {
			end = matches[i+1][0]
		}
		chapters = append(chapters, content[start:end])
	}

	return chapters
}

func summarizeChapter_ollama(chapter *chapter) (string, error) {
	oc := ollamaclient.NewWithModel("mistral:latest")

	oc.Verbose = false

	if err := oc.PullIfNeeded(); err != nil {
		fmt.Println("Error:", err)
		return "Error", err
	}

	prompt := fmt.Sprintf("Summarize this chapter with title %s and brief %s and content %s", chapter.Title, chapter.QuickBrief, chapter.Content)
	output, err := oc.GetOutput(prompt)
	if err != nil {
		fmt.Println("Error:", err)
		return "Error", err
	}
	fmt.Printf("\n---------------------\n%s\n", strings.TrimSpace(output))

	return output, nil
}

func summarizeChapter(client *openai.Client, chapter string) (string, error) {
	req := openai.CompletionRequest{
		Model:     "text-davinci-003",
		Prompt:    fmt.Sprintf("Summarize the following chapter:\n\n%s", chapter),
		MaxTokens: 200, // Adjust this based on your needs
	}

	resp, err := client.CreateCompletion(context.Background(), req)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(resp.Choices[0].Text), nil
}
