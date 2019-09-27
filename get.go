package main

import (
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"sync"

	"github.com/labstack/echo"
	"github.com/labstack/echo-contrib/session"
)

func setInitializeFunction() {
	idToUserServer.server.InitializeFunction = func() { // db.MustExec("DELETE FROM user WHERE id > 1000")
		users := []User{}
		idToUserServerMap := map[string]interface{}{}
		err := db.Select(&users, "SELECT * FROM user WHERE id <= 1000")
		if err != nil {
			panic(err)
		}
		for _, u := range users {
			key := strconv.Itoa(int(u.ID))
			idToUserServerMap[key] = u
		}
		idToUserServer.MSet(idToUserServerMap)
	}
	accountNameToIDServer.server.InitializeFunction = func() {
		users := []User{}
		accountNametoIDServerMap := map[string]interface{}{}
		err := db.Select(&users, "SELECT * FROM user WHERE id <= 1000")
		if err != nil {
			panic(err)
		}
		for _, u := range users {
			key := strconv.Itoa(int(u.ID))
			accountNametoIDServerMap[u.Name] = key
		}
		accountNameToIDServer.MSet(accountNametoIDServerMap)
	}
	channelIdToMessageCountServer.server.InitializeFunction = func() {
		idAndCounts := []IdAndCount{} // db.MustExec("DELETE FROM message WHERE id > 10000")
		err := db.Select(&idAndCounts, "SELECT channel_id,COUNT(*) as cnt FROM message WHERE id <= 10000 GROUP BY channel_id")
		if err != nil {
			panic(err)
		}
		idToCountMap := map[string]interface{}{}
		for i := 1; i <= 10; i++ {
			idToCountMap[strconv.Itoa(i)] = 0
		}
		for i := 2711; i <= 2900; i++ {
			idToCountMap[strconv.Itoa(i)] = 0
		}
		for _, ic := range idAndCounts {
			idToCountMap[strconv.Itoa(int(ic.ChannelID))] = int(ic.Count)
		}
		channelIdToMessageCountServer.MSet(idToCountMap)
	}
}

var sessionCache = sync.Map{}

func getInitialize(c echo.Context) error {
	sessionCache = sync.Map{}
	db.MustExec("DELETE FROM channel WHERE id > 10")
	db.MustExec("DELETE FROM message WHERE id > 10000")
	func() {
		// db.MustExec("DELETE FROM haveread")
		userIdToLastReadServer.FlushAll()
		accountNameToIDServer.Initialize()
		idToUserServer.Initialize()
		channelIdToMessageCountServer.Initialize()
	}()
	func() { // db.MustExec("DELETE FROM image WHERE id > 1001")
		exec.Command("rm -rf /home/isucon/icons").Run()
		exec.Command("mkdir /home/isucon/icons").Run()
		// Image : ID(>1001 は消える.Nameで取得.)/Name/DATA
		// ~/icons/ に保存されている。
		// 5秒くらいだし毎回やる？
		type Image struct {
			ID   int64  `db:"id"`
			Name string `db:"name"`
			Data []byte `db:"data"`
		}
		images := []Image{}
		err := db.Select(&images, "SELECT name, data FROM image WHERE ID <= 1001")
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
	}()
	return c.String(204, "")
}

func fetchUnread(c echo.Context) error {
	userID := sessUserID(c)
	if userID == 0 {
		return c.NoContent(http.StatusForbidden)
	}
	channelIdStrs := channelIdToMessageCountServer.AllKeys()
	mGot := channelIdToMessageCountServer.MGet(channelIdStrs)
	preLastReads := map[int64]int64{}
	userIDStr := strconv.Itoa(int(userID))
	userIdToLastReadServer.Get(userIDStr, &preLastReads)
	c.Response().WriteHeader(http.StatusOK)
	c.Response().Header()["Content-Type"] = []string{"application/json; charset=UTF-8"}
	c.Response().Write([]byte("["))
	for i, chIDStr := range channelIdStrs {
		chIDi, _ := strconv.Atoi(chIDStr)
		chID := int64(chIDi)
		read, ok := preLastReads[chID]
		if !ok {
			read = 0
		}
		cnt := 0
		ok = mGot.Get(strconv.Itoa(int(chID)), &cnt)
		if !ok {
			cnt = 0
		}
		c.Response().Write([]byte(`{"channel_id":` + strconv.Itoa(int(chID)) + `,"unread":` + strconv.Itoa(cnt-int(read)) + `}`))
		if i+1 != len(channelIdStrs) {
			c.Response().Write([]byte(","))
		}
	}
	c.Response().Write([]byte("]"))

	return nil
}

func getHistory(c echo.Context) error {
	chID, err := strconv.ParseInt(c.Param("channel_id"), 10, 64)
	if err != nil || chID <= 0 {
		return ErrBadReqeust
	}

	user, err := ensureLogin(c)
	if user == nil {
		return err
	}

	var page int64
	pageStr := c.QueryParam("page")
	if pageStr == "" {
		page = 1
	} else {
		page, err = strconv.ParseInt(pageStr, 10, 64)
		if err != nil || page < 1 {
			return ErrBadReqeust
		}
	}

	const N = 20
	var cnti int
	ok := channelIdToMessageCountServer.Get(strconv.Itoa(int(chID)), &cnti)
	if !ok {
		cnti = 0
	}
	cnt := int64(cnti)
	maxPage := int64(cnt+N-1) / N
	if maxPage == 0 {
		maxPage = 1
	}
	if page > maxPage {
		return ErrBadReqeust
	}

	messages := []Message{}
	err = db.Select(&messages,
		"SELECT * FROM message WHERE channel_id = ? ORDER BY id DESC LIMIT ? OFFSET ?",
		chID, N, (page-1)*N)
	if err != nil {
		return err
	}

	mjson := make([]map[string]interface{}, 0)
	for i := len(messages) - 1; i >= 0; i-- {
		r, err := jsonifyMessage(messages[i])
		if err != nil {
			return err
		}
		mjson = append(mjson, r)
	}

	channels := []ChannelInfo{}
	err = db.Select(&channels, "SELECT * FROM channel ORDER BY id")
	if err != nil {
		return err
	}
	viewshistoryhtml(c.Response(), map[string]interface{}{
		"ChannelID": chID,
		"Channels":  channels,
		"Messages":  mjson,
		"MaxPage":   maxPage,
		"Page":      page,
		"User":      user,
	})
	return nil
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

func getProfile(c echo.Context) error {
	self, err := ensureLogin(c)
	if self == nil {
		return err
	}

	channels := []ChannelInfo{}
	err = db.Select(&channels, "SELECT * FROM channel ORDER BY id")
	if err != nil {
		return err
	}

	userName := c.Param("user_name")
	idStr := ""
	ok := accountNameToIDServer.Get(userName, &idStr)
	if !ok {
		return echo.ErrNotFound
	}
	var other User
	ok = idToUserServer.Get(idStr, &other)
	if !ok {
		return echo.ErrNotFound
	}

	return c.Render(http.StatusOK, "profile", map[string]interface{}{
		"ChannelID":   0,
		"Channels":    channels,
		"User":        self,
		"Other":       other,
		"SelfProfile": self.ID == other.ID,
	})
}

func getAddChannel(c echo.Context) error {
	self, err := ensureLogin(c)
	if self == nil {
		return err
	}

	channels := []ChannelInfo{}
	err = db.Select(&channels, "SELECT * FROM channel ORDER BY id")
	if err != nil {
		return err
	}

	return c.Render(http.StatusOK, "add_channel", map[string]interface{}{
		"ChannelID": 0,
		"Channels":  channels,
		"User":      self,
	})
}

func getIndex(c echo.Context) error {
	userID := sessUserID(c)
	if userID != 0 {
		return c.Redirect(http.StatusSeeOther, "/channel/1")
	}

	return c.Render(http.StatusOK, "index", map[string]interface{}{
		"ChannelID": nil,
	})
}

func getChannel(c echo.Context) error {
	user, err := ensureLogin(c)
	if user == nil {
		return err
	}
	cID, err := strconv.Atoi(c.Param("channel_id"))
	if err != nil {
		return err
	}
	channels := []ChannelInfo{}
	err = db.Select(&channels, "SELECT * FROM channel ORDER BY id")
	if err != nil {
		return err
	}

	var desc string
	for _, ch := range channels {
		if ch.ID == int64(cID) {
			desc = ch.Description
			break
		}
	}
	return c.Render(http.StatusOK, "channel", map[string]interface{}{
		"ChannelID":   cID,
		"Channels":    channels,
		"User":        user,
		"Description": desc,
	})
}

func getRegister(c echo.Context) error {
	return c.Render(http.StatusOK, "register", map[string]interface{}{
		"ChannelID": 0,
		"Channels":  []ChannelInfo{},
		"User":      nil,
	})
}

func getLogin(c echo.Context) error {
	return c.Render(http.StatusOK, "login", map[string]interface{}{
		"ChannelID": 0,
		"Channels":  []ChannelInfo{},
		"User":      nil,
	})
}

func getLogout(c echo.Context) error {
	sess, _ := session.Get("session", c)
	delete(sess.Values, "user_id")
	sess.Save(c.Request(), c.Response())
	return c.Redirect(http.StatusSeeOther, "/")
}
