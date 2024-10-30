package session

import (
	"errors"
	"sync"
	"time"
)

type Control interface {
	Get(token string) (string, error)
	Set(token, uid string)
	Delete(token string)
}

type memorySession struct {
	sessions map[string]sessionData
	mu       sync.RWMutex
}

type sessionData struct {
	uid      string
	expireAt time.Time
}

func NewControl() Control {
	return &memorySession{
		sessions: make(map[string]sessionData),
	}
}

func (m *memorySession) Get(token string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessionData, ok := m.sessions[token]
	if !ok {
		return "", errors.New("session not found")
	}
	now := time.Now()
	if sessionData.expireAt.Before(now) {
		m.mu.Lock()
		delete(m.sessions, token)
		m.mu.Unlock()
		return "", errors.New("session expired")
	}
	return sessionData.uid, nil
}

func (m *memorySession) Set(token, uid string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	expireAt := time.Now().Add(time.Hour)
	newSession := sessionData{
		uid:      uid,
		expireAt: expireAt,
	}
	m.sessions[token] = newSession
}

func (m *memorySession) Delete(token string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.sessions, token)
}

func (m *memorySession) ClearExpiredSessions() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for key, sessionData := range m.sessions {
		if sessionData.expireAt.Before(time.Now()) {
			delete(m.sessions, key)
		}
	}
}
