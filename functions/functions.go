package functions

import (
	"bytes"
	"fmt"
	"github.com/eikarna/weago/enums"
	// "github.com/eikarna/weago/libs/ai"
	"github.com/goccy/go-json"
	"io/ioutil"
	"log"
	"net/http"
	"reflect"
	"strings"
	"time"
	"regexp"
)

func RemoveColonDigits(input string) string {
	parts := strings.Split(input, "@")
	localPart := strings.Split(parts[0], ":")[0]
	return localPart + "@" + parts[1]
}

func getKeysForValues(m interface{}) []string {
	var keys []string
	mapValue := reflect.ValueOf(m)

	if mapValue.Kind() != reflect.Map {
		return nil
	}

	for _, key := range mapValue.MapKeys() {
		// Collect keys if needed
		keys = append(keys, key.String())
	}

	return keys
}

func SaveSystem() {
	ticker := time.NewTicker(5000 * time.Millisecond)
	go func() {
		for {
			select {
			case <-enums.SaveStop:
				return
			case <-ticker.C:
				if enums.SaveSuccess {
					// Simple Synchronization
					enums.SaveSuccess = false
					for _, jid := range getKeysForValues(enums.ChatCache) {
						err := enums.SaveChatData(jid, enums.ChatCache[jid])
						if err != nil {
							log.Println("Error saving chat data:", err)
						}
					}
					for _, jid := range getKeysForValues(enums.ChatInfo) {
						err := enums.SaveChatInfo(enums.ChatDB, jid)
						if err != nil {
							log.Println("Error saving chat info data:", err)
						}
					}
					enums.SaveSuccess = true
				} else {
					// wait for a secs
				}
			}
		}
	}()
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

func NormalizeText(text string) string {
	// Remove extra spaces between words
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")

	// Replace multiple newlines with a single newline
	text = regexp.MustCompile(`\n+`).ReplaceAllString(text, "\n")

	// Trim any leading or trailing spaces and newlines
	text = strings.TrimSpace(text)

	return text
}
