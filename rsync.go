package sfserver

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"strings"
)

var syncEvent = make(chan *FileEvent, 10)

func runSync() {
	for e := range syncEvent {
		fmt.Println("开始同步文件:", e.fileName)

		var params []string
		params = append(params, conf.bin)
		params = append(params, conf.Params)
		params = append(params, "--password-file="+conf.Password_file)
		params = append(params, e.fileName)
		var out bytes.Buffer
		fmt.Println("exec command rsync:", out.String())
		for _, server := range conf.Server {
			cmd := exec.Command("/bin/sh", "-c", strings.Join(params, " ")+" "+conf.User+"@"+strings.TrimSpace(server))
			fmt.Println(cmd.Args)
			cmd.Stdout = &out
			err := cmd.Run()
			if err != nil {
				errLog(fmt.Sprintf("%s", cmd.Args))
				log.Fatal(err.Error())
			}
			fmt.Println("exec command rsync:", out.String())
		}
	}
}
