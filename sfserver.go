package sfserver

import (
	"fmt"
	"log"
	"os"
)

var conf *Config = NewConfig()

func Run() {

	fmt.Println("初始化配置文件")

	if conf.Path == "" {
		log.Fatal("请指定监控目录")
	}

	w, err := NewWatcher()
	if err != nil {
		errLog("初始化notify失败")
		log.Fatal(err)
	}

	done := make(chan int)

	fmt.Println("监控目录为:", conf.Path)

	//设置过滤的目录
	w.setSkipDir(conf.Skipdir)

	//设置过滤的后缀
	w.setSkipExt(conf.Skipext)

	e := w.Watch(conf.Path, conf.Flags)

	if e != nil {
		log.Fatal(e)
	}

	runSync()

	<-done
}

func errLog(msg string) {
	fp, err := os.OpenFile(conf.Faillog, os.O_RDWR|os.O_CREATE|os.O_APPEND, os.ModePerm)
	defer fp.Close()
	if err != nil {
		log.Fatal(err)
	}
	fp.WriteString(msg + "\n")
}
