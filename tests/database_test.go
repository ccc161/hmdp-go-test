package tests

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"testing"
	"time"
)

// 数据库测试
func TestDBCreateAndDropTable(t *testing.T) {
	if DBClient == nil {
		t.Fatal("DBClient not initialized")
	}

	// 生成唯一表名
	ts := time.Now().Unix()
	u := uuid.New().String()
	tableName := fmt.Sprintf("tb_%d_%s", ts, u)

	// 创建表
	createSQL := fmt.Sprintf("CREATE TABLE `%s` (id INT PRIMARY KEY)", tableName)
	if _, err := DBClient.Exec(createSQL); err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// 清理表
	defer func() {
		dropSQL := fmt.Sprintf("DROP TABLE `%s`", tableName)
		if _, err := DBClient.Exec(dropSQL); err != nil {
			t.Fatalf("Failed to drop table: %v", err)
		}
	}()
}

// Redis测试
func TestRedisSetAndDelKey(t *testing.T) {
	if RedisClient == nil {
		t.Fatal("RedisClient not initialized")
	}

	// 生成唯一键名
	ctx := context.Background()
	ts := time.Now().Unix()
	u := uuid.New().String()
	key := fmt.Sprintf("test_key_%d_%s", ts, u)

	// 设置键值
	if err := RedisClient.Set(ctx, key, u, 0).Err(); err != nil {
		t.Fatalf("Failed to set key: %v", err)
	}

	// 删除键值
	if err := RedisClient.Del(ctx, key).Err(); err != nil {
		t.Fatalf("Failed to delete key: %v", err)
	}
}
