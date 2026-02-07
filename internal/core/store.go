package core

import (
	"sync"
	"time"
)

type Store struct {
	mu            sync.Mutex
	clients       map[string]Client
	codes         map[string]AuthCode
	users         map[string]User
	refreshTokens map[string]RefreshToken
	revokedTokens map[string]RevokedToken
}

func NewStore() *Store {
	return &Store{
		clients:       make(map[string]Client),
		codes:         make(map[string]AuthCode),
		users:         make(map[string]User),
		refreshTokens: make(map[string]RefreshToken),
		revokedTokens: make(map[string]RevokedToken),
	}
}

func (s *Store) AddClient(c Client) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clients[c.ID] = c
}

func (s *Store) GetClient(id string) (Client, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.clients[id]
	return c, ok
}

func (s *Store) ListClients() []Client {
	s.mu.Lock()
	defer s.mu.Unlock()
	clients := make([]Client, 0, len(s.clients))
	for _, c := range s.clients {
		clients = append(clients, c)
	}
	return clients
}

func (s *Store) UpdateClient(c Client) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.clients[c.ID]; !exists {
		return false
	}
	s.clients[c.ID] = c
	return true
}

func (s *Store) DeleteClient(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.clients[id]; !exists {
		return false
	}
	delete(s.clients, id)
	return true
}

func (s *Store) SaveCode(ac AuthCode) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.codes[ac.Code] = ac
}

func (s *Store) ConsumeCode(code string) (AuthCode, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ac, ok := s.codes[code]
	if !ok {
		return AuthCode{}, false
	}
	if ac.Used || time.Now().After(ac.ExpiresAt) {
		return AuthCode{}, false
	}
	ac.Used = true
	s.codes[code] = ac
	return ac, true
}

func (s *Store) AddUser(u User) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.users[u.Email] = u
}

func (s *Store) GetUser(email string) (User, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.users[email]
	return u, ok
}

func (s *Store) ListUsers() []User {
	s.mu.Lock()
	defer s.mu.Unlock()
	users := make([]User, 0, len(s.users))
	for _, u := range s.users {
		users = append(users, u)
	}
	return users
}

func (s *Store) UpdateUser(u User) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.users[u.Email]; !exists {
		return false
	}
	s.users[u.Email] = u
	return true
}

func (s *Store) DeleteUser(email string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.users[email]; !exists {
		return false
	}
	delete(s.users, email)
	return true
}

func (s *Store) SaveRefreshToken(rt RefreshToken) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.refreshTokens[rt.Token] = rt
}

func (s *Store) GetRefreshToken(token string) (RefreshToken, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rt, ok := s.refreshTokens[token]
	if !ok || rt.Revoked || time.Now().After(rt.ExpiresAt) {
		return RefreshToken{}, false
	}
	return rt, true
}

func (s *Store) RevokeRefreshToken(token string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	rt, ok := s.refreshTokens[token]
	if !ok {
		return false
	}
	rt.Revoked = true
	s.refreshTokens[token] = rt
	return true
}

func (s *Store) RevokeAccessToken(tokenID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.revokedTokens[tokenID] = RevokedToken{
		Token:     tokenID,
		RevokedAt: time.Now(),
	}
}

func (s *Store) IsAccessTokenRevoked(tokenID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, revoked := s.revokedTokens[tokenID]
	return revoked
}

func (s *Store) RevokeRefreshTokensByUser(userID, clientID string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	count := 0
	for token, rt := range s.refreshTokens {
		if rt.UserID == userID && rt.ClientID == clientID && !rt.Revoked {
			rt.Revoked = true
			s.refreshTokens[token] = rt
			count++
		}
	}
	return count
}
