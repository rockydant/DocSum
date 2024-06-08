package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

const apiKey = "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"

func main() {
	// Read the document
	content, err := os.ReadFile("TheArtOfThinkingClearly.txt")
	if err != nil {
		log.Fatalf("Failed to read document: %v", err)
	}

	// Split the document into chapters using regex
	chapters := splitIntoChapters(string(content))

	log.Printf("Chapter count: %d", len(chapters))

	// Initialize the OpenAI client
	client := openai.NewClient(apiKey)

	// Summarize each chapter
	var summaries []string
	summaries = append(summaries, fmt.Sprintf("Total chapters: %d", len(chapters)))
	for i, chapter := range chapters {
		summary, err := summarizeChapter(client, chapter)
		if err != nil {
			log.Printf("Failed to summarize chapter %d: %v", i+1, err)
			continue
		}
		summaries = append(summaries, fmt.Sprintf("Summary of Chapter %d: %s", i+1, summary))
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
