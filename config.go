package sfserver

import (
	"fmt"
	"github.com/vaughan0/go-ini"
	"log"
	"strings"
	"syscall"
)

type RsyncConfig struct {
	bin           string
	Params        string   ///usr/bin/rsync -avz --delete
	Password_file string   //密码文件存放地址
	User          string   //rsync指定的用户名
	Server        []string //rsync服务器
	Rskipdir      []string
	Rskipext      []string
}

type Config struct {
	Path    string         //监控路径
	Flags   uint32         //监控的事件
	Skipdir map[string]int //指定不需要监控的目录
	Skipext map[string]int //指定不需要监控的文件后缀
	Faillog string         //失败的日志文件

	RsyncConfig
}

const fsn = syscall.IN_CREATE | syscall.IN_MOVED_TO | syscall.IN_DELETE_SELF | syscall.IN_DELETE | syscall.IN_MODIFY | syscall.IN_MOVE_SELF | syscall.IN_MOVED_FROM

func NewConfig() *Config {

	var config = &Config{Flags: fsn, Faillog: "/tmp/sfserver.log", Skipdir: make(map[string]int), Skipext: make(map[string]int)}

	file, err := ini.LoadFile("config.ini")
	if err != nil {
		log.Fatal(err)
	}
	//读取配置参数
	fmt.Println("读取基本参数")
	for key, value := range file["global"] {
		//fmt.Println(key, "-->", value)
		switch key {
		case "path":
			if value != "" {
				config.Path = value
			}
		case "flags":
			fmt.Println("需要监控的事件", value)
			for _, flags := range strings.Split(value, ",") {
				flags = strings.TrimSpace(flags)
				if flags == "create" {
					config.Flags |= syscall.IN_CREATE | syscall.IN_MOVED_TO
				}

				if flags == "delete" {
					config.Flags |= syscall.IN_DELETE_SELF | syscall.IN_DELETE
				}

				if flags == "modify" {
					config.Flags |= syscall.IN_MODIFY
				}

				if flags == "rename" {
					config.Flags |= syscall.IN_MOVE_SELF | syscall.IN_MOVED_FROM
				}
			}

		case "skipdir":
			if value != "" {
				skipdir := strings.Split(value, ",")
				for _, dir := range skipdir {
					fmt.Println("=====>", strings.TrimSpace(dir), "<======")
					config.Skipdir[strings.TrimSpace(dir)] = 1
				}
			}
		case "skipext":
			if value != "" {
				skipext := strings.Split(value, ",")
				for _, ext := range skipext {
					config.Skipext[strings.TrimSpace(ext)] = 1
				}
			}

		case "faillog":
			if value != "" {
				config.Faillog = value
			}
		}
	}

	fmt.Println("开始读取rsync的配置")
	//读取rsync相关的参数
	for rkey, rvalue := range file["rsync"] {
		//fmt.Println(rkey, "-->", rvalue)
		switch rkey {
		case "rsync.bin":
			if rvalue != "" {
				config.bin = rvalue
			}
		case "rsync.params":
			if rvalue != "" {
				config.Params = rvalue
			}
		case "rsync.password_file":
			if rvalue != "" {
				config.Password_file = rvalue
			}
		case "rsync.user":
			if rvalue != "" {
				config.User = rvalue
			}
		case "rsync.server":
			if rvalue != "" {
				config.Server = strings.Split(rvalue, ",")
				fmt.Println(config.Server)
			}
		case "rsync.rskipdir":
			if rvalue != "" {
				config.Rskipdir = strings.Split(rvalue, ",")
			}
		case "rsync.rskipext":
			if rvalue != "" {
				config.Rskipext = strings.Split(rvalue, ",")
			}
		}
	}
	return config
}

func (c Config) String() string {
	return fmt.Sprintf("%s\n", c.Path)
}
