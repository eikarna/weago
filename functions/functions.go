package functions

import (
	"bytes"
	"fmt"
	"github.com/eikarna/weago/enums"
	"github.com/goccy/go-json"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"regexp"
	"strings"
)

func RemoveColonDigits(input string) string {
	parts := strings.Split(input, "@")
	localPart := strings.Split(parts[0], ":")[0]
	return localPart + "@" + parts[1]
}

func NormalizeText(text string) string {
	// Remove extra spaces between words
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")

	// Replace multiple newlines with a single newline
	text = regexp.MustCompile(`\n+`).ReplaceAllString(text, "\n")

	// Trim any leading or trailing spaces and newlines
	text = strings.TrimSpace(text)

	return text
}

func NormalizeNewlines(text string) string {
	// Replace multiple newlines with a single newline
	re := regexp.MustCompile(`\n+`)
	normalizedText := re.ReplaceAllString(text, "\n")

	// Optionally, you can trim leading and trailing newlines
	normalizedText = strings.Trim(normalizedText, "\n")

	return normalizedText
}

func PpJSON(data interface{}) string {
	a, _ := json.MarshalIndent(data, "", "  ")
	return string(a)
}

func Get(url string) (string, error) {
	// Send the GET request
	resp, err := http.Get(url)
	if err != nil {
		// log.Fatalf("Error making request: %v", err)
		return "", err
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		// log.Fatalf("Error reading response body: %v", err)
		return "", err
	}

	// Print the response body
	return string(body), nil
}

func Post(url string, body []byte) (string, error) {
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("request failed with status code: %d", resp.StatusCode)
	}

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return "", fmt.Errorf("error decoding response: %v", err)
	}
	fmt.Printf("Decoded Result:\n%#v\n\n", result)

	// Extract the nested response text
	candidates, ok := result["candidates"].([]interface{})
	if !ok || len(candidates) == 0 {
		return "", fmt.Errorf("no candidates found in response")
	}

	firstCandidate, ok := candidates[0].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("unexpected structure in candidates")
	}

	content, ok := firstCandidate["content"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("content not found in first candidate")
	}

	parts, ok := content["parts"].([]interface{})
	if !ok || len(parts) == 0 {
		return "", fmt.Errorf("no parts found in content")
	}

	firstPart, ok := parts[0].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("unexpected structure in parts")
	}

	responseText, ok := firstPart["text"].(string)
	if !ok {
		return "", fmt.Errorf("text not found in first part")
	}

	return responseText, nil
}

func UploadImage(filePath, mimeType string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	// Get the file size
	fileInfo, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("failed to get file info: %v", err)
	}
	numBytes := fileInfo.Size()

	// Prepare the request URL
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/upload/v1beta/files?key=%s", enums.BotInfo.ApiKey)

	// Prepare the headers
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("X-Goog-Upload-Command", "start, upload, finalize")
	req.Header.Set("X-Goog-Upload-Header-Content-Length", fmt.Sprintf("%d", numBytes))
	req.Header.Set("X-Goog-Upload-Header-Content-Type", mimeType)
	req.Header.Set("Content-Type", "application/json")

	// Create a buffer and a multipart writer
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add the file to the request
	part, err := writer.CreateFormFile("file", filePath)
	if err != nil {
		return "", fmt.Errorf("failed to create form file: %v", err)
	}
	_, err = io.Copy(part, file)
	if err != nil {
		return "", fmt.Errorf("failed to copy file to form: %v", err)
	}
	writer.Close()

	// Attach the body to the request
	req.Body = io.NopCloser(body)

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Read the response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %v", err)
	}
	// Extract the file URI from the response
	var responseMap map[string]interface{}
	if err := json.Unmarshal(respBody, &responseMap); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %v", err)
	}
	fileURI, ok := responseMap["file"].(map[string]interface{})["uri"].(string)
	if !ok {
		return "", fmt.Errorf("failed to extract file URI from response")
	}

	return fileURI, nil
}

func SaveBufferToTempFile(bufferData []byte, fileExtension string) (string, error) {
	// Create a temporary file with the specified extension
	tempFile, err := ioutil.TempFile("", fmt.Sprintf("whatsmeow_buffer_*%s", fileExtension))
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %v", err)
	}
	defer tempFile.Close()

	// Write the buffer data to the file
	_, err = tempFile.Write(bufferData)
	if err != nil {
		return "", fmt.Errorf("failed to write buffer to temp file: %v", err)
	}

	// Return the temp file path
	return tempFile.Name(), nil
}
