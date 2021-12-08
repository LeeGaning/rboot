package bearychat

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/ghaoo/rboot"
	"github.com/sirupsen/logrus"
)

// bearychat adapter
type beary struct {
	in  chan *rboot.Message
	out chan *rboot.Message
}

func newBeary(bot *rboot.Robot) rboot.Adapter {
	b := &beary{
		in:  make(chan *rboot.Message),
		out: make(chan *rboot.Message),
	}

	b.run(bot)

	return b
}

func (b *beary) Name() string {
	return `bearychat`
}

func (b *beary) Incoming() chan *rboot.Message {
	return b.in
}

func (b *beary) Outgoing() chan *rboot.Message {
	return b.out
}

// 监听 rboot 需要发送给 bearychat 的消息
func (b *beary) listenOutgoing() {
	for msg := range b.out {
		res := Response{
			Text:         msg.ToString(),
			Channel:      msg.To,
			User:         msg.Sender,
			Notification: msg.Header.Get("Notification"),
		}

		if msg.Header.Get("msgtype") != "" && msg.Header.Get("msgtype") != "markdown" {
			res.Markdown = false
		}

		// 图片使用 Header 传递，图片链接用 “,” 隔开
		hatts := msg.Header.Get("Attachments")
		if hatts != "" {
			var attachments []Attachment
			err := json.Unmarshal([]byte(hatts), &attachments)
			if err != nil {
				logrus.WithField("func", "bearychat listenOutgoing unmarshal attachments").Errorf("listen outgoing message err: %v", err)
			}

			res.Attachments = attachments
		}

		if err := sendMessage(res); err != nil {
			logrus.WithField("func", "bearychat listenOutgoing send msg").Errorf("listen outgoing message err: %v", err)
		}
	}
}

// 监听 bearychat 传入 rboot 的消息
func (b *beary) listenIncoming(w http.ResponseWriter, r *http.Request) {

	fmt.Println("bearychat incoming ...")
	req := Request{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		w.WriteHeader(http.StatusNotImplemented)
		logrus.WithField("func", "bearychat listenIncoming").Errorf("listen incoming message err: %v", err)
		return
	}

	// 验证token
	if req.Token != os.Getenv("BEARYCHAT_TOKEN") {
		w.WriteHeader(http.StatusNotExtended)
		return
	}

	// 删除 bearychat 设置的 TRIGGER_WORD
	req.Text = strings.TrimPrefix(req.Text, os.Getenv("BEARYCHAT_TRIGGER_WORD"))

	msg := rboot.NewMessage(req.Text)
	msg.From = req.ChannelName
	msg.Sender = req.UserName

	b.in <- msg

}

func (b *beary) run(bot *rboot.Robot) {
	go b.listenOutgoing()

	bot.Router.HandleFunc("/beary", b.listenIncoming).Methods("POST", "GET").Name("beary_listen_message")
}

func init() {
	rboot.RegisterAdapter(`bearychat`, newBeary)
}
