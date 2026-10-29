package service

import "gobattle/game/repository"

type ItemService struct {
	db *repository.DB
}

func NewItemService(db *repository.DB) *ItemService {
	return &ItemService{db: db}
}
