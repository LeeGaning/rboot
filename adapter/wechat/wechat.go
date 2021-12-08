package wechat

import (
	"github.com/ghaoo/rboot"
	"github.com/sirupsen/logrus"
)

type wx struct {
	in  chan *rboot.Message
	out chan *rboot.Message
	vx  *VxPush
	bot *rboot.Robot
}

func New(bot *rboot.Robot) rboot.Adapter {

	w := &wx{
		in:  make(chan *rboot.Message),
		out: make(chan *rboot.Message),
		vx:  NewVxPush(),
		bot: bot,
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
	// go w.vx.GetAccessToken() // 刷新token
	for out := range w.out {
		msg := &PushMsg{
			NameId: out.To,         //当前消息的用户名
			Msg:    out.ToString(), //消息内容
		}
		w.vx.MsgID(msg)
		logrus.Debugf("wechat send:%v", out)
	}

}

func init() {
	rboot.RegisterAdapter(`wechat`, New)
}
