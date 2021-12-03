package wechat

import (
	"strconv"
	"strings"

	"github.com/ghaoo/rboot"

)

type wx struct {
	in  chan *rboot.Message
	out chan *rboot.Message

	bot    *rboot.Robot
}
//	Appid
	Secret
	Templateid string `yaml:"Templateid"`
	Url        string `yaml:"Url"`
func New(bot *rboot.Robot) rboot.Adapter {
	// 初始化微信
	corpid := os.Getenv("WX_APPID")
	secret := os.Getenv("WX_SECRET")
	agentid := os.Getenv("WX_TEMPLATEID")
	agentid := os.Getenv("WX_TEMPLATEID")

	w := &wx{
		in:     make(chan *rboot.Message),
		out:    make(chan *rboot.Message),
		bot:    bot,
	}

	go w.run()

	return w
}

func (w *wx) Name() string {
	return "wechat"
}

func (w *wx) Incoming() chan *rboot.Message {
	return w.in
}

func (w *wx) Outgoing() chan *rboot.Message {
	return w.out
}

func (w *wx) run() {
	go func() {
		for msg := range w.out {
			if len(msg.Header.GetKey("file")) > 0 {

				for _, f := range msg.Header.GetKey("file") {
					w.client.SendFile(f, msg.To)
				}
			}

			w.client.SendTextMsg(msg.String(), msg.To)
		}
	}()

	es := w.client.Stream

	for e := range es.Event {
		switch e.Type {
		case sdk.EVENT_STOP_LOOP:
			return
		case sdk.EVENT_NEW_MESSAGE:
			msg := e.Data.(sdk.MsgData)

			isFriend := false
			if c := w.client.ContactByUserName(msg.SenderUserName); c != nil {
				if c.Type == sdk.Friend || c.Type == sdk.FriendAndMember {
					isFriend = true
				}
			}

			content := msg.Content

			if msg.AtMe {
				atme := `@`
				if len(w.client.MySelf.DisplayName) > 0 {
					atme += w.client.MySelf.DisplayName
				} else {
					atme += w.client.MySelf.NickName
				}
				content = strings.TrimSpace(strings.TrimPrefix(content, atme))
			}

			rmsg := rboot.NewMessage(content)
			rmsg.To = msg.ToUserName
			rmsg.From = msg.FromUserName
			rmsg.Sender = msg.SenderUserName
			rmsg.Header.Set("AtMe", strconv.FormatBool(msg.AtMe))
			rmsg.Header.Set("SendByMySelf", strconv.FormatBool(msg.IsSendedByMySelf))
			rmsg.Header.Set("GroupMsg", strconv.FormatBool(msg.IsGroupMsg))
			rmsg.Header.Set("IsFriend", strconv.FormatBool(isFriend))

			w.in <- rmsg
		}

	}
}

func init() {
	rboot.RegisterAdapter(`wechat`, New)
}
