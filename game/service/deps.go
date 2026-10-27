package service

import "gobattle/game/repository"

type Deps struct {
	DB *repository.DB

	UserService *UserService
	ItemService *ItemService
}
