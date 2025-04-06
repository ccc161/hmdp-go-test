package utils

import (
	"github.com/go-resty/resty/v2"
	"github.com/spf13/viper"
	"time"
)

func InitHttpClient() *resty.Client {
	httpClient := resty.New().SetBaseURL(viper.GetString("api.base_url")).
		SetHeaders(map[string]string{
			"User-Agent":   "Apifox/1.0.0 (https://apifox.com)",
			"Accept":       "*/*",
			"Connection":   "keep-alive",
			"Content-Type": "application/json",
		}).
		SetTimeout(5 * time.Second)
	return httpClient
}
