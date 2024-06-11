package main

import (
	"regexp"
	"strconv"
	"strings"
)

type chapter struct {
	Number     int
	Title      string
	QuickBrief string
	Content    string
}

func buildNewChapter(rawContent string) chapter {
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
	return newChapter
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

		chapters = append(chapters, buildNewChapter(content[start:end]))
	}

	return chapters
}
