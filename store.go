package main

import (
	"gopkg.in/olahol/melody.v1"
	"sync"
)

type Routers struct {
	routers map[string]*MediaRouter
	sync.Mutex
}

var routers *Routers = new(Routers)

func (r *Routers) Get(routerId string) *MediaRouter {
	r.Lock()
	defer r.Unlock()
	return r.routers[routerId]
}

func (r *Routers) Add(router *MediaRouter) {
	r.Lock()
	defer r.Unlock()
	r.routers[router.routerID] = router
}

func (r *Routers) Remove(router *MediaRouter) {
	r.Lock()
	defer r.Unlock()
	delete(r.routers, router.routerID)
}

type SessionInfo struct {
	StreamID     string
	SubscriberID string
}

type Sessions struct {
	sessions map[*melody.Session]*SessionInfo
	sync.Mutex
}

var sessions *Sessions = new(Sessions)

func (s *Sessions) Add(session *melody.Session) {
	s.Lock()
	defer s.Unlock()
	s.sessions[session] = new(SessionInfo)
}

func (s *Sessions) Get(session *melody.Session) *SessionInfo {
	s.Lock()
	defer s.Unlock()
	return s.sessions[session]
}

func (s *Sessions) Remove(session *melody.Session) {
	s.Lock()
	defer s.Unlock()
	delete(s.sessions, session)
}