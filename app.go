package main

import (
	"html/template"
	"log"
	"net/http"
	_ "net/http/pprof"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/middleware"
)

func main() {
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()
	e := echo.New()
	funcs := template.FuncMap{
		"add":    tAdd,
		"xrange": tRange,
	}
	e.Renderer = &Renderer{
		templates: template.Must(template.New("").Funcs(funcs).ParseGlob("views/*.html")),
	}
	e.Use(session.Middleware(sessions.NewCookieStore([]byte("secretonymoris"))))
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "request:\"${method} ${uri}\" status:${status} latency:${latency} (${latency_human}) bytes:${bytes_out}\n",
	}))
	e.Use(middleware.Static("../public"))
	// get init
	e.GET("/initialize", getInitialize)
	// get
	e.GET("/", getIndex)
	e.GET("/register", getRegister)
	e.GET("/login", getLogin)
	e.GET("/logout", getLogout)
	e.GET("/channel/:channel_id", getChannel)
	e.GET("/message", getMessage)
	e.GET("/fetch", fetchUnread)
	e.GET("/history/:channel_id", getHistory)
	e.GET("/profile/:user_name", getProfile)
	e.GET("/profile/:user_name/", getProfile)
	e.GET("add_channel", getAddChannel)
	// post
	e.POST("/register", postRegister)
	e.POST("/login", postLogin)
	e.POST("/message", postMessage)
	e.POST("/profile", postProfile)
	e.POST("add_channel", postAddChannel)
	// start
	setInitializeFunction()
	e.Start(":5000")
}
