package service

type ItemService struct {
	deps *Deps
}

func NewItemService(d *Deps) *ItemService {
	return &ItemService{deps: d}
}
