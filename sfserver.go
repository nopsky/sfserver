package sfserver

import (
	"fmt"
	"log"
)

var conf *Config = NewConfig()

func Run() {

	fmt.Println("初始化配置文件")

	if conf.Path == "" {
		log.Fatal("请指定监控目录")
	}

	w, err := NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

	done := make(chan int)

	fmt.Println("监控目录为:", conf.Path)

	e := w.Watch(conf.Path, conf.Flags)

	if e != nil {
		log.Fatal(e)
	}

	RunSync()
	//w.PrintMap()
	<-done
}
