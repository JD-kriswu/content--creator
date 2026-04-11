// backend/internal/service/feishu_session.go
package service

import "sync"

type FeishuState string

const (
	FeishuIdle      FeishuState = "idle"
	FeishuAnalyzing FeishuState = "analyzing"
	FeishuAwaiting  FeishuState = "awaiting"
	FeishuWriting   FeishuState = "writing"
)

type FeishuSession struct {
	ChatID     string
	BotID      uint
	UserID     uint
	FeishuUID  uint
	ConvID     uint
	WorkflowID uint
	State      FeishuState
	lock       sync.Mutex
}

type FeishuSessionMgr struct {
	sessions map[string]*FeishuSession
	mu       sync.RWMutex
}

var feishuSessionMgr *FeishuSessionMgr
var feishuSessionOnce sync.Once

func GetFeishuSessionMgr() *FeishuSessionMgr {
	feishuSessionOnce.Do(func() {
		feishuSessionMgr = &FeishuSessionMgr{
			sessions: make(map[string]*FeishuSession),
		}
	})
	return feishuSessionMgr
}

func (m *FeishuSessionMgr) GetOrCreate(chatID string, botID, userID, feishuUID uint) *FeishuSession {
	m.mu.Lock()
	defer m.mu.Unlock()

	if sess, ok := m.sessions[chatID]; ok {
		return sess
	}
	sess := &FeishuSession{
		ChatID:    chatID,
		BotID:     botID,
		UserID:    userID,
		FeishuUID: feishuUID,
		State:     FeishuIdle,
	}
	m.sessions[chatID] = sess
	return sess
}

func (m *FeishuSessionMgr) Get(chatID string) *FeishuSession {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sessions[chatID]
}

func (m *FeishuSessionMgr) SetState(chatID string, state FeishuState) {
	m.mu.RLock()
	sess := m.sessions[chatID]
	m.mu.RUnlock()
	if sess != nil {
		sess.lock.Lock()
		sess.State = state
		sess.lock.Unlock()
	}
}

func (m *FeishuSessionMgr) SetWorkflowID(chatID string, wfID uint) {
	m.mu.RLock()
	sess := m.sessions[chatID]
	m.mu.RUnlock()
	if sess != nil {
		sess.lock.Lock()
		sess.WorkflowID = wfID
		sess.lock.Unlock()
	}
}

func (m *FeishuSessionMgr) SetConvID(chatID string, convID uint) {
	m.mu.RLock()
	sess := m.sessions[chatID]
	m.mu.RUnlock()
	if sess != nil {
		sess.lock.Lock()
		sess.ConvID = convID
		sess.lock.Unlock()
	}
}

func (m *FeishuSessionMgr) IsBusy(chatID string) bool {
	m.mu.RLock()
	sess := m.sessions[chatID]
	m.mu.RUnlock()
	if sess == nil {
		return false
	}
	sess.lock.Lock()
	defer sess.lock.Unlock()
	return sess.State == FeishuAnalyzing || sess.State == FeishuWriting
}

func (m *FeishuSessionMgr) Clear(chatID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, chatID)
}