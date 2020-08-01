package client

import (
	"sync"

	log "github.com/sirupsen/logrus"
)

type Pusher struct {
	Key    string
	Path   string
	Source string
}

func NewPusher(key, path, id string) (pusher *Pusher) {
	pusher = &Pusher{
		Key:    key,
		Path:   path,
		Source: id,
	}
	return
}

type Server struct {
	Pushers        map[string]*Pusher // Path <-> Pusher
	PushersLock    sync.RWMutex
	AddPusherCh    chan *Pusher
	RemovePusherCh chan *Pusher
}

var Instance = &Server{
	Pushers:        make(map[string]*Pusher),
	AddPusherCh:    make(chan *Pusher),
	RemovePusherCh: make(chan *Pusher),
}

func GetServer() *Server {
	return Instance
}

func (server *Server) AddPusher(pusher *Pusher) bool {
	added := false
	server.PushersLock.Lock()
	_, ok := server.Pushers[pusher.Path]
	if !ok {
		server.Pushers[pusher.Path] = pusher
		log.Printf("%v start, now pusher size[%d]", pusher, len(server.Pushers))
		added = true
	} else {
		added = false
	}
	server.PushersLock.Unlock()
	if added {
		server.AddPusherCh <- pusher
	}
	return added
}

func (server *Server) RemovePusher(pusher *Pusher) {
	removed := false
	server.PushersLock.Lock()
	if _pusher, ok := server.Pushers[pusher.Path]; ok && pusher.Source == _pusher.Source {
		delete(server.Pushers, pusher.Path)
		log.Printf("%v end, now pusher size[%d]\n", pusher, len(server.Pushers))
		removed = true
	}
	server.PushersLock.Unlock()
	if removed {
		server.RemovePusherCh <- pusher
	}
}

func (server *Server) GetPusher(path string) (pusher *Pusher) {
	server.PushersLock.RLock()
	pusher = server.Pushers[path]
	server.PushersLock.RUnlock()
	return
}

func (server *Server) GetPushers() (pushers map[string]*Pusher) {
	pushers = make(map[string]*Pusher)
	server.PushersLock.RLock()
	for k, v := range server.Pushers {
		pushers[k] = v
	}
	server.PushersLock.RUnlock()
	return
}

func (server *Server) GetPusherSize() (size int) {
	server.PushersLock.RLock()
	size = len(server.Pushers)
	server.PushersLock.RUnlock()
	return
}
