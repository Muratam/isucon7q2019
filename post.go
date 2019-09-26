package main

import (
	"crypto/sha1"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/labstack/echo"
)

func postAddChannel(c echo.Context) error {
	self, err := ensureLogin(c)
	if self == nil {
		return err
	}

	name := c.FormValue("name")
	desc := c.FormValue("description")
	if name == "" || desc == "" {
		return ErrBadReqeust
	}

	res, err := db.Exec(
		"INSERT INTO channel (name, description, updated_at, created_at) VALUES (?, ?, NOW(), NOW())",
		name, desc)
	if err != nil {
		return err
	}
	lastID, _ := res.LastInsertId()
	return c.Redirect(http.StatusSeeOther,
		fmt.Sprintf("/channel/%v", lastID))
}

func postProfile(c echo.Context) error {
	self, err := ensureLogin(c)
	if self == nil {
		return err
	}

	avatarName := ""
	var avatarData []byte

	if fh, err := c.FormFile("avatar_icon"); err == http.ErrMissingFile {
		// no file upload
	} else if err != nil {
		return err
	} else {
		dotPos := strings.LastIndexByte(fh.Filename, '.')
		if dotPos < 0 {
			return ErrBadReqeust
		}
		ext := fh.Filename[dotPos:]
		switch ext {
		case ".jpg", ".jpeg", ".png", ".gif":
			break
		default:
			return ErrBadReqeust
		}

		file, err := fh.Open()
		if err != nil {
			return err
		}
		avatarData, _ = ioutil.ReadAll(file)
		file.Close()

		if len(avatarData) > avatarMaxBytes {
			return ErrBadReqeust
		}

		avatarName = fmt.Sprintf("%x%s", sha1.Sum(avatarData), ext)
	}
	dirty := false
	if avatarName != "" && len(avatarData) > 0 {
		file, err := os.Create("/home/isucon/icons/" + avatarName)
		if err != nil {
			return err
		}
		defer file.Close()
		file.Write(avatarData)
		dirty = true
		self.AvatarIcon = avatarName
	}
	if name := c.FormValue("display_name"); name != "" {
		dirty = true
		self.DisplayName = name
	}
	if dirty {
		idToUserServer.Set(strconv.Itoa(int(self.ID)), self)
	}
	return c.Redirect(http.StatusSeeOther, "/")
}
func postMessage(c echo.Context) error {
	user, err := ensureLogin(c)
	if user == nil {
		return err
	}

	message := c.FormValue("message")
	if message == "" {
		return echo.ErrForbidden
	}

	var chanID int64
	if x, err := strconv.Atoi(c.FormValue("channel_id")); err != nil {
		return echo.ErrForbidden
	} else {
		chanID = int64(x)
	}

	if _, err := addMessage(chanID, user.ID, message); err != nil {
		return err
	}

	return c.NoContent(204)
}
func postRegister(c echo.Context) error {
	name := c.FormValue("name")
	pw := c.FormValue("password")
	if name == "" || pw == "" {
		return ErrBadReqeust
	}
	randomString := func(n int) string {
		b := make([]byte, n)
		z := len(LettersAndDigits)

		for i := 0; i < n; i++ {
			b[i] = LettersAndDigits[rand.Intn(z)]
		}
		return string(b)
	}
	register := func(name, password string) (int64, error) {
		if accountNameToIDServer.Exists(name) {
			return 0, echo.ErrForbidden
		}
		salt := randomString(20)
		digest := fmt.Sprintf("%x", sha1.Sum([]byte(salt+password)))
		user := User{
			Name:        name,
			Salt:        salt,
			Password:    digest,
			DisplayName: name,
			AvatarIcon:  "default.png",
			CreatedAt:   time.Now().Truncate(time.Second),
		}
		id := idToUserServer.Insert(user)
		accountNameToIDServer.Set(name, strconv.Itoa(int(id)))
		return int64(id), nil
	}
	userID, err := register(name, pw)
	if err != nil {
		return c.NoContent(http.StatusConflict)
	}
	sessSetUserID(c, userID)
	return c.Redirect(http.StatusSeeOther, "/")
}
func postLogin(c echo.Context) error {
	name := c.FormValue("name")
	pw := c.FormValue("password")
	if name == "" || pw == "" {
		return ErrBadReqeust
	}

	id := ""
	ok := accountNameToIDServer.Get(name, &id)
	if !ok {
		return echo.ErrForbidden
	}
	user := User{}
	idToUserServer.Get(id, &user)
	digest := fmt.Sprintf("%x", sha1.Sum([]byte(user.Salt+pw)))
	if digest != user.Password {
		return echo.ErrForbidden
	}
	sessSetUserID(c, user.ID)
	return c.Redirect(http.StatusSeeOther, "/")
}
