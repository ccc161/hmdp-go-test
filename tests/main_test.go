package tests

import (
	"database/sql"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/go-resty/resty/v2"
	"github.com/spf13/viper"
	"hmdp-go-test/utils"
	"log"
	"os"
	"path/filepath"
	"testing"
)

var DBClient *sql.DB
var RedisClient *redis.Client
var HttpClient *resty.Client
var SendCodeUrlPrefix string
var LoginUrl string
var AddSeckillVoucherUrl string
var PurchaseSeckillVoucherUrlPrefix string
var AuthsFilePath string

func TestMain(m *testing.M) {
	// 1. 初始化配置
	setupConfig()

	// 2. 初始化全局资源（数据库等）
	DBClient, RedisClient = setupDBResources()
	HttpClient = setupHttpResource()
	setupUrls()
	setupFilePaths()
	defer teardownResources(DBClient, RedisClient)

	// 3. 运行测试套件
	code := m.Run()

	// 4. 退出
	os.Exit(code)
}

func setupUrls() {
	SendCodeUrlPrefix = viper.GetString("api.base_url") + viper.GetString("api.prefix.auth_code") + "?phone="
	LoginUrl = viper.GetString("api.base_url") + viper.GetString("api.prefix.login")
	AddSeckillVoucherUrl = viper.GetString("api.base_url") + viper.GetString("api.prefix.voucher")
	PurchaseSeckillVoucherUrlPrefix = viper.GetString("api.base_url") + viper.GetString("api.prefix.purchase")
}

func setupFilePaths() {
	dir, err := os.Getwd()
	if err != nil {
		panic("Failed to get working directory: " + err.Error())
	}
	AuthsFilePath = filepath.Join(dir, viper.GetString("test.user.auth_file_name"))
}

func setupConfig() {
	viper.SetConfigFile("../configs/config.yaml")
	if err := viper.ReadInConfig(); err != nil {
		panic("Failed to read config: " + err.Error())
	}
}

func setupDBResources() (*sql.DB, *redis.Client) {
	// 初始化数据库连接等
	mysqlClient, err := utils.InitMySQL()
	if err != nil {
		log.Fatal(err)
		return nil, nil
	}
	return mysqlClient, utils.InitRedis()
}

func setupHttpResource() *resty.Client {
	return utils.InitHttpClient()
}

func teardownResources(db *sql.DB, redis *redis.Client) {
	// 关闭连接等清理工作
	err := db.Close()
	if err != nil {
		log.Fatal(err)
		return
	}
	err = redis.Close()
	if err != nil {
		log.Fatal(err)
		return
	}
}

func TestWorkPath(t *testing.T) {
	fmt.Println(os.Getwd())
}
