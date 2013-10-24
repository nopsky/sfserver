package sfserver

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
)

var syncEvent = make(chan *FileEvent, 10)

func RunSync() {
	for e := range syncEvent {
		fmt.Println("开始同步文件:", e.fileName)
		cmd := exec.Command("rsync", "-avz", "--delete", "--password-file=/home/rsync.ps", e.fileName, "nopsky@11.11.11.12::backup")
		var out bytes.Buffer
		cmd.Stdout = &out
		err := cmd.Run()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("exec command rsync:", out.String())
		fmt.Println(cmd.Args)
	}
}
