package utils

import (
	"database/sql"
	"fmt"
	"github.com/go-redis/redis/v8"
	_ "github.com/go-sql-driver/mysql"
	"github.com/spf13/viper"
)

func InitMySQL() (*sql.DB, error) {
	dataSourceName := fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?parseTime=true",
		viper.GetString("database.mysql.user"),
		viper.GetString("database.mysql.password"),
		viper.GetString("database.mysql.host"),
		viper.GetInt("database.mysql.port"),
		viper.GetString("database.mysql.dbname"),
	)
	db, err := sql.Open("mysql", dataSourceName)

	if err == nil {
		err = db.Ping()
	}
	return db, err
}

func InitRedis() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     viper.GetString("database.redis.address"),
		Password: viper.GetString("database.redis.password"),
		DB:       viper.GetInt("database.redis.db"),
	})
}
