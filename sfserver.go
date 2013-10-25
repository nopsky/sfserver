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

	//设置过滤的目录
	fmt.Printf("dir:%v --- ext:%v\n", conf.Skipdir, conf.Skipext)
	w.setSkipDir(conf.Skipdir)

	//设置过滤的后缀
	w.setSkipExt(conf.Skipext)

	e := w.Watch(conf.Path, conf.Flags)

	if e != nil {
		log.Fatal(e)
	}

	runSync()

	PrintMap(w.skipDir)
	PrintMap(w.skipExt)
	//w.PrintMap()
	<-done
}
