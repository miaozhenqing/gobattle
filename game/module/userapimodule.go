package module

import (
	"context"
	"gobattle/game/service"
	"gobattle/shared/api"
	"gobattle/shared/protocol"
	"unicode/utf8"
)

var _ api.UserApi = (*UserApiModule)(nil)

type UserApiModule struct {
	userService *service.UserService
}

func NewUserApiModule(userService *service.UserService) *UserApiModule {
	return &UserApiModule{userService: userService}
}

func (m *UserApiModule) Login(ctx context.Context, req *api.LoginReq) (*api.LoginResp, error) {
	if !validUsername(req.Username) {
		return nil, protocol.CodeInvalidUsername
	}
	res, err := m.userService.Login(ctx, req.Username)
	if err != nil {
		return nil, err
	}
	u := res.User
	return &api.LoginResp{UserId: u.UserId, Username: u.Username, RoleName: u.RoleName, IsNew: res.Created}, nil
}

func validUsername(s string) bool {
	if s == "" || !utf8.ValidString(s) {
		return false
	}
	return utf8.RuneCountInString(s) <= 32
}
