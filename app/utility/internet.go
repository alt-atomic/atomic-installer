package utility

import (
	"net/http"
	"time"
)

// CheckInternet проверяет наличие интернет-соединения
func CheckInternet() bool {
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	resp, err := client.Get("https://ya.ru")
	if err != nil {
		return false
	}
	resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}
