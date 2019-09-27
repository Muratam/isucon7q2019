package main

import (
	"net/http"

	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo"
)

const (
	avatarMaxBytes = 1 * 1024 * 1024
)
const LettersAndDigits = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var (
	db            *sqlx.DB
	ErrBadReqeust = echo.NewHTTPError(http.StatusBadRequest)
)
var isMasterServerIP = MyServerIsOnMasterServerIP()
var accountNameToIDServer = NewSyncMapServerConn(GetMasterServerAddress()+":8885", isMasterServerIP)
var idToUserServer = NewSyncMapServerConn(GetMasterServerAddress()+":8884", isMasterServerIP)
var userIdToLastReadServer = NewSyncMapServerConn(GetMasterServerAddress()+":8883", isMasterServerIP)
