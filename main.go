package main

import (
	"bytes"
	"encoding/json"
	"fmt"
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
					imagePath := fmt.Sprintf("%s/frame_%d.png", framePath, startFrame)

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

					fmt.Println("Status:", res.Status)
					fmt.Println("Response:", string(respBody))
					if strings.Contains(string(respBody), "error") {
						panic(string(respBody))
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
		
		time.Sleep(10800 * time.Second)
	}

}
