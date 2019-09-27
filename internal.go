package main

import (
	crand "crypto/rand"
	"database/sql"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/sessions"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo"
	"github.com/labstack/echo-contrib/session"
)

func (r *Renderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return r.templates.ExecuteTemplate(w, name, data)
}

func init() {
	seedBuf := make([]byte, 8)
	crand.Read(seedBuf)
	rand.Seed(int64(binary.LittleEndian.Uint64(seedBuf)))

	db_host := os.Getenv("ISUBATA_DB_HOST")
	if db_host == "" {
		db_host = "127.0.0.1"
	}
	db_port := "3306" //os.Getenv("ISUBATA_DB_PORT")
	//if db_port == "" {
	//	db_port = "3306"
	//}
	db_user := "isucon" //os.Getenv("ISUBATA_DB_USER")
	//if db_user == "" {
	//	db_user = "isucon"
	//}
	db_password := ":isucon" // os.Getenv("ISUBATA_DB_PASSWORD")
	//if db_password != "" {
	//	db_password = "isucon" //":" + db_password
	//}

	dsn := fmt.Sprintf("%s%s@tcp(%s:%s)/isubata?parseTime=true&loc=Local&charset=utf8mb4",
		db_user, db_password, db_host, db_port)

	log.Printf("Connecting to db: %q", dsn)
	db, _ = sqlx.Connect("mysql", dsn)
	for {
		err := db.Ping()
		if err == nil {
			break
		}
		log.Println(err)
		time.Sleep(time.Second * 3)
	}

	db.SetMaxOpenConns(20)
	db.SetConnMaxLifetime(5 * time.Minute)
	log.Printf("Succeeded to connect db.")
}

func getUser(userID int64) (*User, error) {
	u := User{}
	ok := idToUserServer.Get(strconv.Itoa(int(userID)), &u)
	if !ok {
		return nil, sql.ErrNoRows
	}
	return &u, nil
}

func sessUserID(c echo.Context) int64 {
	sess, _ := session.Get("session", c)
	var userID int64
	if x, ok := sess.Values["user_id"]; ok {
		userID, _ = x.(int64)
	}
	return userID
}

func sessSetUserID(c echo.Context, id int64) {
	sess, _ := session.Get("session", c)
	sess.Options = &sessions.Options{
		HttpOnly: true,
		MaxAge:   360000,
	}
	sess.Values["user_id"] = id
	sess.Save(c.Request(), c.Response())
}

func ensureLogin(c echo.Context) (*User, error) {
	var user *User
	var err error

	userID := sessUserID(c)
	if userID == 0 {
		c.Redirect(http.StatusSeeOther, "/login")
		return nil, nil
	}

	user, err = getUser(userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		sess, _ := session.Get("session", c)
		delete(sess.Values, "user_id")
		sess.Save(c.Request(), c.Response())
		c.Redirect(http.StatusSeeOther, "/login")
		return nil, nil
	}
	return user, nil

}

// TODO: 多分N+1
func jsonifyMessage(m Message) (map[string]interface{}, error) {
	u := User{}
	ok := idToUserServer.Get(strconv.Itoa(int(m.UserID)), &u)
	if !ok {
		return nil, echo.ErrNotFound
	}
	r := make(map[string]interface{})
	r["id"] = m.ID
	r["user"] = u
	r["date"] = m.CreatedAt.Format("2006/01/02 15:04:05")
	r["content"] = m.Content
	return r, nil
}
func getMessage(c echo.Context) error {
	userID := sessUserID(c)
	if userID == 0 {
		return c.NoContent(http.StatusForbidden)
	}

	chanID, err := strconv.ParseInt(c.QueryParam("channel_id"), 10, 64)
	if err != nil {
		return err
	}
	lastID, err := strconv.ParseInt(c.QueryParam("last_message_id"), 10, 64)
	if err != nil {
		return err
	}

	messages := []Message{}
	err = db.Select(&messages,
		"SELECT * FROM message WHERE id > ? AND channel_id = ? ORDER BY id DESC LIMIT 100",
		lastID, chanID)
	if err != nil {
		return err
	}

	response := make([]map[string]interface{}, 0)
	for i := len(messages) - 1; i >= 0; i-- {
		m := messages[i]
		r, err := jsonifyMessage(m)
		if err != nil {
			return err
		}
		response = append(response, r)
	}

	if len(messages) > 0 {
		// WARN: 遅そう.トランザクションは???
		preLastReads := map[int64]int64{}
		userIDStr := strconv.Itoa(int(userID))
		userIdToLastReadServer.Get(userIDStr, &preLastReads)
		var cnt int64
		// 読んだ個数を記録
		err = db.Get(&cnt, "SELECT COUNT(*) FROM message WHERE channel_id = ? AND id <= ?",
			chanID, messages[0].ID)
		if err != nil {
			return err
		}
		preLastReads[chanID] = cnt
		userIdToLastReadServer.Set(userIDStr, preLastReads)
	}

	return c.JSON(http.StatusOK, response)
}

func tAdd(a, b int64) int64 {
	return a + b
}

func tRange(a, b int64) []int64 {
	r := make([]int64, b-a+1)
	for i := int64(0); i <= (b - a); i++ {
		r[i] = a + i
	}
	return r
}
