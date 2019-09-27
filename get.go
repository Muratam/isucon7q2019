package main

import (
	"log"
	"net/http"
	"os"
	"os/exec"
	"sort"
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
	channelIdToMessagesServer.server.InitializeFunction = func() {
		messages := []Message{} // db.MustExec("DELETE FROM message WHERE id > 10000")
		err := db.Select(&messages, "SELECT * FROM message WHERE id <= 10000 ORDER BY id")
		if err != nil {
			panic(err)
		}
		localMap := map[string]interface{}{}
		for i := 1; i <= 10; i++ {
			localMap[strconv.Itoa(i)] = []Message{}
		}
		for i := 2711; i <= 2900; i++ {
			localMap[strconv.Itoa(i)] = []Message{}
		}

		for _, msg := range messages {
			key := strconv.Itoa(int(msg.ChannelID))
			if val, ok := localMap[key]; ok {
				localMap[key] = append(val.([]Message), msg)
			} else {
				localMap[key] = []Message{msg}
			}
		}
		channelIdToMessagesServer.MSet(localMap)
	}
	messageNumServer.server.InitializeFunction = func() {
		messageNumServer.Set("cnt", 10000)
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
		channelIdToMessagesServer.Initialize()
		messageNumServer.Initialize()
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
	channelIdStrs := channelIdToMessagesServer.AllKeys()
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
		cnt := channelIdToMessagesServer.LLen(strconv.Itoa(int(chID)))
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
	cnti := channelIdToMessagesServer.LLen(strconv.Itoa(int(chID)))
	cnt := int64(cnti)
	maxPage := int64(cnt+N-1) / N
	if maxPage == 0 {
		maxPage = 1
	}
	if page > maxPage {
		return ErrBadReqeust
	}

	offset := (page - 1) * N
	got := channelIdToMessagesServer.LRange(strconv.Itoa(int(chID)), int(-offset-1-N), int(-offset-1))
	messages := make([]Message, got.Len())
	for i := 0; i < got.Len(); i++ {
		got.Get(got.Len()-1-i, &messages[i])
	}
	// 5 6 7 8 9 10 [11 12 13]
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
	//  35 33 31 ... 20
	got := channelIdToMessagesServer.LRange(strconv.Itoa(int(chanID)), -100, -1)
	messagespre := make([]Message, got.Len())
	for i := 0; i < got.Len(); i++ {
		got.Get(i, &messagespre[i])
	}
	sort.Slice(messagespre, func(i, j int) bool {
		return messagespre[i].ID > messagespre[j].ID
	})
	messages := []Message{}
	for _, msg := range messagespre {
		if msg.ID <= lastID {
			break
		}
		messages = append(messages, msg)
	}
	// [35 33 31 ... 20] [18 ...]
	if len(messages) > 0 {
		preLastReads := map[int64]int64{}
		userIDStr := strconv.Itoa(int(userID))
		userIdToLastReadServer.Get(userIDStr, &preLastReads)
		cnti := channelIdToMessagesServer.LLen(strconv.Itoa(int(chanID)))
		preLastReads[chanID] = int64(cnti)
		userIdToLastReadServer.Set(userIDStr, preLastReads)
	}
	c.Response().WriteHeader(http.StatusOK)
	c.Response().Header()["Content-Type"] = []string{"application/json; charset=UTF-8"}
	c.Response().Write([]byte("["))
	userIDStrs := make([]string, len(messages))
	for i, m := range messages {
		userIDStrs[i] = strconv.Itoa(int(m.UserID))
	}
	mGot := idToUserServer.MGet(userIDStrs)
	for i := len(messages) - 1; i >= 0; i-- {
		m := messages[i]
		u := User{}
		mGot.Get(strconv.Itoa(int(m.UserID)), &u)
		c.Response().Write([]byte( // WARN escape
			`{"id":` + strconv.Itoa(int(m.ID)) +
				`,"date":"` + m.CreatedAt.Format("2006/01/02 15:04:05") + `"` +
				`,"content":"` + m.Content + `"` +
				`,"user":{"name":"` + u.Name + `"` +
				`,"display_name":"` + u.DisplayName + `"` +
				`,"avatar_icon":"` + u.AvatarIcon + `"` +
				`}}`))
		if i != 0 {
			c.Response().Write([]byte(","))
		}
	}
	c.Response().Write([]byte("]"))
	return nil
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
