package sfserver

import (
	"sync"
)

type watch struct {
	wd    int
	flags uint32
}

type WatchMap struct {
	lock    *sync.RWMutex    "结构读写锁"
	watcher map[string]watch "path->wd的map结构"
	paths   map[int]string   "wd->path的map结构"
}

func newWatchMap() *WatchMap {
	wm := &WatchMap{
		lock:    new(sync.RWMutex),
		watcher: make(map[string]watch),
		paths:   make(map[int]string),
	}
	return wm
}

func (wm *WatchMap) find(path string) interface{} {
	wm.lock.RLock()
	defer wm.lock.RUnlock()
	if w, found := wm.watcher[path]; found {
		return w
	}
	return nil
}

func (wm *WatchMap) add(path string, wd int, flags uint32) {
	wm.lock.Lock()
	defer wm.lock.Unlock()
	wm.watcher[path] = watch{wd, flags}
	wm.paths[wd] = path
	return
}

func (wm *WatchMap) remove(path string) {
	wm.lock.Lock()
	defer wm.lock.Unlock()
	if w, found := wm.watcher[path]; found {
		delete(wm.paths, w.wd)
	}
	delete(wm.watcher, path)

}

func (wm *WatchMap) update(path string, flags uint32) bool {
	wm.lock.Lock()
	defer wm.lock.Unlock()
	if w, found := wm.watcher[path]; found {
		w.flags |= flags
		return true
	} else {
		return false
	}
}

func (wm *WatchMap) getPath(wd int) (path string) {
	if p, found := wm.paths[wd]; found {
		path = p
	}
	return path
}
