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
	"strings"
	"time"

	"github.com/akadanar/eafio/fbapi"
	"github.com/akadanar/eafio/imageutils"
	"github.com/akadanar/eafio/utils"
)

type PostResp struct {
	ID     string `json:"id"`
	PostID string `json:"post_id"`
}

func main() {
	frame, errFrame := utils.LoadFrameLogs("framelogs.json")
	config, errConfig := utils.LoadConfig("config.json")
	delay := 180000 * time.Second
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
				maxFrame, errMaxF := utils.CountFilesInFolder(framePath)
				if errMaxF != nil {
					panic(errMaxF)
				}

				for f := startFrame; f <= maxFrame; f++ {
					caption := fmt.Sprintf("Season %d, Episode %d, Frame %d out of %d\n", season, eps, f, maxFrame)
					imagePath := fmt.Sprintf("%s/frame_%d.png", framePath, f)

					var buf bytes.Buffer
					writer := multipart.NewWriter(&buf)

					// Add file to form
					if err := utils.AddFilePart(writer, "source", imagePath); err != nil {
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

					croppedImageBytes, err := imageutils.CropRandomSquare(originalImageBytes)
					if err != nil {
						log.Fatalln("Crop error:", err)
					}

					mediaID, err := fbapi.UploadPhoto(croppedImageBytes, config.AccessToken)
					if err != nil {
						log.Fatalln("Upload error:", err)
					}

					// 2. Post a comment with the uploaded image
					err = fbapi.CommentWithPhoto(mediaID, "Random Crop:", post.PostID, config.AccessToken)
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
					time.Sleep(delay)
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
		maxFrame, errMaxF := utils.CountFilesInFolder(framePath)
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
		if err := utils.AddFilePart(writer, "source", imagePath); err != nil {
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

		croppedImageBytes, err := imageutils.CropRandomSquare(originalImageBytes)
		if err != nil {
			log.Fatalln("Crop error:", err)
		}

		mediaID, err := fbapi.UploadPhoto(croppedImageBytes, config.AccessToken)
		if err != nil {
			log.Fatalln("Upload error:", err)
		}

		// 2. Post a comment with the uploaded image
		err = fbapi.CommentWithPhoto(mediaID, "Random Crop", post.PostID, config.AccessToken)
		if err != nil {
			log.Fatalln("Comment error:", err)
		}

		time.Sleep(delay)
	}

}
