package module

import (
	"context"
	"fmt"
	"gobattle/framework/btrouter"
	"gobattle/game/domain"
	"gobattle/game/service"
	"gobattle/shared/api"
	"gobattle/shared/protocol"
	"strconv"
	"strings"
)

var _ api.GmApi = (*GmApiModule)(nil)

type GmApiModule struct {
	itemService *service.ItemService
	gmMap       map[string]gmFunc
}
type gmFunc func(ctx context.Context, args string) string

func NewGmApiModule(itemService *service.ItemService) *GmApiModule {
	m := &GmApiModule{
		itemService: itemService,
		gmMap:       make(map[string]gmFunc),
	}
	m.gmMap["GMItemAdd"] = m.gmItemAdd
	return m
}

func (m *GmApiModule) gmItemAdd(ctx context.Context, args string) string {
	split := strings.Split(args, "|")
	userId, _ := strconv.ParseInt(split[0], 10, 64)
	category, _ := strconv.Atoi(split[1])
	itemId, _ := strconv.Atoi(split[2])
	num, _ := strconv.ParseInt(split[3], 10, 64)

	_, err := m.itemService.Add(ctx, userId, []domain.ItemAmount{
		{ItemKey: domain.ItemKey{Category: domain.ItemCategory(category), ItemId: itemId}, Count: num},
	}, 0)
	if err != nil {
		return err.Error()
	}
	return "ok"
}

func GMRespFail(args ...any) (*protocol.GMResp, error) {
	var parts []string

	for _, arg := range args {
		parts = append(parts, fmt.Sprint(arg))
	}

	join := strings.Join(parts, " ")
	return &protocol.GMResp{Msg: join}, btrouter.CodeInvalidArgument
}
func GMRespSucc(args ...any) (*protocol.GMResp, error) {
	var parts []string

	for _, arg := range args {
		parts = append(parts, fmt.Sprint(arg))
	}

	join := strings.Join(parts, " ")
	return &protocol.GMResp{Msg: join}, nil
}

func (m *GmApiModule) Exec(ctx context.Context, req *protocol.GMReq) (*protocol.GMResp, error) {
	fn, ok := m.gmMap[req.Name]
	if !ok {
		return GMRespFail("no gm:", req.Name)
	}
	return GMRespSucc(fn(ctx, req.Args))
}
