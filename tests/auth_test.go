package tests

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"hmdp-go-test/models"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func sendCode(phone int64) (*resty.Response, error) {
	phoneString := strconv.FormatInt(phone, 10)
	url := SendCodeUrlPrefix + phoneString
	response, err := HttpClient.R().Post(url)
	return response, err
}

func queryCode(phone int64) (error, string) {
	ctx := context.Background()
	stringCmd := RedisClient.Get(ctx, "login:code:"+strconv.FormatInt(phone, 10))
	code := stringCmd.Val()
	if stringCmd.Err() != nil {
		return stringCmd.Err(), ""
	}
	if len(code) <= 0 {
		return fmt.Errorf("empty code"), ""
	}
	return nil, code
}

func sendAndQueryCode(phone int64) (error, string) {
	_, err := sendCode(phone)
	if err != nil {
		return err, ""
	}
	err, code := queryCode(phone)
	if err != nil {
		return err, ""
	}
	return nil, code
}

func getAuthWithPhoneAndCode(phone int64, code string) (error, string) {
	payload := map[string]interface{}{
		"phone": strconv.FormatInt(phone, 10),
		"code":  code,
	}
	resp, err := HttpClient.R().SetBody(payload).Post(LoginUrl)
	if err != nil {
		return err, ""
	}
	var result models.Result
	err = json.Unmarshal(resp.Body(), &result)
	if err != nil {
		return err, ""
	}
	return err, string(result.Data)
}

func getAuthWithPhone(phone int64) (error, string) {
	err, code := sendAndQueryCode(phone)
	if err != nil {
		return err, ""
	}
	err, auth := getAuthWithPhoneAndCode(phone, code)
	if err != nil {
		return err, ""
	}
	return err, auth
}

func TestLogin(t *testing.T) {
	phone := int64(18000000000)
	err, code := sendAndQueryCode(phone)
	assert.Nil(t, err)
	assert.NotNil(t, code)
	err, auth := getAuthWithPhoneAndCode(phone, code)
	assert.Nil(t, err)
	assert.NotNil(t, auth)
}

func TestSendCodeMulti(t *testing.T) {
	var basePhone = viper.GetInt64("test.user.base_phone")
	testCount := viper.GetInt("test.user.user_count")
	var wg sync.WaitGroup
	errChan := make(chan error, testCount) // 带缓冲的通道防止阻塞

	wg.Add(testCount)

	for i := 0; i < testCount; i++ {
		go func(phoneOffset int) {
			defer wg.Done()
			// 发送请求
			phone := basePhone + int64(phoneOffset)
			response, err := sendCode(phone)
			if err != nil {
				errChan <- fmt.Errorf("请求失败 phone=%v: %v", phone, err)
				return
			}

			// 校验状态码
			if response.StatusCode() != 200 {
				errChan <- fmt.Errorf("状态码错误 phone=%v: %v", phone, response.StatusCode())
				return
			}

			// 检查Redis
			err, _ = queryCode(phone)
			if err != nil {
				errChan <- fmt.Errorf("验证码未找到 phone=%v", phone)
			}
		}(i) // 显式传递循环变量
	}

	// 关闭错误通道的协程
	go func() {
		wg.Wait()
		close(errChan)
	}()

	// 实时处理错误
	for err := range errChan {
		t.Error(err)
	}
}

func TestGenerateAuths(t *testing.T) {
	basePhone := viper.GetInt64("test.user.base_phone")
	expectedAuthsCount := viper.GetInt("test.user.user_count")
	var wg sync.WaitGroup
	errChan := make(chan error, expectedAuthsCount)
	authChan := make(chan []string, expectedAuthsCount) // 用于传输手机号和auth的通道
	successCount := atomic.Int64{}
	// 打开文件，使用追加模式
	file, err := os.OpenFile(AuthsFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("无法打开文件: %v", err)
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)
	writer := csv.NewWriter(file)
	defer writer.Flush()
	// 启动生成goroutine
	wg.Add(expectedAuthsCount)
	for i := 0; i < expectedAuthsCount; {
		for j := 0; j < viper.GetInt("test.user.batch_size") && i < expectedAuthsCount; j++ {
			go func(phoneOffset int) {
				defer wg.Done()
				phone := basePhone + int64(phoneOffset)
				err, auth := getAuthWithPhone(phone)
				if err != nil {
					errChan <- fmt.Errorf("获取auth失败（手机号: %d）: %w", phone, err)
					return
				}
				successCount.Add(1)
				authChan <- []string{strconv.FormatInt(phone, 10), auth}
			}(i)
			i++
		}
		time.Sleep(1 * time.Second)
	}
	wg.Wait()
	// 处理authChan，保存到auths.csv文件
	size := len(authChan)
	for authData := range authChan {
		err := writer.Write(authData)
		if err != nil {
			fmt.Printf("write auth err, data: %v, error : %v\n", authData, err)
		}
		size--
		if size == 0 {
			close(authChan)
		}
	}
	assert.GreaterOrEqual(t, expectedAuthsCount, int(successCount.Load()))
}
