package utils

import (
	"encoding/json"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
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

func LoadFrameLogs(filename string) (*FrameLogs, error) {
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

func LoadConfig(filename string) (*Config, error) {
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

func CountFilesInFolder(path string) (int, error) {
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

func AddFilePart(writer *multipart.Writer, fieldName, filePath string) error {
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
