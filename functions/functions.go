package functions

import (
	"bytes"
	"fmt"
	"github.com/goccy/go-json"
	"io/ioutil"
	"net/http"
	"strings"
)

func RemoveColonDigits(input string) string {
	parts := strings.Split(input, "@")
	localPart := strings.Split(parts[0], ":")[0]
	return localPart + "@" + parts[1]
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
	var result []map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return "", fmt.Errorf("error decoding response: %v", err)
	}
	parsedJson, ok := result[0]["response"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("Error parsing 'response' array")
	}
	responseText, ok := parsedJson["response"].(string)
	if !ok {
		return "", fmt.Errorf("invalid response text format")
	}

	return responseText, nil
}
