package domain

type Session struct {
	sessionID string
	userID    string
}

func NewSession(token, userID string) *Session {
	return &Session{
		sessionID: token,
		userID:    userID,
	}
}

func (s *Session) SessionID() string {
	return s.sessionID
}

func (s *Session) UserID() string {
	return s.userID
}
