package domain

import "gobattle/game/model"

type UserLogin struct {
	User    *model.User
	Created bool
}
