package upload

import "sync"

type Progress struct {
	Total    int64 `json:"total"`
	Uploaded int64 `json:"uploaded"`
}

type Manager struct {
	mu          sync.RWMutex
	progresses  map[string]*Progress
	subscribers map[string]map[chan Progress]struct{}
}

func NewManager() *Manager {
	return &Manager{
		progresses:  make(map[string]*Progress),
		subscribers: make(map[string]map[chan Progress]struct{}),
	}
}

var DefaultManager = NewManager()

func (m *Manager) Start(id string, total int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.progresses[id] = &Progress{Total: total}
}

func (m *Manager) Update(id string, uploaded int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if p, ok := m.progresses[id]; ok {
		if uploaded > p.Uploaded {
			p.Uploaded = uploaded
		}
		for ch := range m.subscribers[id] {
			select {
			case ch <- *p:
			default:
			}
		}
	}
}

func (m *Manager) Finish(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.progresses, id)
	if subs, ok := m.subscribers[id]; ok {
		for ch := range subs {
			close(ch)
		}
		delete(m.subscribers, id)
	}
}

func (m *Manager) Get(id string) (Progress, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.progresses[id]
	if !ok {
		return Progress{}, false
	}
	return *p, true
}

func (m *Manager) Subscribe(id string) chan Progress {
	ch := make(chan Progress, 10)
	m.mu.Lock()
	if _, ok := m.subscribers[id]; !ok {
		m.subscribers[id] = make(map[chan Progress]struct{})
	}
	m.subscribers[id][ch] = struct{}{}
	if p, ok := m.progresses[id]; ok {
		ch <- *p
	}
	m.mu.Unlock()
	return ch
}

func (m *Manager) Unsubscribe(id string, ch chan Progress) {
	m.mu.Lock()
	if subs, ok := m.subscribers[id]; ok {
		if _, ok := subs[ch]; ok {
			delete(subs, ch)
			close(ch)
		}
		if len(subs) == 0 {
			delete(m.subscribers, id)
		}
	}
	m.mu.Unlock()
}
