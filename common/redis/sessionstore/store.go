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

type PushNotice struct {
	UserId    int64  `json:"userId"`
	SessionId string `json:"sessionId"`
	Version   int64  `json:"version"`
	MsgId     uint32 `json:"msgId"`
	Payload   []byte `json:"payload"`
}

type PushTarget struct {
	UserId  int64
	MsgId   uint32
	Payload []byte
}

type SetOnlineResult struct {
	NewSession OnlineInfo
	OldSession OnlineInfo
}

type OnlineReader struct {
	client *redis.Client
}

type Store struct {
	*OnlineReader
	ttl time.Duration
}

func NewOnlineReader(client *redis.Client) *OnlineReader {
	if client == nil {
		panic("invalid online reader config")
	}
	return &OnlineReader{client: client}
}

func NewStore(client *redis.Client, ttl time.Duration) *Store {
	if client == nil || ttl <= 0 {
		panic("invalid session store config")
	}
	return &Store{OnlineReader: NewOnlineReader(client), ttl: ttl}
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

func PushChannel(gatewayId int64) string {
	return fmt.Sprintf("gw:push:%d", gatewayId)
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

func parseOnlineInfo(userId int64, values map[string]string) (OnlineInfo, bool, error) {
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
	if values["sessionId"] == "" || gatewayId <= 0 || version <= 0 {
		return OnlineInfo{}, false, fmt.Errorf("invalid online info for user %d", userId)
	}
	return OnlineInfo{UserId: userId, GatewayId: gatewayId, SessionId: values["sessionId"], Version: version}, true, nil
}

func (s *OnlineReader) GetOnline(ctx context.Context, userId int64) (OnlineInfo, bool, error) {
	values, err := s.client.HGetAll(ctx, OnlineKey(userId)).Result()
	if err != nil {
		return OnlineInfo{}, false, err
	}
	return parseOnlineInfo(userId, values)
}

func (s *OnlineReader) GetOnlineBatch(ctx context.Context, userIds []int64) (map[int64]OnlineInfo, error) {
	result := make(map[int64]OnlineInfo, len(userIds))
	uniqueUserIds := make([]int64, 0, len(userIds))
	seen := make(map[int64]struct{}, len(userIds))
	for _, userId := range userIds {
		if userId <= 0 {
			return nil, fmt.Errorf("invalid userId %d", userId)
		}
		if _, ok := seen[userId]; ok {
			continue
		}
		seen[userId] = struct{}{}
		uniqueUserIds = append(uniqueUserIds, userId)
	}
	if len(uniqueUserIds) == 0 {
		return result, nil
	}
	pipe := s.client.Pipeline()
	commands := make([]*redis.MapStringStringCmd, len(uniqueUserIds))
	for i, userId := range uniqueUserIds {
		commands[i] = pipe.HGetAll(ctx, OnlineKey(userId))
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return nil, err
	}
	for i, cmd := range commands {
		info, exists, err := parseOnlineInfo(uniqueUserIds[i], cmd.Val())
		if err != nil {
			return nil, err
		}
		if exists {
			result[uniqueUserIds[i]] = info
		}
	}
	return result, nil
}

func (s *OnlineReader) PublishPushBatch(ctx context.Context, targets []PushTarget) (int, error) {
	for _, target := range targets {
		if target.UserId <= 0 {
			return 0, fmt.Errorf("invalid push userId %d", target.UserId)
		}
	}
	count := 0
	for start := 0; start < len(targets); start += 300 {
		end := min(start+300, len(targets))
		n, err := s.publishPushBatch(ctx, targets[start:end])
		count += n
		if err != nil {
			return count, err
		}
	}
	return count, nil
}

func (s *OnlineReader) publishPushBatch(ctx context.Context, targets []PushTarget) (int, error) {
	if len(targets) == 0 {
		return 0, nil
	}
	userIds := make([]int64, 0, len(targets))
	seen := make(map[int64]struct{}, len(targets))
	for _, target := range targets {
		if target.UserId <= 0 {
			return 0, fmt.Errorf("invalid push userId %d", target.UserId)
		}
		if _, ok := seen[target.UserId]; !ok {
			seen[target.UserId] = struct{}{}
			userIds = append(userIds, target.UserId)
		}
	}
	onlineByUser, err := s.GetOnlineBatch(ctx, userIds)
	if err != nil {
		return 0, err
	}
	noticesByGateway := make(map[int64][]PushNotice)
	onlineCount := 0
	for _, target := range targets {
		online, ok := onlineByUser[target.UserId]
		if !ok {
			continue
		}
		onlineCount++
		noticesByGateway[online.GatewayId] = append(noticesByGateway[online.GatewayId], PushNotice{
			UserId: target.UserId, SessionId: online.SessionId, Version: online.Version,
			MsgId: target.MsgId, Payload: target.Payload,
		})
	}
	if len(noticesByGateway) == 0 {
		return 0, nil
	}
	pipe := s.client.Pipeline()
	publishesByGateway := make(map[int64]*redis.IntCmd, len(noticesByGateway))
	for gatewayId, notices := range noticesByGateway {
		payload, err := json.Marshal(notices)
		if err != nil {
			return 0, err
		}
		publishesByGateway[gatewayId] = pipe.Publish(ctx, PushChannel(gatewayId), payload)
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return 0, err
	}
	for gatewayId, cmd := range publishesByGateway {
		if cmd.Val() == 0 {
			return onlineCount, fmt.Errorf("gateway %d has no push subscriber", gatewayId)
		}
	}
	return onlineCount, nil
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
