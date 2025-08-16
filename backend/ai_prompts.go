package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type GeminiRequest struct {
	Contents []*Content `json:"contents"`
}
type Content struct {
	Parts []*Part `json:"parts"`
}
type Part struct {
	Text string `json:"text"`
}
type GeminiResponse struct {
	Candidates []*Candidate `json:"candidates"`
}
type Candidate struct {
	Content *Content `json:"content"`
}

func callGemini(apiKey, prompt string) (string, error) {
	url := "https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash-latest:generateContent?key=" + apiKey
	reqBody := GeminiRequest{Contents: []*Content{{Parts: []*Part{{Text: prompt}}}}}
	jsonData, _ := json.Marshal(reqBody)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))

	// THIS IS THE CORRECTED LINE. The parentheses on Header() are removed.
	req.Header.Set("Content-Type", "application/json")

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API request failed with status: %s", resp.Status)
	}
	var apiResponse GeminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return "", err
	}
	if len(apiResponse.Candidates) > 0 && apiResponse.Candidates[0].Content != nil && len(apiResponse.Candidates[0].Content.Parts) > 0 {
		return apiResponse.Candidates[0].Content.Parts[0].Text, nil
	}
	return "", fmt.Errorf("unexpected API response format")
}

func analyzeCode(apiKey, code, lang string) (string, error) {
	prompt := fmt.Sprintf("You are an expert code reviewer. Analyze the following %s code snippet. Provide a comprehensive review covering these three areas, using markdown headings for each:\n\n### Correctness\n- Point out any potential bugs, logical errors, or unhandled edge cases.\n\n### Suggestions\n- Suggest improvements for readability, performance, or idiomatic style.\n\n### Solidity\n- Give a rating from 1-10 on how robust and production-ready the code is, and briefly justify your rating.\n\n```%s\n%s\n```", lang, lang, code)
	return callGemini(apiKey, prompt)
}

func refactorCode(apiKey, code, lang string) (string, error) {
	prompt := fmt.Sprintf("You are an expert software engineer. Refactor the following %s code snippet to improve its quality, readability, and performance. Provide ONLY the refactored code inside a single markdown code block, with no additional explanation before or after the code block.\n\n```%s\n%s\n```", lang, lang, code)
	return callGemini(apiKey, prompt)
}

func addCommentsToCode(apiKey, code, lang string) (string, error) {
	prompt := fmt.Sprintf("You are an expert software engineer. Add clear, concise, and helpful comments to the following %s code snippet. Explain the 'why' behind the code, not just the 'what'. Provide ONLY the commented code inside a single markdown code block, with no additional explanation before or after the code block.\n\n```%s\n%s\n```", lang, lang, code)
	return callGemini(apiKey, prompt)
}
