package main

import (
	"fmt"
	"log"
	"strings"
	"syscall"
	"unsafe"
)

const (
	// Options for inotify_init() are not exported
	// sys_IN_CLOEXEC    uint32 = syscall.IN_CLOEXEC
	// sys_IN_NONBLOCK   uint32 = syscall.IN_NONBLOCK

	// Options for AddWatch
	sys_IN_DONT_FOLLOW uint32 = syscall.IN_DONT_FOLLOW
	sys_IN_ONESHOT     uint32 = syscall.IN_ONESHOT
	sys_IN_ONLYDIR     uint32 = syscall.IN_ONLYDIR

	// The "sys_IN_MASK_ADD" option is not exported, as AddWatch
	// adds it automatically, if there is already a watch for the given path
	// sys_IN_MASK_ADD      uint32 = syscall.IN_MASK_ADD

	// Events
	sys_IN_ACCESS        uint32 = syscall.IN_ACCESS
	sys_IN_ALL_EVENTS    uint32 = syscall.IN_ALL_EVENTS
	sys_IN_ATTRIB        uint32 = syscall.IN_ATTRIB
	sys_IN_CLOSE         uint32 = syscall.IN_CLOSE
	sys_IN_CLOSE_NOWRITE uint32 = syscall.IN_CLOSE_NOWRITE
	sys_IN_CLOSE_WRITE   uint32 = syscall.IN_CLOSE_WRITE
	sys_IN_CREATE        uint32 = syscall.IN_CREATE
	sys_IN_DELETE        uint32 = syscall.IN_DELETE
	sys_IN_DELETE_SELF   uint32 = syscall.IN_DELETE_SELF
	sys_IN_MODIFY        uint32 = syscall.IN_MODIFY
	sys_IN_MOVE          uint32 = syscall.IN_MOVE
	sys_IN_MOVED_FROM    uint32 = syscall.IN_MOVED_FROM
	sys_IN_MOVED_TO      uint32 = syscall.IN_MOVED_TO
	sys_IN_MOVE_SELF     uint32 = syscall.IN_MOVE_SELF
	sys_IN_OPEN          uint32 = syscall.IN_OPEN

	sys_AGNOSTIC_EVENTS = sys_IN_MOVED_TO | sys_IN_MOVED_FROM | sys_IN_CREATE | sys_IN_ATTRIB | sys_IN_MODIFY | sys_IN_MOVE_SELF | sys_IN_DELETE | sys_IN_DELETE_SELF

	// Special events
	sys_IN_ISDIR      uint32 = syscall.IN_ISDIR
	sys_IN_IGNORED    uint32 = syscall.IN_IGNORED
	sys_IN_Q_OVERFLOW uint32 = syscall.IN_Q_OVERFLOW
	sys_IN_UNMOUNT    uint32 = syscall.IN_UNMOUNT
)

func main() {
	var (
		buf   [syscall.SizeofInotifyEvent * 4096]byte // Buffer for a maximum of 4096 raw events
		n     int                                     // Number of bytes read with read()
		errno error                                   // Syscall errno
	)
	fd, erron := syscall.InotifyInit()

	if fd == -1 {
		log.Fatal("inotify init error", erron)
	}

	wd, err := syscall.InotifyAddWatch(fd, "/work/golang/test", sys_AGNOSTIC_EVENTS)
	if err != nil {
		log.Fatal("add watch error", err)
	}
	n, errno = syscall.Read(fd, buf[0:])

	if n == 0 {
		syscall.Close(fd)
	}

	if n < syscall.SizeofInotifyEvent {
		log.Fatal("size of InotifyEvent error", errno)
	}

	var offset uint32 = 0
	var filename string

	for offset <= uint32(n-syscall.SizeofInotifyEvent) {
		raw := (*syscall.InotifyEvent)(unsafe.Pointer(&buf[offset]))
		if raw.Len > 0 {
			// Point "bytes" at the first byte of the filename
			bytes := (*[syscall.PathMax]byte)(unsafe.Pointer(&buf[offset+syscall.SizeofInotifyEvent]))
			// The filename is padded with NUL bytes. TrimRight() gets rid of those.
			filename += "/" + strings.TrimRight(string(bytes[0:raw.Len]), "\000")
		}
		var mask uint32 = raw.Mask
		if (mask & sys_IN_CREATE) == sys_IN_CREATE {
			if (mask & sys_IN_ISDIR) == sys_IN_ISDIR {
				fmt.Printf("新目录:%s 被创建了\n", filename)
			} else {
				fmt.Printf("新文件:%s 被创建了\n", filename)
			}
		} else if (mask & sys_IN_DELETE) == sys_IN_DELETE {
			if (mask & sys_IN_ISDIR) == sys_IN_ISDIR {
				fmt.Printf("删除的目录:%s\n", filename)
			} else {
				fmt.Printf("删除的文件:%s\n", filename)
			}
		}
		offset += syscall.SizeofInotifyEvent + raw.Len
	}

	syscall.InotifyRmWatch(fd, uint32(wd))

	syscall.Close(fd)
}
