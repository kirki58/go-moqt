package data

import (
	"go-moq/pkg/model"
	"go-moq/pkg/session"
)

type Publisher struct {
	session *session.Session // Underlying, fully established MOQT session
	localTrackCatalog []model.MoqtFullTrackName // Servable tracks by this publisher
}

func NewPublisher(sess *session.Session, tracks ...model.MoqtFullTrackName) *Publisher{
	return &Publisher{
		session: sess,
		localTrackCatalog: tracks,
	}
}

