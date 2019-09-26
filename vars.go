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