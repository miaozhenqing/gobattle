package service

type ItemService struct {
	*Deps
}

func NewItemService(d *Deps) *ItemService {
	return &ItemService{Deps: d}
}
