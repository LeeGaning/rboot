package rboot

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

// Adapter 是管理聊天转接器进出消息的接口
type Adapter interface {
	Incoming() chan *Message // 接收到的消息
	Outgoing() chan *Message // 回复的消息
}

type adapterF func(*Robot) Adapter

var adapters = make(map[string]adapterF)

// RegisterAdapter 注册转接器，名称不可重复
// 转接器需实现 Adapter 接口
func RegisterAdapter(name string, adp adapterF) {
	if name == "" {
		panic("RegisterAdapter: adapter must have a name")
	}
	if _, ok := adapters[name]; ok {
		panic("RegisterAdapter: adapter named " + name + " already registered. ")
	}
	adapters[name] = adp
}

// DetectAdapter 根据转接器名称获取转接器实例
func DetectAdapter(name string) (adapterF, error) {
	if adp, ok := adapters[name]; ok {
		return adp, nil
	}

	if len(adapters) == 0 {
		return nil, errors.New("no adapter available")
	}

	if name == "" {
		if len(adapters) == 1 {
			for _, adp := range adapters {
				return adp, nil
			}
		}
		return nil, errors.New("multiple adapters available; must choose one")
	}
	return nil, errors.New("unknown adapter " + name)
}

var (
	stdin  io.Reader = os.Stdin
	stdout io.Writer = os.Stdout
)

type cli struct {
	in     chan *Message
	out    chan *Message
	writer *bufio.Writer
}

// New returns an initialized adapter
func newCli(bot *Robot) Adapter {

	c := &cli{
		in:     make(chan *Message),
		out:    make(chan *Message),
		writer: bufio.NewWriter(stdout),
	}

	go c.run()

	return c
}

func (c *cli) Incoming() chan *Message {
	return c.in
}

func (c *cli) Outgoing() chan *Message {
	return c.out
}

// Run executes the adapter run loop
func (c *cli) run() {

	name := os.Getenv(`ROBOT_ALIAS`)
	if name == `` {
		name = os.Getenv(`ROBOT_NAME`)
	}

	go func() {
		scanner := bufio.NewScanner(stdin)
		for scanner.Scan() {
			msg := NewMessage(scanner.Text())

			c.in <- msg

		forLoop:
			for {
				select {
				case msg := <-c.out:
					c.writeString(msg.ToString())
				default:
					break forLoop
				}
			}
		}
	}()

	go func() {
		for msg := range c.out {
			c.writeString(msg.ToString())
		}
	}()
}

func (c *cli) writeString(str string) error {

	msg := fmt.Sprintf("%s\n", strings.TrimSpace(str))

	if _, err := c.writer.WriteString(msg); err != nil {
		return err
	}

	return c.writer.Flush()
}

func init() {
	RegisterAdapter(`cli`, newCli)
}
