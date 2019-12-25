package main

import (
	_ "github.com/ghaoo/rboot/adapter"
	_ "github.com/ghaoo/rboot/scripts"
	_ "github.com/ghaoo/rboot/brains/bolt"

	"github.com/ghaoo/rboot"
	"github.com/sirupsen/logrus"
)

func main() {

	bot := rboot.New()

	bot.Go()
}

func init() {
	logrus.SetLevel(logrus.TraceLevel)
	logrus.SetFormatter(&logrus.TextFormatter{
		ForceColors: true,
	})
}
