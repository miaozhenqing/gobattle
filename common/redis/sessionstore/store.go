package sessionstore

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type OnlineInfo struct {
	UserId    int64
	GatewayId int64
	SessionId string
	Version   int64
}

type KickNotice struct {
	UserId    string `json:"userId"`
	SessionId string `json:"sessionId"`
	Version   string `json:"version"`
}

type SetOnlineResult struct {
	NewSession OnlineInfo
	OldSession OnlineInfo
}

type Store struct {
	client *redis.Client
	ttl    time.Duration
}

func NewStore(client *redis.Client, ttl time.Duration) *Store {
	if client == nil || ttl <= 0 {
		panic("invalid session store config")
	}
	return &Store{client: client, ttl: ttl}
}

const setOnlineScript = `
	local oldGateway = redis.call('HGET', KEYS[1], 'gatewayId')
	local oldSession = redis.call('HGET', KEYS[1], 'sessionId')
	local oldVersion = redis.call('HGET', KEYS[1], 'version')
	local version = redis.call('INCR', KEYS[2])
	redis.call('HSET', KEYS[1], 'gatewayId', ARGV[1], 'sessionId', ARGV[2], 'version', version)
	redis.call('PEXPIRE', KEYS[1], ARGV[3])
	return {tostring(version), oldGateway or '', oldSession or '', oldVersion or ''}
`

const renewScript = `
	if redis.call('HGET', KEYS[1], 'gatewayId') ~= ARGV[1] or
	   redis.call('HGET', KEYS[1], 'sessionId') ~= ARGV[2] or
	   redis.call('HGET', KEYS[1], 'version') ~= ARGV[3] then
		return 0
	end
	return redis.call('PEXPIRE', KEYS[1], ARGV[4])
`

const deleteScript = `
	if redis.call('HGET', KEYS[1], 'gatewayId') ~= ARGV[1] or
	   redis.call('HGET', KEYS[1], 'sessionId') ~= ARGV[2] or
	   redis.call('HGET', KEYS[1], 'version') ~= ARGV[3] then
		return 0
	end
	return redis.call('DEL', KEYS[1])
`

func NewSessionId() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func OnlineKey(userId int64) string {
	return fmt.Sprintf("gw:online:{u:%d}", userId)
}

func VersionKey(userId int64) string {
	return fmt.Sprintf("gw:version:{u:%d}", userId)
}

func KickChannel(gatewayId int64) string {
	return fmt.Sprintf("gw:kick:%d", gatewayId)
}

func (s *Store) SetOnline(ctx context.Context, userId, gatewayId int64, sessionId string) (SetOnlineResult, error) {
	online := SetOnlineResult{NewSession: OnlineInfo{UserId: userId, GatewayId: gatewayId, SessionId: sessionId}}
	result, err := s.client.Eval(ctx,
		setOnlineScript,
		[]string{OnlineKey(userId), VersionKey(userId)},
		strconv.FormatInt(gatewayId, 10), sessionId, s.ttl.Milliseconds()).
		Slice()
	if err != nil {
		// 请求可能在 Redis 提交后丢失响应：只能通过本次 sessionId 读回确认，不重发写入。
		current, exists, readErr := s.GetOnline(ctx, userId)
		if readErr == nil && exists && current.GatewayId == gatewayId && current.SessionId == sessionId {
			online.NewSession = current
			return online, nil
		}
		return SetOnlineResult{}, err
	}
	if len(result) != 4 {
		return SetOnlineResult{}, fmt.Errorf("invalid set online result: %v", result)
	}
	online.NewSession.Version, err = strconv.ParseInt(fmt.Sprint(result[0]), 10, 64)
	if err != nil {
		return SetOnlineResult{}, err
	}
	if fmt.Sprint(result[1]) != "" {
		online.OldSession.UserId = userId
		online.OldSession.GatewayId, err = strconv.ParseInt(fmt.Sprint(result[1]), 10, 64)
		if err != nil {
			return SetOnlineResult{}, err
		}
		online.OldSession.SessionId = fmt.Sprint(result[2])
		online.OldSession.Version, err = strconv.ParseInt(fmt.Sprint(result[3]), 10, 64)
		if err != nil {
			return SetOnlineResult{}, err
		}
	}
	return online, nil
}

func (s *Store) PublishKickNotice(ctx context.Context, oldSession OnlineInfo) error {
	if oldSession.GatewayId == 0 || oldSession.SessionId == "" {
		return nil
	}
	payload, err := json.Marshal(KickNotice{
		UserId:    strconv.FormatInt(oldSession.UserId, 10),
		SessionId: oldSession.SessionId,
		Version:   strconv.FormatInt(oldSession.Version, 10),
	})
	if err != nil {
		return err
	}
	return s.client.Publish(ctx, KickChannel(oldSession.GatewayId), payload).Err()
}

func (s *Store) GetOnline(ctx context.Context, userId int64) (OnlineInfo, bool, error) {
	values, err := s.client.HGetAll(ctx, OnlineKey(userId)).Result()
	if err != nil {
		return OnlineInfo{}, false, err
	}
	if len(values) == 0 {
		return OnlineInfo{}, false, nil
	}
	gatewayId, err := strconv.ParseInt(values["gatewayId"], 10, 64)
	if err != nil {
		return OnlineInfo{}, false, err
	}
	version, err := strconv.ParseInt(values["version"], 10, 64)
	if err != nil {
		return OnlineInfo{}, false, err
	}
	if values["sessionId"] == "" {
		return OnlineInfo{}, false, fmt.Errorf("missing sessionId for user %d", userId)
	}
	return OnlineInfo{
		UserId:    userId,
		GatewayId: gatewayId,
		SessionId: values["sessionId"],
		Version:   version,
	}, true, nil
}

func (s *Store) RenewOnlineBatch(ctx context.Context, records []OnlineInfo) ([]bool, error) {
	if len(records) == 0 {
		return nil, nil
	}
	pipe := s.client.Pipeline()
	commands := make([]*redis.Cmd, len(records))
	for i, r := range records {
		commands[i] = pipe.Eval(ctx, renewScript, []string{OnlineKey(r.UserId)},
			strconv.FormatInt(r.GatewayId, 10), r.SessionId,
			strconv.FormatInt(r.Version, 10), s.ttl.Milliseconds())
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return nil, err
	}
	result := make([]bool, len(records))
	for i, cmd := range commands {
		n, err := cmd.Int64()
		if err != nil {
			return nil, err
		}
		result[i] = n == 1
	}
	return result, nil
}

func (s *Store) DeleteOnlineIfMatch(ctx context.Context, r OnlineInfo) error {
	return s.client.Eval(ctx, deleteScript, []string{OnlineKey(r.UserId)},
		strconv.FormatInt(r.GatewayId, 10), r.SessionId, strconv.FormatInt(r.Version, 10)).Err()
}
