# DocSum

This project is a simple golang app that summarizes text by chunking the input file into paragraphs and then summarizing each paragraph using OpenAI API or Ollama API.

## Installation
go mod download

## Usage
go run main.go -input <input_file> -output <output_file> -key <api_key>
(or set key in .env file)

# Summary