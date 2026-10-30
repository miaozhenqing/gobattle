package tpl

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/emirpasic/gods/maps/treemap"
)

type TplManager struct {
	itemTplById      *treemap.Map
	matchModeTplById *treemap.Map
}

// todo 暂时读取系统配置json文件
func Load(dir string) (*TplManager, error) {
	m := &TplManager{}
	var err error
	if m.itemTplById, err = loadTable(dir, "item.json", func(t *ItemTpl) int { return t.Id }); err != nil {
		return nil, err
	}
	if m.matchModeTplById, err = loadTable(dir, "matchmode.json", func(t *MatchModeTpl) int { return t.Id }); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *TplManager) GetItemTpl(id int) (*ItemTpl, bool) {
	return getTpl[ItemTpl](m.itemTplById, id)
}

func (m *TplManager) GetMatchModeTpl(id int) (*MatchModeTpl, bool) {
	return getTpl[MatchModeTpl](m.matchModeTplById, id)
}

func loadTable[T any](dir, fileName string, idOf func(*T) int) (*treemap.Map, error) {
	data, err := os.ReadFile(filepath.Join(dir, fileName))
	if err != nil {
		return nil, fmt.Errorf("tpl %s read: %w", fileName, err)
	}
	var tpls []*T
	if err := json.Unmarshal(data, &tpls); err != nil {
		return nil, fmt.Errorf("tpl %s unmarshal: %w", fileName, err)
	}
	tplById := treemap.NewWithIntComparator()
	for _, t := range tpls {
		id := idOf(t)
		if _, dup := tplById.Get(id); dup {
			return nil, fmt.Errorf("tpl %s duplicate id %d", fileName, id)
		}
		tplById.Put(id, t)
	}
	return tplById, nil
}

func getTpl[T any](tplById *treemap.Map, id int) (*T, bool) {
	v, ok := tplById.Get(id)
	if !ok {
		return nil, false
	}
	return v.(*T), true
}
