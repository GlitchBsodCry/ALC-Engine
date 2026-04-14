package model

type AISession Session

func (AISession) TableName() string {
	return "sessions"
}
