package sfserver

import (
	//"bytes"
	"fmt"
	"log"
	"os"
	//"os/exec"
	"syscall"
	//"errors"
	"path/filepath"
	"strings"
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

	// Block for 100ms on each call to Select
	selectWaitTime = 100e6
)

type FileEvent struct {
	wd       int32
	mask     uint32
	cookie   uint32
	nameLen  uint32
	fileName string
}

// IsCreate reports whether the FileEvent was triggerd by a creation
func (e *FileEvent) IsCreate() bool {
	return (e.mask&sys_IN_CREATE) == sys_IN_CREATE || (e.mask&sys_IN_MOVED_TO) == sys_IN_MOVED_TO
}

// IsDelete reports whether the FileEvent was triggerd by a delete
func (e *FileEvent) IsDelete() bool {
	return (e.mask&sys_IN_DELETE_SELF) == sys_IN_DELETE_SELF || (e.mask&sys_IN_DELETE) == sys_IN_DELETE
}

// IsModify reports whether the FileEvent was triggerd by a file modification or attribute change
func (e *FileEvent) IsModify() bool {
	return ((e.mask & sys_IN_MODIFY) == sys_IN_MODIFY)
}

// IsRename reports whether the FileEvent was triggerd by a change name
func (e *FileEvent) IsRename() bool {
	return ((e.mask&sys_IN_MOVE_SELF) == sys_IN_MOVE_SELF || (e.mask&sys_IN_MOVED_FROM) == sys_IN_MOVED_FROM)
}

type Watcher struct {
	wm          *WatchMap
	fd          int             "inotify事件描述符"
	acceptEvent chan *FileEvent "接受事件通道"
	handleEvent chan *FileEvent "处理事件通道"
	Error       chan error      "错误通道"
	done        chan bool       "当监控事件关闭时,发送退出信息到此通道"
	isClose     bool            "判断是否已关闭"
	skipDir     map[string]int  "不监控的目录"
	skipExt     map[string]int  "不监控的文件后缀"
}

func NewWatcher() (w *Watcher, err error) {
	fd, err := syscall.InotifyInit()
	if fd == -1 {
		log.Fatal("Watcher Init Error", err)
	}

	watcher := &Watcher{
		wm:          newWatchMap(),
		fd:          fd,
		acceptEvent: make(chan *FileEvent),
		handleEvent: make(chan *FileEvent),
		Error:       make(chan error),
		done:        make(chan bool),
		skipDir:     make(map[string]int),
		skipExt:     make(map[string]int),
		isClose:     false,
	}

	go watcher.readEvent()   //读取事件,把事件发送到acceptEvent通道中
	go watcher.purgeEvents() //处理时间,把处理的结果发送到handleEvent通道中

	return watcher, nil
}

func (w *Watcher) setSkipDir(skipDir map[string]int) {
	w.skipDir = skipDir
	fmt.Println(w.skipDir)
}

func (w *Watcher) setSkipExt(skipExt map[string]int) {
	w.skipExt = skipExt
	fmt.Println(w.skipExt)
}

func PrintMap(m map[string]int) {
	for a, b := range m {
		fmt.Println(a, "->", b)
	}
}

func (w *Watcher) Watch(path string, flags uint32) error {
	f, err := os.Lstat(path)

	if err != nil {
		return err
	}
	//如果起始监控的是目录,则监控所有的目录
	if f.IsDir() {
		//判断是否是需要过滤的监控的目录
		// if _, found := w.skipDir[path]; found {
		// 	fmt.Println("过滤目录:", path)
		// 	return nil
		// }

		err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
			if info.IsDir() {
				//判断是否是需要过滤的监控的目录
				if _, found := w.skipDir[path]; found {
					fmt.Println("过滤目录:", path)
					return nil
				}
				w.AddWatch(path, flags)
			}
			return nil
		})

		if err != nil {
			return err
		}
	} else {
		w.AddWatch(path, flags) //如果是文件,则只监听文件
	}

	return nil
}

func (w *Watcher) AddWatch(path string, flags uint32) error {
	if w.isClose {
		log.Fatal("监控已经关闭")
	}

	//判断是否已经监控了
	if found := w.wm.find(path); found != nil {
		f := found.(watch)
		f.flags |= flags
		flags |= syscall.IN_MASK_ADD
	}

	wd, err := syscall.InotifyAddWatch(w.fd, path, flags)
	fmt.Println("开始监控目录:", path, wd)
	if wd == -1 {
		return err
	}
	w.wm.add(path, wd, flags)
	return nil

}

func (w *Watcher) RemoveWatch(path string) error {
	f, err := os.Lstat(path)

	if err != nil {
		return err
	}
	//如果起始监控的是目录,则删除已监控的所有目录
	if f.IsDir() {
		err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
			if info.IsDir() {
				w.rmWatch(path)
			}
			return nil
		})

		if err != nil {
			return err
		}
	}
	return nil
}

func (w *Watcher) rmWatch(path string) error {

	if found := w.wm.find(path); found != nil {
		success, errno := syscall.InotifyRmWatch(w.fd, uint32(found.(watch).wd))
		if success == -1 {
			return os.NewSyscallError("inotify_rm_watch", errno)
		}
		w.wm.remove(path)
		fmt.Println("移除监控目录:", path)
	} else {
		fmt.Println("没有找到对应的监控", path)
	}
	return nil

}

func (w *Watcher) Close() error {
	if w.isClose {
		return nil
	}
	w.isClose = true

	for path, _ := range w.wm.watcher {
		w.rmWatch(path)
	}

	w.done <- true

	return nil
}

func (w *Watcher) readEvent() {
	var (
		buf   [syscall.SizeofInotifyEvent * 4096]byte // Buffer for a maximum of 4096 raw events
		n     int                                     // Number of bytes read with read()
		errno error                                   // Syscall errno
	)

	for {

		select {
		case <-w.done:
			syscall.Close(w.fd)
			close(w.acceptEvent)
			close(w.Error)
			return
		default:
		}

		n, errno = syscall.Read(w.fd, buf[0:])

		if n == 0 {
			syscall.Close(w.fd)
			close(w.acceptEvent)
			close(w.Error)
			return
		}

		if n < syscall.SizeofInotifyEvent {
			log.Fatal("size of InotifyEvent error", errno)
		}

		var offset uint32 = 0

		for offset <= uint32(n-syscall.SizeofInotifyEvent) {
			raw := (*syscall.InotifyEvent)(unsafe.Pointer(&buf[offset]))
			event := new(FileEvent)
			event.wd = raw.Wd
			event.nameLen = raw.Len
			event.mask = raw.Mask
			event.cookie = raw.Cookie
			path := w.wm.paths[int(raw.Wd)]
			if raw.Len > 0 {
				// Point "bytes" at the first byte of the filename
				bytes := (*[syscall.PathMax]byte)(unsafe.Pointer(&buf[offset+syscall.SizeofInotifyEvent]))
				// The filename is padded with NUL bytes. TrimRight() gets rid of those.
				event.fileName = path + "/" + strings.TrimRight(string(bytes[0:raw.Len]), "\000")
			}

			if _, found := w.skipExt[filepath.Ext(event.fileName)]; !found {
				fmt.Println("--->", w.skipExt, "--->", filepath.Ext(event.fileName), "--->", found)
				//发送事件acceptEvent通道
				w.acceptEvent <- event
			} else {
				fmt.Println("过滤文件:", event.fileName)
			}

			offset += syscall.SizeofInotifyEvent + raw.Len
		}
	}
}

func (w *Watcher) purgeEvents() {
	for event := range w.acceptEvent {

		if (event.mask & sys_IN_OPEN) == sys_IN_OPEN {
			fmt.Println("sys_IN_OPEN is use")
		}

		if (event.mask & sys_IN_MOVE_SELF) == sys_IN_MOVE_SELF {
			fmt.Println("sys_IN_MOVE_SELF is use")
		}

		if (event.mask & sys_IN_MOVED_TO) == sys_IN_MOVED_TO {
			fmt.Println("sys_IN_MOVED_TO is use")
		}

		if (event.mask & sys_IN_MOVED_FROM) == sys_IN_MOVED_FROM {
			fmt.Println("sys_IN_MOVED_FROM is use")
		}

		if (event.mask & sys_IN_ACCESS) == sys_IN_ACCESS {
			fmt.Println("sys_in_access is use")
		}

		if (event.mask & sys_IN_ALL_EVENTS) == sys_IN_ALL_EVENTS {
			fmt.Println("sys_IN_ALL_EVENTS is use")
		}

		if (event.mask & sys_IN_ATTRIB) == sys_IN_ATTRIB {
			fmt.Println("sys_IN_ATTRIB is use")
		}

		if (event.mask & sys_IN_CLOSE) == sys_IN_CLOSE {
			fmt.Println("sys_IN_CLOSE is use")
		}

		// if (event.mask & sys_IN_CLOSE_NOWRITE) == sys_IN_CLOSE_NOWRITE {
		// 	fmt.Println("sys_IN_CLOSE_NOWRITE is use")
		// }

		if (event.mask & sys_IN_CLOSE_WRITE) == sys_IN_CLOSE_WRITE {
			fmt.Println("sys_IN_CLOSE_WRITE is use")
		}

		if (event.mask & sys_IN_CREATE) == sys_IN_CREATE {
			fmt.Println("sys_IN_CREATE is use")
		}

		if (event.mask & sys_IN_DELETE) == sys_IN_DELETE {
			fmt.Println("sys_IN_DELETE is use")
		}

		if (event.mask & sys_IN_DELETE_SELF) == sys_IN_DELETE_SELF {
			fmt.Println("sys_IN_DELETE_SELF is use")
		}

		if (event.mask & sys_IN_MODIFY) == sys_IN_MODIFY {
			fmt.Println("sys_IN_MODIFY is use")
		}

		if (event.mask & sys_IN_MOVE) == sys_IN_MOVE {
			fmt.Println("sys_IN_MOVE is use")
		}
		sendEvent := false
		switch {
		case event.IsCreate():
			if (event.mask & sys_IN_ISDIR) == sys_IN_ISDIR {
				w.Watch(event.fileName, sys_IN_ALL_EVENTS)
				fmt.Println("create a dir ", event.fileName)
			} else {
				fmt.Println("create a file ", event.fileName)
			}
			sendEvent = true
		case event.IsDelete():
			if (event.mask & sys_IN_ISDIR) == sys_IN_ISDIR {
				w.rmWatch(event.fileName)
				pathlen := len(event.fileName) - len(filepath.Base(event.fileName))
				event.fileName = event.fileName[0:pathlen]
			}
			fmt.Println("is delete ", event.fileName)
			event.fileName = filepath.Dir(event.fileName) + "/"
			sendEvent = true
		case event.IsModify():
			fmt.Println("is modify ", event.fileName)

			sendEvent = true
		case event.IsRename():
			fmt.Println("is rename ", event.fileName)
			sendEvent = true
		}

		if sendEvent {
			syncEvent <- event
		}

		//rsync -avz --password-file=/home/rsync.ps /work/golang/test/ nopsky@11.11.11.12::backup

	}
}
