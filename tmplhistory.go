package main

import (
	"net/http"
	"strconv"
)

func viewshistoryhtml(w http.ResponseWriter, data map[string]interface{}) {
	ChannelID := data["ChannelID"].(int64)
	Channels := data["Channels"].([]ChannelInfo)
	Messages := data["Messages"].([]map[string]interface{})
	MaxPage := data["MaxPage"].(int64)
	Page := data["Page"].(int64)
	Me := data["User"].(*User)
	w.WriteHeader(http.StatusOK)
	w.Header()["Content-Type"] = []string{"text/html; charset=utf-8"}
	// header
	w.Write([]byte(` <!DOCTYPE html><html><head><meta http-equiv="Content-Type" content="text/html" charset="utf-8"><title>Isubata</title><link rel="stylesheet" href="/css/bootstrap.min.css"><link rel="stylesheet" href="/css/main.css"><script type="text/javascript" src="/js/jquery.min.js"></script><script type="text/javascript" src="/js/tether.min.js"></script><script type="text/javascript" src="/js/bootstrap.min.js"></script></head><body><nav class="navbar navbar-toggleable-md navbar-inverse fixed-top bg-inverse"><button class="navbar-toggler navbar-toggler-right hidden-lg-up" type="button" data-toggle="collapse" data-target="#navbarsExampleDefault" aria-controls="navbarsExampleDefault" aria-expanded="false" aria-label="Toggle navigation"><span class="navbar-toggler-icon"></span></button><a class="navbar-brand" href="/">Isubata</a><div class="collapse navbar-collapse" id="navbarsExampleDefault"><ul class="nav navbar-nav ml-auto"> `))
	if ChannelID != 0 {
		w.Write([]byte(` <li class="nav-item"><a href="/history/`))
		w.Write([]byte(strconv.Itoa(int(ChannelID))))
		w.Write([]byte(`" class="nav-link">チャットログ</a></li> `))
	}
	if Me != nil {
		w.Write([]byte(` <li class="nav-item"><a href="/add_channel" class="nav-link">チャンネル追加</a></li><li class="nav-item"><a href="/profile/`))
		w.Write([]byte(Me.Name))
		w.Write([]byte(`" class="nav-link">`))
		w.Write([]byte(Me.DisplayName))
		w.Write([]byte(`</a></li><li class="nav-item"><a href="/logout" class="nav-link">ログアウト</a></li> `))
	} else {
		w.Write([]byte(` <li><a href="/register" class="nav-link">新規登録</a></li><li><a href="/login" class="nav-link">ログイン</a></li> `))
	}
	w.Write([]byte(` </ul></div></nav><div class="container-fluid"><div class="row"><nav class="col-sm-3 col-md-3 hidden-xs-down bg-faded sidebar"> `))
	if Me != nil {
		w.Write([]byte(` <ul class="nav nav-pills flex-column"> `))
		for _, ch := range Channels {
			w.Write([]byte(` <li class="nav-item"><a class="nav-link justify-content-between `))
			if ChannelID != ch.ID { // WARN:
				w.Write([]byte(` active `))
			}
			w.Write([]byte(`" href="/channel/`))
			w.Write([]byte(strconv.Itoa(int(ch.ID))))
			w.Write([]byte(`"> `))
			w.Write([]byte(ch.Name))
			w.Write([]byte(` <span class="badge badge-pill badge-primary float-right" id="unread-`))
			w.Write([]byte(strconv.Itoa(int(ch.ID))))
			w.Write([]byte(`"></span></a></li> `))
		}
		w.Write([]byte(` </ul> `))
	}
	w.Write([]byte(` </nav><main class="col-sm-9 offset-sm-3 col-md-9 offset-md-3 pt-3"> `))
	// --------
	w.Write([]byte(` <div id="history"> `))

	for _, message := range Messages {
		user := message["user"].(User)
		w.Write([]byte(` <div class="media message"><img class="avatar d-flex align-self-start mr-3" src="/icons/`))
		w.Write([]byte(user.AvatarIcon))
		w.Write([]byte(`" alt="no avatar"><div class="media-body"><h5 class="mt-0"><a href="/profile/`))
		w.Write([]byte(user.Name))
		w.Write([]byte(`">`))
		w.Write([]byte(user.DisplayName))
		w.Write([]byte(`@`))
		w.Write([]byte(user.Name))
		w.Write([]byte(`</a></h5><p class="content">`))
		w.Write([]byte(message["content"].(string)))
		w.Write([]byte(`</p><p class="message-date">`))
		w.Write([]byte(message["date"].(string)))
		w.Write([]byte(`</p></div></div> `))
	}
	w.Write([]byte(` </div><nav><ul class="pagination"> `))
	if Page != 1 {
		w.Write([]byte(` <li><a href="/history/`))
		w.Write([]byte(strconv.Itoa(int(ChannelID))))
		w.Write([]byte(`?page=`))
		w.Write([]byte(strconv.Itoa(int(Page - 1))))
		w.Write([]byte(`"><span>«</span></a></li> `))
	}
	for _, p := range tRange(1, MaxPage) { // WARN:
		if p == Page {
			w.Write([]byte(`<li class="active">`))
		} else {
			w.Write([]byte(`<li>`))
		}
		w.Write([]byte(` <a href="/history/`))
		w.Write([]byte(strconv.Itoa(int(ChannelID))))
		w.Write([]byte(`?page=`))
		w.Write([]byte(strconv.Itoa(int(p))))
		w.Write([]byte(`">`))
		w.Write([]byte(strconv.Itoa(int(p))))
		w.Write([]byte(`</a></li> `))
	}
	if Page != MaxPage {
		w.Write([]byte(` <li><a href="/history/`))
		w.Write([]byte(strconv.Itoa(int(ChannelID))))
		w.Write([]byte(`?page=`))
		w.Write([]byte(strconv.Itoa(int(Page + 1))))
		w.Write([]byte(`"><span>»</span></a></li> `))
	}
	w.Write([]byte(` </ul></nav> `))
	// footer
	w.Write([]byte(` </main></div></div></body></html> `))
}
