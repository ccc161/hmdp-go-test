package tests

import (
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"hmdp-go-test/models"
	"hmdp-go-test/utils"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"
)

func queryUserIds(t *testing.T, phoneToAuth map[string]string) []models.User {
	phones := make([]interface{}, 0, len(phoneToAuth))
	for phone := range phoneToAuth {
		phones = append(phones, phone)
	}
	placeholder := ""
	for i := range len(phones) {
		if i > 0 {
			placeholder += ", "
		}
		placeholder += "?"
	}
	query := fmt.Sprintf("select id, phone from tb_user where phone in (%s)", placeholder)
	result, err := DBClient.Query(query, phones...)
	defer func(result *sql.Rows) {
		err := result.Close()
		if err != nil {
			t.Fatalf("failed to close rows: %v", err)
		}
	}(result)
	if err != nil {
		t.Fatalf("query users error: %s", err.Error())
	}

	users := make([]models.User, 0, len(phoneToAuth))
	for result.Next() {
		var user models.User
		if err := result.Scan(&user.ID, &user.Phone); err != nil {
			t.Fatalf("failed to scan row: %s", err.Error())
		}
		user.Auth = phoneToAuth[user.Phone]
		users = append(users, user)
	}
	if err := result.Err(); err != nil {
		t.Fatalf("query users error: %s", err.Error())
	}
	return users
}

func getPhonesAndAuths(t *testing.T) map[string]string {
	t.Helper()
	file, err := os.Open(AuthsFilePath)
	if err != nil {
		t.Fatalf("Failed to open AuthsFile: %v", err)
		return nil
	}
	reader := csv.NewReader(file)
	phoneToAuth := make(map[string]string)
	for {
		record, err := reader.Read()
		if err != nil {
			// 如果到达文件末尾，结束循环
			if err.Error() == "EOF" {
				break
			}
			// 其他错误，打印错误信息
			t.Fatalf("Failed to parse AuthsFile: %v", err)
		}
		// 确保每行至少有两列
		if len(record) < 2 {
			t.Fatalf("Failed to parse AuthsFile, expected format : {phone},{authorization}")
		}
		phoneToAuth[record[0]] = record[1]
	}
	assert.NotEmpty(t, phoneToAuth)
	return phoneToAuth
}
func TestAddSeckillVoucher(t *testing.T) {
	phoneToAuth := getPhonesAndAuths(t)
	var authorization string
	for phone := range phoneToAuth {
		authorization = phoneToAuth[phone]
		break
	}
	payload := map[string]interface{}{
		"shopId":      1,
		"title":       "100元代金券",
		"subTitle":    "周一至周五均可使用",
		"rules":       "全场通用",
		"payValue":    8000,
		"actualValue": 10000,
		"type":        1,
		"stock":       100,
		"beginTime":   "2025-01-25T10:09:17",
		"endTime":     "2030-12-31T12:09:04",
	}
	request := HttpClient.R()
	request.Header.Set("Authorization", authorization)
	request = request.SetBody(payload)
	response, err := request.Post(AddSeckillVoucherUrl)
	if err != nil {
		return
	}
	var result models.Result
	err = json.Unmarshal(response.Body(), &result)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	assert.Equal(t, result.Success, true)
}
func cleanRedisDatabase(ctx context.Context, voucherId string, stock int) {
	// 删除redis购买记录
	RedisClient.Del(ctx, "seckill:order:"+voucherId)
	// 恢复redis库存
	RedisClient.Set(ctx, "seckill:stock:"+voucherId, strconv.Itoa(stock), -1)
	// 删除登录token
	keys := RedisClient.Keys(ctx, "login:*").Val()
	RedisClient.Del(ctx, keys...)
	// 删除自定义拦截器
	keys = RedisClient.Keys(ctx, "rate:*").Val()
	RedisClient.Del(ctx, keys...)
	keys = RedisClient.Keys(ctx, "{rate:*").Val()
	RedisClient.Del(ctx, keys...)
}
func cleanDatabase(t *testing.T, phoneToAuth map[string]string, voucherId string, stock int) {
	phones := make([]string, 0, len(phoneToAuth))
	for phone := range phoneToAuth {
		phones = append(phones, phone)
	}
	cleanMysqlDatabase(t, phoneToAuth, voucherId, stock)
}

func cleanMysqlDatabase(t *testing.T, phoneToAuth map[string]string, voucherId string, stock int) {
	// 恢复优惠券库存
	restoreMysqlVoucherStock(t, voucherId, stock)
	// 删除订单表中的订单
	deleteMysqlOrders(t, voucherId)
	// 清空消息表
	truncateMessages(t)

}

func truncateMessages(t *testing.T) {
	truncateMysqlTable(t, DBClient, viper.GetString("database.mysql.dbname"), viper.GetString("database.mysql.table.producer_message_table_name"))
	truncateMysqlTable(t, DBClient, viper.GetString("database.mysql.dbname"), viper.GetString("database.mysql.table.consumer_message_table_name"))
}

func truncateMysqlTable(t *testing.T, db *sql.DB, dbName, tableName string) {
	// 1. 检查表是否存在
	var tableExists bool
	checkQuery := `
        SELECT COUNT(*) > 0 
        FROM information_schema.tables 
        WHERE table_schema = ? 
        AND table_name = ?
    `
	err := db.QueryRow(checkQuery, dbName, tableName).Scan(&tableExists)
	if err != nil {
		t.Fatalf("failed to query table: %v", err)
	}

	if !tableExists {
		return
	}

	// 2. 动态执行 TRUNCATE（注意表名需安全处理）
	truncateQuery := fmt.Sprintf("TRUNCATE TABLE `%s`.`%s`", dbName, tableName)
	_, err = db.Exec(truncateQuery)
	if err != nil {
		t.Fatalf("failed to truncate table: %v", err)
	}
}

func deleteMysqlOrders(t *testing.T, voucherId string) {
	del := fmt.Sprintf("delete from tb_voucher_order where voucher_id = ? ")
	_, err := DBClient.Exec(del, voucherId)
	if err != nil {
		t.Fatalf("failed to exec delete voucher order: %s", err.Error())
	}
}

func restoreMysqlVoucherStock(t *testing.T, voucherId string, stock int) {
	update := "update tb_seckill_voucher set stock = ? where voucher_id = ?"
	_, err := DBClient.Exec(update, stock, voucherId)
	if err != nil {
		t.Fatalf("failed to exec update voucher: %s", err.Error())
	}
}

func TestSeckillVoucher(t *testing.T) {
	voucherId := viper.GetString("test.voucher.id")
	stock := viper.GetInt("test.voucher.stock")
	cleanRedisDatabase(context.Background(), voucherId, stock)
	err := os.Remove(AuthsFilePath)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("failed to remove auth file: %v", err)
	}
	TestGenerateAuths(t)
	phonesAndAuths := getPhonesAndAuths(t)
	cleanDatabase(t, phonesAndAuths, voucherId, stock)
	wg := &sync.WaitGroup{}
	wg.Add(len(phonesAndAuths))
	requestStats := utils.NewRequestStats()
	purchaseSeckillVoucher(phonesAndAuths, voucherId, wg, time.Duration(viper.GetInt("test.voucher.purchase_duration_sec"))*time.Second, requestStats)
	wg.Wait()
	assert.GreaterOrEqual(t, min(stock, len(phonesAndAuths)), int(requestStats.PurchaseSuccessCount.Load()))
	fmt.Println(requestStats)
}

func purchaseSeckillVoucherWorker(stats *utils.RequestStats, url string, request *resty.Request) {
	start := time.Now()
	response, err := request.Post(url)
	elapsed := time.Since(start)
	nanosecond := uint64(elapsed.Nanoseconds())
	if err != nil || response == nil {
		stats.Record(utils.ResponseFail, nanosecond)
		return
	}
	var result models.Result
	err = json.Unmarshal(response.Body(), &result)
	if err != nil {
		stats.Record(utils.ResponseFail, nanosecond)
		return
	}
	if result.Success {
		s := result.Data.String()
		_, err := strconv.ParseInt(s, 10, 64)
		if err == nil {
			stats.Record(utils.PurchaseSuccess, nanosecond)
			return
		}
	}
	stats.Record(utils.PurchaseFail, nanosecond)
}

func purchaseSeckillVoucherTimeoutContextWorker(ctx context.Context, stats *utils.RequestStats, url string, request *resty.Request) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			purchaseSeckillVoucherWorker(stats, url, request)
			//time.Sleep(time.Millisecond * 100)
		}
	}
}

func purchaseSeckillVoucher(phonesAndAuths map[string]string, voucherId string, wg *sync.WaitGroup, duration time.Duration, stats *utils.RequestStats) {
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()
	for phone := range phonesAndAuths {
		go func() {
			defer wg.Done()
			url := PurchaseSeckillVoucherUrlPrefix + "/" + voucherId
			auth := phonesAndAuths[phone]
			request := HttpClient.R()
			request.Header.Set("Authorization", auth)
			if duration == 0 {
				purchaseSeckillVoucherWorker(stats, url, request)
			} else {
				go purchaseSeckillVoucherTimeoutContextWorker(ctx, stats, url, request)
			}
		}()
	}
	extra := time.Second
	time.Sleep(duration + extra)
}

func TestRestoreMysqlStock(t *testing.T) {
	restoreMysqlVoucherStock(t, "5", 200)
}

func TestDeleteOrders(t *testing.T) {
	deleteMysqlOrders(t, "5")
}

func TestTruncateMessages(t *testing.T) {
	truncateMessages(t)
}
