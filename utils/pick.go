package main

import (
	"fmt"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

func main() {
	dsn := fmt.Sprintf("%s%s@tcp(%s:%s)/isubata?parseTime=true&loc=Local&charset=utf8mb4",
		"isucon", ":isucon", "127.0.0.1", "3306")
	db, err := sqlx.Open("mysql", dsn)
	if err != nil {
		panic(err)
	}
	defer db.Close()
	type Image struct {
		ID   int64  `db:"id"`
		Name string `db:"name"`
		Data []byte `db:"data"`
	}
	// Image : ID(>1001 は消える.Nameで取得.)/Name/DATA
	// ~/icons/ に保存されている。
	// 5秒くらいだし毎回やる？
	images := []Image{}
	err = db.Select(&images, "SELECT name, data FROM image WHERE ID <= 1001")
	if err != nil {
		panic(err)
	}
	for _, image := range images {
		file, err := os.Create("/home/isucon/icons/" + image.Name)
		if err != nil {
			log.Panic(err)
		}
		file.Write(image.Data)
		file.Close()
	}
}
