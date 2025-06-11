package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"math/rand/v2"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Config struct {
	AccessToken string `json:"access_token"`
	ID          string `json:"id"`
	MaxEps      int    `json:"max_eps"`
	MaxSeason   int    `json:"max_season"`
}

type FrameLogs struct {
	Frame    int  `json:"frame"`
	Eps      int  `json:"eps"`
	Season   int  `json:"season"`
	IsRandom bool `json:"is_random"`
}

type PostResp struct {
	ID     string `json:"id"`
	PostID string `json:"post_id"`
}

func loadFrameLogs(filename string) (*FrameLogs, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	logs := &FrameLogs{}
	err = decoder.Decode(logs)
	return logs, err
}

func loadConfig(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	config := &Config{}
	err = decoder.Decode(config)
	return config, err
}

func countFilesInFolder(path string) (int, error) {
	count := 0
	entries, err := os.ReadDir(path)
	if err != nil {
		return 0, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			count++
		}
	}

	return count, nil
}

func addFilePart(writer *multipart.Writer, fieldName, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	part, err := writer.CreateFormFile(fieldName, filepath.Base(filePath))
	if err != nil {
		return err
	}

	_, err = io.Copy(part, file)
	return err
}

func CropRandomSquare(inputBuffer []byte) ([]byte, error) {
	img, format, err := image.Decode(bytes.NewReader(inputBuffer))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %v", err)
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	maxCropSize := 680
	if width < height {
		if width < maxCropSize {
			maxCropSize = width
		}
	} else {
		if height < maxCropSize {
			maxCropSize = height
		}
	}

	minCropSize := 100
	if maxCropSize < minCropSize {
		return nil, fmt.Errorf("image too small to crop: min 200px required")
	}

	cropSize := rand.IntN(maxCropSize-minCropSize+1) + minCropSize
	maxX := width - cropSize
	maxY := height - cropSize

	randAxis := rand.IntN(2) // 0: X-priority, 1: Y-priority
	var startX, startY int

	if randAxis == 0 {
		if maxX > 0 {
			startX = rand.IntN(maxX + 1)
		}
		if maxY > 0 {
			startY = rand.IntN(maxY/2 + 1)
			if rand.IntN(2) == 0 {
				startY = maxY - startY
			}
		}
	} else {
		if maxY > 0 {
			startY = rand.IntN(maxY + 1)
		}
		if maxX > 0 {
			startX = rand.IntN(maxX/2 + 1)
			if rand.IntN(2) == 0 {
				startX = maxX - startX
			}
		}
	}

	rect := image.Rect(startX, startY, startX+cropSize, startY+cropSize)
	cropped := img.(interface {
		SubImage(r image.Rectangle) image.Image
	}).SubImage(rect)

	var outBuf bytes.Buffer
	switch format {
	case "png":
		err = png.Encode(&outBuf, cropped)
	case "jpeg", "jpg":
		err = jpeg.Encode(&outBuf, cropped, &jpeg.Options{Quality: 90})
	default:
		return nil, fmt.Errorf("unsupported image format: %s", format)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to encode cropped image: %v", err)
	}

	return outBuf.Bytes(), nil
}

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

func main() {
	frame, errFrame := loadFrameLogs("framelogs.json")
	config, errConfig := loadConfig("config.json")
	if errFrame != nil {
		panic(errFrame)
	}
	if errConfig != nil {
		panic(errConfig)
	}
	if !frame.IsRandom {
		for season := frame.Season; season <= config.MaxSeason; season++ {
			for eps := frame.Eps; eps <= config.MaxEps; eps++ {

				startFrame := 1
				if season == frame.Season && eps == frame.Eps {
					startFrame = frame.Frame + 1
				}
				framePath := fmt.Sprintf("./frame/S%d/eps%d", season, eps)
				maxFrame, errMaxF := countFilesInFolder(framePath)
				if errMaxF != nil {
					panic(errMaxF)
				}

				for f := startFrame; f <= maxFrame; f++ {
					caption := fmt.Sprintf("Season %d, Episode %d, Frame %d out of %d\n", season, eps, f, maxFrame)
					imagePath := fmt.Sprintf("%s/frame_%d.png", framePath, f)

					var buf bytes.Buffer
					writer := multipart.NewWriter(&buf)

					// Add file to form
					if err := addFilePart(writer, "source", imagePath); err != nil {
						log.Fatalln("Failed to add file:", err)
					}

					// Add field
					writer.WriteField("caption", caption)
					writer.WriteField("access_token", config.AccessToken)
					writer.Close()

					url := fmt.Sprintf("https://graph.facebook.com/v23.0/%s/photos", config.ID)
					req, err := http.NewRequest("POST", url, &buf)
					if err != nil {
						log.Fatalln(err)
					}
					req.Header.Set("Content-Type", writer.FormDataContentType())

					res, err := http.DefaultClient.Do(req)
					if err != nil {
						log.Fatalln(err)
					}
					defer res.Body.Close()

					respBody, err := io.ReadAll(res.Body)
					if err != nil {
						log.Fatalln(err)
					}

					if strings.Contains(string(respBody), "error") {
						panic(string(respBody))
					}

					fmt.Println("Status:", res.Status)
					fmt.Println("Response:", string(respBody))

					// Unmarshal respBody to struct
					var post PostResp
					err = json.Unmarshal(respBody, &post)
					if err != nil {
						log.Fatalf("Unmarshal JSON failed: %v", err)
					}
					// Random crop part
					// 1. Upload image from buffer
					originalImageBytes, err := os.ReadFile(imagePath)
					if err != nil {
						log.Fatalln("Failed to read original image:", err)
					}

					croppedImageBytes, err := CropRandomSquare(originalImageBytes)
					if err != nil {
						log.Fatalln("Crop error:", err)
					}

					mediaID, err := UploadPhoto(croppedImageBytes, config.AccessToken)
					if err != nil {
						log.Fatalln("Upload error:", err)
					}

					// 2. Post a comment with the uploaded image
					err = CommentWithPhoto(mediaID, "Random Crop:", post.PostID, config.AccessToken)
					if err != nil {
						log.Fatalln("Comment error:", err)
					}

					frame.Season = season
					frame.Eps = eps
					frame.Frame = f

					updatedData, err := json.MarshalIndent(frame, "", "  ")
					if err != nil {
						panic(err)
					}
					err = os.WriteFile("framelogs.json", updatedData, 0644)
					if err != nil {
						panic(err)
					}
					time.Sleep(10800 * time.Second)
				}
			}
		}

	}
	// Random mode
	frame.IsRandom = true
	updatedData, err := json.MarshalIndent(frame, "", "  ")
	if err != nil {
		panic(err)
	}
	err = os.WriteFile("framelogs.json", updatedData, 0644)
	if err != nil {
		panic(err)
	}

	for frame.IsRandom {
		eps := rand.IntN(config.MaxEps) + 1
		season := rand.IntN(config.MaxSeason) + 1
		framePath := fmt.Sprintf("./frame/S%d/eps%d", season, eps)
		maxFrame, errMaxF := countFilesInFolder(framePath)
		if errMaxF != nil {
			panic(errMaxF)
		}

		if maxFrame <= 0 {
			log.Printf("No frames found in %s. Skipping...\n", framePath)
			continue
		}

		frameRand := rand.IntN(maxFrame) + 1
		caption := fmt.Sprintf("[Random]\nSeason %d, Episode %d, Frame %d out of %d\n", season, eps, frameRand, maxFrame)
		imagePath := fmt.Sprintf("%s/frame_%d.png", framePath, frameRand)

		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)

		// Add file to form
		if err := addFilePart(writer, "source", imagePath); err != nil {
			log.Fatalln("Failed to add file:", err)
		}
		writer.WriteField("caption", caption)
		writer.WriteField("access_token", config.AccessToken)
		writer.Close()

		url := fmt.Sprintf("https://graph.facebook.com/v23.0/%s/photos", config.ID)
		req, err := http.NewRequest("POST", url, &buf)
		if err != nil {
			log.Fatalln(err)
		}
		req.Header.Set("Content-Type", writer.FormDataContentType())

		res, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Fatalln(err)
		}
		defer res.Body.Close()

		respBody, err := io.ReadAll(res.Body)
		if err != nil {
			log.Fatalln(err)
		}

		fmt.Println("Status:", res.Status)
		fmt.Println("Response:", string(respBody))

		if strings.Contains(string(respBody), "error") {
			panic(string(respBody))
		}
		// Unmarshal respBody to struct
		var post PostResp
		err = json.Unmarshal(respBody, &post)
		if err != nil {
			log.Fatalf("Unmarshal JSON failed: %v", err)
		}
		//Random Crop Comment Part
		// 1. Upload image from buffer
		originalImageBytes, err := os.ReadFile(imagePath)
		if err != nil {
			log.Fatalln("Failed to read original image:", err)
		}

		croppedImageBytes, err := CropRandomSquare(originalImageBytes)
		if err != nil {
			log.Fatalln("Crop error:", err)
		}

		mediaID, err := UploadPhoto(croppedImageBytes, config.AccessToken)
		if err != nil {
			log.Fatalln("Upload error:", err)
		}

		// 2. Post a comment with the uploaded image
		err = CommentWithPhoto(mediaID, "Random Crop", post.PostID, config.AccessToken)
		if err != nil {
			log.Fatalln("Comment error:", err)
		}

		time.Sleep(10800 * time.Second)
	}

}
