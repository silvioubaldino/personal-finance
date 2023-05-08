package session

import (
	"errors"
	"time"
)

type Control interface {
	Get(token string) (string, error)
	Set(token, uid string)
	Delete(token string)
}

type memorySession struct {
	sessions map[string]sessionData
}

type sessionData struct {
	uid      string
	expireAt time.Time
}

func NewControl() Control {
	return memorySession{
		sessions: make(map[string]sessionData),
	}
}

func (m memorySession) Get(token string) (string, error) {
	sessionData, ok := m.sessions[token]
	if !ok {
		return "", errors.New("session not found")
	}
	now := time.Now()
	if sessionData.expireAt.Before(now) {
		delete(m.sessions, token)
		return "", errors.New("session expired")
	}
	return sessionData.uid, nil
}

func (m memorySession) Set(token, uid string) {
	expireAt := time.Now().Add(time.Hour)
	newSession := sessionData{
		uid:      uid,
		expireAt: expireAt,
	}
	m.sessions[token] = newSession
}

func (m memorySession) Delete(token string) {
	delete(m.sessions, token)
}

func (m memorySession) ClearExpiredSessions() {
	for key, sessionData := range m.sessions {
		if sessionData.expireAt.After(time.Now()) {
			delete(m.sessions, key)
		}
	}
}
