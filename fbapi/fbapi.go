package fbapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
)

// UploadPhoto uploads an image from a buffer to Facebook and returns the media ID
func UploadPhoto(buffer []byte, accessToken string) (string, error) {
	var b bytes.Buffer
	writer := multipart.NewWriter(&b)

	// Add file from buffer
	part, err := writer.CreateFormFile("source", "image.png")
	if err != nil {
		return "", err
	}
	_, err = part.Write(buffer)
	if err != nil {
		return "", err
	}

	// Add access_token to the form
	_ = writer.WriteField("access_token", accessToken)

	err = writer.Close()
	if err != nil {
		return "", err
	}

	// Upload to Facebook (unpublished photo)
	url := fmt.Sprintf("https://graph.facebook.com/v23.0/me/photos?published=false")
	req, err := http.NewRequest("POST", url, &b)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	fmt.Println("Upload Photo Response:", string(respBody))

	// Parse media ID from response
	var respJSON map[string]interface{}
	err = json.Unmarshal(respBody, &respJSON)
	if err != nil {
		return "", err
	}

	mediaID, ok := respJSON["id"].(string)
	if !ok {
		return "", fmt.Errorf("media ID not found in response")
	}
	if errInfo, exists := respJSON["error"]; exists {
		errMap := errInfo.(map[string]interface{})
		return "", fmt.Errorf("API Error: %v", errMap["message"])
	}

	return mediaID, nil
}

// CommentWithPhoto posts a comment with an image to a specific post
func CommentWithPhoto(mediaID, message, postID, accessToken string) error {
	url := fmt.Sprintf("https://graph.facebook.com/v23.0/%s/comments", postID)

	data := map[string]string{
		"access_token":  accessToken,
		"message":       message,
		"attachment_id": mediaID,
	}

	jsonData, _ := json.Marshal(data)

	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if strings.Contains(string(respBody), "error") {
		panic(string(respBody))
	}
	fmt.Println("Comment Response:", string(respBody))

	return nil
}
