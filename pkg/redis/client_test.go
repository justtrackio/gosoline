package redis_test

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/elliotchance/redismock/v9"
	"github.com/justtrackio/gosoline/pkg/cfg"
	cfgMocks "github.com/justtrackio/gosoline/pkg/cfg/mocks"
	"github.com/justtrackio/gosoline/pkg/exec"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/redis"
	baseRedis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type ClientWithMiniRedisTestSuite struct {
	suite.Suite

	config     *cfgMocks.Config
	logger     logMocks.LoggerMock
	settings   *redis.Settings
	server     *miniredis.Miniredis
	baseClient *baseRedis.Client
	client     redis.Client
}

func (s *ClientWithMiniRedisTestSuite) SetupTest() {
	server, err := miniredis.Run()
	if err != nil {
		s.FailNow(err.Error(), "can not start miniredis")

		return
	}

	s.config = new(cfgMocks.Config)
	s.settings = &redis.Settings{}
	s.logger = logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(s.T()))
	executor := exec.NewDefaultExecutor()

	s.baseClient = baseRedis.NewClient(&baseRedis.Options{
		Addr: server.Addr(),
	})

	s.server = server
	s.client = redis.NewClientWithInterfaces(s.logger, s.baseClient, executor, s.settings, "")
}

func (s *ClientWithMiniRedisTestSuite) TestNewClientWithSettings_InvalidPattern_MissingKey() {
	s.settings.Naming.KeyPattern = "prefix-{app.env}" // Missing {key}

	client, err := redis.NewClientWithSettings(s.T().Context(), s.config, s.logger, s.settings)

	s.Error(err)
	s.Nil(client)
	s.Contains(err.Error(), "must end with {key}")
}

func (s *ClientWithMiniRedisTestSuite) TestNewClientWithSettings_InvalidPattern_KeyNotAtEnd() {
	s.settings.Naming.KeyPattern = "prefix-{key}-suffix"

	client, err := redis.NewClientWithSettings(s.T().Context(), s.config, s.logger, s.settings)

	s.Error(err)
	s.Nil(client)
	s.Contains(err.Error(), "must end with {key}")
}

func (s *ClientWithMiniRedisTestSuite) TestNewClientWithSettings_FormatError() {
	s.settings.Naming.KeyPattern = "{app.missing}-{key}"
	s.settings.Identity = cfg.Identity{
		Name: "app",
		Env:  "env",
	}

	// Mock FormatString to return error
	s.config.EXPECT().FormatString("{app.missing}-", mock.Anything).Return("", assert.AnError)

	client, err := redis.NewClientWithSettings(s.T().Context(), s.config, s.logger, s.settings)

	s.Error(err)
	s.Nil(client)
	s.Contains(err.Error(), "redis key naming failed")
}

func (s *ClientWithMiniRedisTestSuite) newPrefixedClient() redis.Client {
	executor := exec.NewDefaultExecutor()

	return redis.NewClientWithInterfaces(s.logger, s.baseClient, executor, s.settings, "my-prefix-")
}

func (s *ClientWithMiniRedisTestSuite) TestClient_Prefixing() {
	client := s.newPrefixedClient()
	ctx := s.T().Context()

	// Test Get — set raw prefixed key, read via client
	err := s.server.Set("my-prefix-foo", "bar")
	s.NoError(err)

	val, err := client.Get(ctx, "foo")
	s.NoError(err)
	s.Equal("bar", val)

	// Test Set — write via client, verify raw prefixed key
	err = client.Set(ctx, "key", "value", time.Minute)
	s.NoError(err)

	val, err = s.server.Get("my-prefix-key")
	s.NoError(err)
	s.Equal("value", val)

	// Test Exists (bug fix validation — Exists now uses executePrefixed)
	err = client.Set(ctx, "exists-key", "v", 0)
	s.NoError(err)

	count, err := client.Exists(ctx, "exists-key")
	s.NoError(err)
	s.Equal(int64(1), count)

	// Verify the key is stored with prefix
	s.True(s.server.Exists("my-prefix-exists-key"))
	// Non-prefixed key should not exist
	s.False(s.server.Exists("exists-key"))

	// Test Del with prefix
	delCount, err := client.Del(ctx, "exists-key")
	s.NoError(err)
	s.Equal(int64(1), delCount)
	s.False(s.server.Exists("my-prefix-exists-key"))

	// Test SetNX
	ok, err := client.SetNX(ctx, "nx-key", "first", time.Minute)
	s.NoError(err)
	s.True(ok)

	ok, err = client.SetNX(ctx, "nx-key", "second", time.Minute)
	s.NoError(err)
	s.False(ok)

	val, err = s.server.Get("my-prefix-nx-key")
	s.NoError(err)
	s.Equal("first", val)

	// Test Expire with prefix
	err = client.Set(ctx, "exp-key", "v", 0)
	s.NoError(err)

	result, err := client.Expire(ctx, "exp-key", time.Second)
	s.NoError(err)
	s.True(result)

	s.server.FastForward(time.Second)

	count, err = client.Exists(ctx, "exp-key")
	s.NoError(err)
	s.Equal(int64(0), count)

	// Test Incr/Decr with prefix
	incrVal, err := client.Incr(ctx, "counter")
	s.NoError(err)
	s.Equal(int64(1), incrVal)

	incrVal, err = client.IncrBy(ctx, "counter", 5)
	s.NoError(err)
	s.Equal(int64(6), incrVal)

	decrVal, err := client.Decr(ctx, "counter")
	s.NoError(err)
	s.Equal(int64(5), decrVal)

	decrVal, err = client.DecrBy(ctx, "counter", 3)
	s.NoError(err)
	s.Equal(int64(2), decrVal)

	// Verify stored under prefixed key
	rawVal, err := s.server.Get("my-prefix-counter")
	s.NoError(err)
	s.Equal("2", rawVal)

	// Test GetDel with prefix
	err = client.Set(ctx, "getdel-key", "getdel-val", 0)
	s.NoError(err)

	gdVal, err := client.GetDel(ctx, "getdel-key")
	s.NoError(err)
	s.Equal("getdel-val", gdVal)

	_, err = client.GetDel(ctx, "getdel-key")
	s.Equal(redis.Nil, err)

	// Test GetSet with prefix
	gsVal, err := client.GetSet(ctx, "getset-key", "v1")
	s.Equal(redis.Nil, err)
	s.Equal("", gsVal)

	gsVal, err = client.GetSet(ctx, "getset-key", "v2")
	s.NoError(err)
	s.Equal("v1", gsVal)

	rawVal, err = s.server.Get("my-prefix-getset-key")
	s.NoError(err)
	s.Equal("v2", rawVal)
}

func (s *ClientWithMiniRedisTestSuite) TestClient_Prefixing_HashOps() {
	client := s.newPrefixedClient()
	ctx := s.T().Context()

	// HSet
	err := client.HSet(ctx, "hash", "field1", "val1")
	s.NoError(err)

	// Verify stored under prefixed key
	rawVal := s.server.HGet("my-prefix-hash", "field1")
	s.Equal("val1", rawVal)

	// HGet
	val, err := client.HGet(ctx, "hash", "field1")
	s.NoError(err)
	s.Equal("val1", val)

	// HExists
	exists, err := client.HExists(ctx, "hash", "field1")
	s.NoError(err)
	s.True(exists)

	exists, err = client.HExists(ctx, "hash", "nonexistent")
	s.NoError(err)
	s.False(exists)

	// HSetNX
	isNew, err := client.HSetNX(ctx, "hash", "field2", "val2")
	s.NoError(err)
	s.True(isNew)

	isNew, err = client.HSetNX(ctx, "hash", "field2", "val2-dup")
	s.NoError(err)
	s.False(isNew)

	// HKeys
	keys, err := client.HKeys(ctx, "hash")
	s.NoError(err)
	s.ElementsMatch([]string{"field1", "field2"}, keys)

	// HMSet
	err = client.HMSet(ctx, "hash", map[string]any{"field3": "val3", "field4": "val4"})
	s.NoError(err)

	// HMGet
	vals, err := client.HMGet(ctx, "hash", "field1", "field3", "nonexistent")
	s.NoError(err)
	s.Equal([]any{"val1", "val3", nil}, vals)

	// HGetAll
	allVals, err := client.HGetAll(ctx, "hash")
	s.NoError(err)
	s.Equal(map[string]string{
		"field1": "val1",
		"field2": "val2",
		"field3": "val3",
		"field4": "val4",
	}, allVals)

	// HDel
	delCount, err := client.HDel(ctx, "hash", "field4")
	s.NoError(err)
	s.Equal(int64(1), delCount)

	// Verify key is not stored without prefix
	s.False(s.server.Exists("hash"))
}

func (s *ClientWithMiniRedisTestSuite) TestClient_Prefixing_MSet_MGet() {
	client := s.newPrefixedClient()
	ctx := s.T().Context()

	// MSet — validates the variadic fix (pairs... spread)
	err := client.MSet(ctx, "k1", "v1", "k2", "v2", "k3", "v3")
	s.NoError(err)

	// Verify all keys stored with prefix
	rawVal, err := s.server.Get("my-prefix-k1")
	s.NoError(err)
	s.Equal("v1", rawVal)

	rawVal, err = s.server.Get("my-prefix-k2")
	s.NoError(err)
	s.Equal("v2", rawVal)

	rawVal, err = s.server.Get("my-prefix-k3")
	s.NoError(err)
	s.Equal("v3", rawVal)

	// Verify non-prefixed keys don't exist
	s.False(s.server.Exists("k1"))
	s.False(s.server.Exists("k2"))

	// MGet
	vals, err := client.MGet(ctx, "k1", "k2", "k3", "nonexistent")
	s.NoError(err)
	s.Equal([]any{"v1", "v2", "v3", nil}, vals)
}

func (s *ClientWithMiniRedisTestSuite) TestClient_Prefixing_ListOps() {
	client := s.newPrefixedClient()
	ctx := s.T().Context()

	// LPush/RPush
	count, err := client.LPush(ctx, "list", "a", "b")
	s.NoError(err)
	s.Equal(int64(2), count)

	count, err = client.RPush(ctx, "list", "c")
	s.NoError(err)
	s.Equal(int64(3), count)

	// Verify stored under prefixed key
	s.True(s.server.Exists("my-prefix-list"))
	s.False(s.server.Exists("list"))

	// LLen
	length, err := client.LLen(ctx, "list")
	s.NoError(err)
	s.Equal(int64(3), length)

	// LPop
	val, err := client.LPop(ctx, "list")
	s.NoError(err)
	s.Equal("b", val)

	// RPop
	val, err = client.RPop(ctx, "list")
	s.NoError(err)
	s.Equal("c", val)

	// LRem
	_, err = client.LPush(ctx, "list2", "x", "x", "y")
	s.NoError(err)
	removed, err := client.LRem(ctx, "list2", 0, "x")
	s.NoError(err)
	s.Equal(int64(2), removed)
}

func (s *ClientWithMiniRedisTestSuite) TestClient_Prefixing_SetOps() {
	client := s.newPrefixedClient()
	ctx := s.T().Context()

	// SAdd
	_, err := client.SAdd(ctx, "set1", "a", "b", "c", "d")
	s.NoError(err)
	_, err = client.SAdd(ctx, "set2", "b", "c")
	s.NoError(err)
	_, err = client.SAdd(ctx, "set3", "c", "d")
	s.NoError(err)

	// Verify stored under prefixed keys
	s.True(s.server.Exists("my-prefix-set1"))
	s.False(s.server.Exists("set1"))

	// SCard
	card, err := client.SCard(ctx, "set1")
	s.NoError(err)
	s.Equal(int64(4), card)

	// SMembers
	members, err := client.SMembers(ctx, "set1")
	s.NoError(err)
	s.ElementsMatch([]string{"a", "b", "c", "d"}, members)

	// SIsMember
	isMember, err := client.SIsMember(ctx, "set1", "a")
	s.NoError(err)
	s.True(isMember)

	isMember, err = client.SIsMember(ctx, "set1", "z")
	s.NoError(err)
	s.False(isMember)

	// SDiff with prefix
	diffs, err := client.SDiff(ctx, "set1", "set2", "set3")
	s.NoError(err)
	s.ElementsMatch([]string{"a"}, diffs)

	// SInter with prefix
	inters, err := client.SInter(ctx, "set1", "set2", "set3")
	s.NoError(err)
	s.ElementsMatch([]string{"c"}, inters)

	// SUnion with prefix
	union, err := client.SUnion(ctx, "set1", "set2", "set3")
	s.NoError(err)
	s.ElementsMatch([]string{"a", "b", "c", "d"}, union)

	// SDiffStore
	numDiff, err := client.SDiffStore(ctx, "diff-dest", "set1", "set2", "set3")
	s.NoError(err)
	s.Equal(int64(1), numDiff)
	s.True(s.server.Exists("my-prefix-diff-dest"))

	// SInterStore
	numInter, err := client.SInterStore(ctx, "inter-dest", "set1", "set2", "set3")
	s.NoError(err)
	s.Equal(int64(1), numInter)
	s.True(s.server.Exists("my-prefix-inter-dest"))

	// SUnionStore
	numUnion, err := client.SUnionStore(ctx, "union-dest", "set1", "set2", "set3")
	s.NoError(err)
	s.Equal(int64(4), numUnion)
	s.True(s.server.Exists("my-prefix-union-dest"))

	// SMove
	_, err = client.SAdd(ctx, "smove-src", "x", "y")
	s.NoError(err)
	moved, err := client.SMove(ctx, "smove-src", "smove-dst", "x")
	s.NoError(err)
	s.True(moved)
	s.True(s.server.Exists("my-prefix-smove-dst"))

	// SPop
	_, err = client.SAdd(ctx, "spop-set", "one")
	s.NoError(err)
	poppedVal, err := client.SPop(ctx, "spop-set")
	s.NoError(err)
	s.Equal("one", poppedVal)

	// SRem
	_, err = client.SAdd(ctx, "srem-set", "a", "b", "c")
	s.NoError(err)
	numRemoved, err := client.SRem(ctx, "srem-set", "a", "c")
	s.NoError(err)
	s.Equal(int64(2), numRemoved)

	// SRandMember
	_, err = client.SAdd(ctx, "srand-set", "m1", "m2")
	s.NoError(err)
	randMember, err := client.SRandMember(ctx, "srand-set")
	s.NoError(err)
	s.Contains([]string{"m1", "m2"}, randMember)
}

func (s *ClientWithMiniRedisTestSuite) TestClient_Prefixing_SortedSets() {
	client := s.newPrefixedClient()
	ctx := s.T().Context()

	// ZAdd
	inserts, err := client.ZAdd(ctx, "zset", 10.0, "m0")
	s.NoError(err)
	s.Equal(int64(1), inserts)

	// Verify stored under prefixed key
	s.True(s.server.Exists("my-prefix-zset"))
	s.False(s.server.Exists("zset"))

	// ZAddArgs
	args := redis.ZAddArgs{
		Key: "zset",
		Members: []redis.Z{
			{Member: "m1", Score: 20},
			{Member: "m2", Score: 30},
			{Member: "m3", Score: 40},
		},
	}
	inserts, err = client.ZAddArgs(ctx, args)
	s.NoError(err)
	s.Equal(int64(3), inserts)

	// ZAddArgsIncr
	incrArgs := redis.ZAddArgs{
		Key:     "zset",
		Members: []redis.Z{{Member: "m0", Score: 5}},
	}
	newScore, err := client.ZAddArgsIncr(ctx, incrArgs)
	s.NoError(err)
	s.Equal(15.0, newScore) // 10 + 5

	// ZCard
	card, err := client.ZCard(ctx, "zset")
	s.NoError(err)
	s.Equal(int64(4), card)

	// ZCount
	count, err := client.ZCount(ctx, "zset", "15", "35")
	s.NoError(err)
	s.Equal(int64(3), count) // m0=15, m1=20, m2=30

	// ZIncrBy
	newScore, err = client.ZIncrBy(ctx, "zset", 100, "m0")
	s.NoError(err)
	s.Equal(115.0, newScore)

	// ZScore
	score, err := client.ZScore(ctx, "zset", "m1")
	s.NoError(err)
	s.Equal(20.0, score)

	// ZRange
	members, err := client.ZRange(ctx, "zset", 0, 1)
	s.NoError(err)
	s.Equal([]string{"m1", "m2"}, members)

	// ZRangeArgs
	rangeArgs := redis.ZRangeArgs{
		Key:     "zset",
		Start:   "20",
		Stop:    "40",
		ByScore: true,
	}
	members, err = client.ZRangeArgs(ctx, rangeArgs)
	s.NoError(err)
	s.Equal([]string{"m1", "m2", "m3"}, members)

	// ZRangeArgsWithScore
	zs, err := client.ZRangeArgsWithScore(ctx, redis.ZRangeArgs{
		Key:   "zset",
		Start: 0,
		Stop:  0,
	})
	s.NoError(err)
	s.Len(zs, 1)
	s.Equal("m1", zs[0].Member)
	s.Equal(20.0, zs[0].Score)

	// ZRandMember
	randMembers, err := client.ZRandMember(ctx, "zset", 2)
	s.NoError(err)
	s.Len(randMembers, 2)

	// ZRank
	rank, err := client.ZRank(ctx, "zset", "m1")
	s.NoError(err)
	s.Equal(int64(0), rank) // m1 has lowest score after m0 was incr'd to 115

	// ZRevRank
	revRank, err := client.ZRevRank(ctx, "zset", "m0")
	s.NoError(err)
	s.Equal(int64(0), revRank) // m0=115 is highest

	// ZRem
	removed, err := client.ZRem(ctx, "zset", "m3")
	s.NoError(err)
	s.Equal(int64(1), removed)

	card, err = client.ZCard(ctx, "zset")
	s.NoError(err)
	s.Equal(int64(3), card)
}

func (s *ClientWithMiniRedisTestSuite) TestClient_Prefixing_Pipeline_KeyValue() {
	client := s.newPrefixedClient()
	ctx := s.T().Context()

	// Pipeline Set + Get + Exists + Del + Incr + Expire + TTL
	pipe := client.Pipeline()

	pipe.Set(ctx, "pk1", "pv1", 0)
	pipe.Set(ctx, "pk2", "pv2", 0)
	getCmd := pipe.Get(ctx, "pk1")
	existsCmd := pipe.Exists(ctx, "pk1", "pk2")
	incrCmd := pipe.Incr(ctx, "pcounter")

	_, err := pipe.Exec(ctx)
	s.NoError(err)

	s.Equal("pv1", getCmd.Val())
	s.Equal(int64(2), existsCmd.Val())
	s.Equal(int64(1), incrCmd.Val())

	// Verify keys stored with prefix in raw miniredis
	s.True(s.server.Exists("my-prefix-pk1"))
	s.True(s.server.Exists("my-prefix-pk2"))
	s.True(s.server.Exists("my-prefix-pcounter"))
	s.False(s.server.Exists("pk1"))
	s.False(s.server.Exists("pk2"))
	s.False(s.server.Exists("pcounter"))

	// Pipeline Del
	pipe2 := client.Pipeline()
	delCmd := pipe2.Del(ctx, "pk1", "pk2")
	_, err = pipe2.Exec(ctx)
	s.NoError(err)
	s.Equal(int64(2), delCmd.Val())
	s.False(s.server.Exists("my-prefix-pk1"))

	// Pipeline SetNX
	pipe3 := client.Pipeline()
	setNxCmd := pipe3.SetNX(ctx, "nx-pipe", "v1", time.Minute)
	_, err = pipe3.Exec(ctx)
	s.NoError(err)
	s.True(setNxCmd.Val())

	rawVal, err := s.server.Get("my-prefix-nx-pipe")
	s.NoError(err)
	s.Equal("v1", rawVal)

	// Pipeline MSet + MGet
	pipe4 := client.Pipeline()
	pipe4.MSet(ctx, "mk1", "mv1", "mk2", "mv2")
	mgetCmd := pipe4.MGet(ctx, "mk1", "mk2")
	_, err = pipe4.Exec(ctx)
	s.NoError(err)

	s.True(s.server.Exists("my-prefix-mk1"))
	s.True(s.server.Exists("my-prefix-mk2"))
	s.Equal([]any{"mv1", "mv2"}, mgetCmd.Val())

	// Pipeline Expire + TTL + ExpireNX
	err = s.server.Set("my-prefix-ttl-key", "x")
	s.NoError(err)

	pipe5 := client.Pipeline()
	expireCmd := pipe5.Expire(ctx, "ttl-key", 10*time.Second)
	ttlCmd := pipe5.TTL(ctx, "ttl-key")
	_, err = pipe5.Exec(ctx)
	s.NoError(err)
	s.True(expireCmd.Val())
	s.True(ttlCmd.Val() > 0)

	// Pipeline IncrBy / DecrBy
	pipe6 := client.Pipeline()
	incrByCmd := pipe6.IncrBy(ctx, "pcounter", 10)
	decrCmd := pipe6.Decr(ctx, "pcounter")
	decrByCmd := pipe6.DecrBy(ctx, "pcounter", 3)
	_, err = pipe6.Exec(ctx)
	s.NoError(err)
	s.Equal(int64(11), incrByCmd.Val()) // 1 + 10
	s.Equal(int64(10), decrCmd.Val())
	s.Equal(int64(7), decrByCmd.Val())

	// Pipeline GetDel + GetSet
	err = s.server.Set("my-prefix-gd-pipe", "gd-val")
	s.NoError(err)

	pipe7 := client.Pipeline()
	getDelCmd := pipe7.GetDel(ctx, "gd-pipe")
	_, err = pipe7.Exec(ctx)
	s.NoError(err)
	s.Equal("gd-val", getDelCmd.Val())
	s.False(s.server.Exists("my-prefix-gd-pipe"))

	pipe8 := client.Pipeline()
	pipe8.Set(ctx, "gs-pipe", "old", 0)
	getSetCmd := pipe8.GetSet(ctx, "gs-pipe", "new")
	_, err = pipe8.Exec(ctx)
	s.NoError(err)
	s.Equal("old", getSetCmd.Val())
	rawVal, err = s.server.Get("my-prefix-gs-pipe")
	s.NoError(err)
	s.Equal("new", rawVal)
}

func (s *ClientWithMiniRedisTestSuite) TestClient_Prefixing_Pipeline_Lists() {
	client := s.newPrefixedClient()
	ctx := s.T().Context()

	pipe := client.Pipeline()
	lpushCmd := pipe.LPush(ctx, "plist", "a", "b")
	rpushCmd := pipe.RPush(ctx, "plist", "c")
	llenCmd := pipe.LLen(ctx, "plist")
	_, err := pipe.Exec(ctx)
	s.NoError(err)

	s.Equal(int64(2), lpushCmd.Val())
	s.Equal(int64(3), rpushCmd.Val())
	s.Equal(int64(3), llenCmd.Val())

	// Verify prefixed key
	s.True(s.server.Exists("my-prefix-plist"))
	s.False(s.server.Exists("plist"))

	pipe2 := client.Pipeline()
	lpopCmd := pipe2.LPop(ctx, "plist")
	rpopCmd := pipe2.RPop(ctx, "plist")
	_, err = pipe2.Exec(ctx)
	s.NoError(err)
	s.Equal("b", lpopCmd.Val())
	s.Equal("c", rpopCmd.Val())

	// LRem
	pipe3 := client.Pipeline()
	pipe3.LPush(ctx, "plist2", "x", "x", "y")
	_, err = pipe3.Exec(ctx)
	s.NoError(err)

	pipe4 := client.Pipeline()
	lremCmd := pipe4.LRem(ctx, "plist2", 0, "x")
	_, err = pipe4.Exec(ctx)
	s.NoError(err)
	s.Equal(int64(2), lremCmd.Val())
}

func (s *ClientWithMiniRedisTestSuite) TestClient_Prefixing_Pipeline_Hashes() {
	client := s.newPrefixedClient()
	ctx := s.T().Context()

	pipe := client.Pipeline()
	pipe.HSet(ctx, "phash", "f1", "v1")
	pipe.HSet(ctx, "phash", "f2", "v2")
	pipe.HSetNX(ctx, "phash", "f3", "v3")
	pipe.HMSet(ctx, "phash", "f4", "v4")
	_, err := pipe.Exec(ctx)
	s.NoError(err)

	// Verify prefixed key
	s.True(s.server.Exists("my-prefix-phash"))
	s.False(s.server.Exists("phash"))

	pipe2 := client.Pipeline()
	hgetCmd := pipe2.HGet(ctx, "phash", "f1")
	hexistsCmd := pipe2.HExists(ctx, "phash", "f1")
	hexistsFalseCmd := pipe2.HExists(ctx, "phash", "nonexistent")
	hkeysCmd := pipe2.HKeys(ctx, "phash")
	hmgetCmd := pipe2.HMGet(ctx, "phash", "f1", "f2", "f4")
	hgetallCmd := pipe2.HGetAll(ctx, "phash")
	_, err = pipe2.Exec(ctx)
	s.NoError(err)

	s.Equal("v1", hgetCmd.Val())
	s.True(hexistsCmd.Val())
	s.False(hexistsFalseCmd.Val())
	s.ElementsMatch([]string{"f1", "f2", "f3", "f4"}, hkeysCmd.Val())
	s.Equal([]any{"v1", "v2", "v4"}, hmgetCmd.Val())
	s.Equal(map[string]string{"f1": "v1", "f2": "v2", "f3": "v3", "f4": "v4"}, hgetallCmd.Val())

	// HDel
	pipe3 := client.Pipeline()
	hdelCmd := pipe3.HDel(ctx, "phash", "f4")
	_, err = pipe3.Exec(ctx)
	s.NoError(err)
	s.Equal(int64(1), hdelCmd.Val())
}

func (s *ClientWithMiniRedisTestSuite) TestClient_Prefixing_Pipeline_Sets() {
	client := s.newPrefixedClient()
	ctx := s.T().Context()

	pipe := client.Pipeline()
	pipe.SAdd(ctx, "ps1", "a", "b", "c", "d")
	pipe.SAdd(ctx, "ps2", "b", "c")
	pipe.SAdd(ctx, "ps3", "c", "d")
	_, err := pipe.Exec(ctx)
	s.NoError(err)

	s.True(s.server.Exists("my-prefix-ps1"))
	s.False(s.server.Exists("ps1"))

	pipe2 := client.Pipeline()
	scardCmd := pipe2.SCard(ctx, "ps1")
	sismemberCmd := pipe2.SIsMember(ctx, "ps1", "a")
	smembersCmd := pipe2.SMembers(ctx, "ps1")
	sdiffCmd := pipe2.SDiff(ctx, "ps1", "ps2", "ps3")
	sinterCmd := pipe2.SInter(ctx, "ps1", "ps2", "ps3")
	sunionCmd := pipe2.SUnion(ctx, "ps1", "ps2", "ps3")
	_, err = pipe2.Exec(ctx)
	s.NoError(err)

	s.Equal(int64(4), scardCmd.Val())
	s.True(sismemberCmd.Val())
	s.ElementsMatch([]string{"a", "b", "c", "d"}, smembersCmd.Val())
	s.ElementsMatch([]string{"a"}, sdiffCmd.Val())
	s.ElementsMatch([]string{"c"}, sinterCmd.Val())
	s.ElementsMatch([]string{"a", "b", "c", "d"}, sunionCmd.Val())

	// SDiffStore, SInterStore, SUnionStore
	pipe3 := client.Pipeline()
	sdiffStoreCmd := pipe3.SDiffStore(ctx, "pdiff-dest", "ps1", "ps2", "ps3")
	sinterStoreCmd := pipe3.SInterStore(ctx, "pinter-dest", "ps1", "ps2", "ps3")
	sunionStoreCmd := pipe3.SUnionStore(ctx, "punion-dest", "ps1", "ps2", "ps3")
	_, err = pipe3.Exec(ctx)
	s.NoError(err)

	s.Equal(int64(1), sdiffStoreCmd.Val())
	s.Equal(int64(1), sinterStoreCmd.Val())
	s.Equal(int64(4), sunionStoreCmd.Val())

	// Verify destination keys have prefix
	s.True(s.server.Exists("my-prefix-pdiff-dest"))
	s.True(s.server.Exists("my-prefix-pinter-dest"))
	s.True(s.server.Exists("my-prefix-punion-dest"))

	// SMove
	pipe4 := client.Pipeline()
	pipe4.SAdd(ctx, "pmove-src", "x", "y")
	_, err = pipe4.Exec(ctx)
	s.NoError(err)

	pipe5 := client.Pipeline()
	smoveCmd := pipe5.SMove(ctx, "pmove-src", "pmove-dst", "x")
	_, err = pipe5.Exec(ctx)
	s.NoError(err)
	s.True(smoveCmd.Val())
	s.True(s.server.Exists("my-prefix-pmove-dst"))

	// SPop, SRem, SRandMember
	pipe6 := client.Pipeline()
	pipe6.SAdd(ctx, "ppop-set", "one", "two")
	_, err = pipe6.Exec(ctx)
	s.NoError(err)

	pipe7 := client.Pipeline()
	spopCmd := pipe7.SPop(ctx, "ppop-set")
	_, err = pipe7.Exec(ctx)
	s.NoError(err)
	s.Contains([]string{"one", "two"}, spopCmd.Val())

	pipe8 := client.Pipeline()
	pipe8.SAdd(ctx, "prem-set", "a", "b", "c")
	_, err = pipe8.Exec(ctx)
	s.NoError(err)

	pipe9 := client.Pipeline()
	sremCmd := pipe9.SRem(ctx, "prem-set", "a", "c")
	_, err = pipe9.Exec(ctx)
	s.NoError(err)
	s.Equal(int64(2), sremCmd.Val())

	pipe10 := client.Pipeline()
	pipe10.SAdd(ctx, "prand-set", "m1", "m2")
	_, err = pipe10.Exec(ctx)
	s.NoError(err)

	pipe11 := client.Pipeline()
	srandCmd := pipe11.SRandMember(ctx, "prand-set")
	_, err = pipe11.Exec(ctx)
	s.NoError(err)
	s.Contains([]string{"m1", "m2"}, srandCmd.Val())
}

func (s *ClientWithMiniRedisTestSuite) TestClient_Prefixing_Pipeline_SortedSets() {
	client := s.newPrefixedClient()
	ctx := s.T().Context()

	pipe := client.Pipeline()
	pipe.ZAdd(ctx, "pzset", baseRedis.Z{Score: 10, Member: "m0"})
	pipe.ZAdd(ctx, "pzset", baseRedis.Z{Score: 20, Member: "m1"})
	pipe.ZAdd(ctx, "pzset", baseRedis.Z{Score: 30, Member: "m2"})
	pipe.ZAdd(ctx, "pzset", baseRedis.Z{Score: 40, Member: "m3"})
	_, err := pipe.Exec(ctx)
	s.NoError(err)

	s.True(s.server.Exists("my-prefix-pzset"))
	s.False(s.server.Exists("pzset"))

	pipe2 := client.Pipeline()
	zcardCmd := pipe2.ZCard(ctx, "pzset")
	zcountCmd := pipe2.ZCount(ctx, "pzset", "10", "30")
	zscoreCmd := pipe2.ZScore(ctx, "pzset", "m1")
	zrangeCmd := pipe2.ZRange(ctx, "pzset", 0, 1)
	zrankCmd := pipe2.ZRank(ctx, "pzset", "m2")
	zrevrankCmd := pipe2.ZRevRank(ctx, "pzset", "m3")
	zrevrangeCmd := pipe2.ZRevRange(ctx, "pzset", 0, 1)
	_, err = pipe2.Exec(ctx)
	s.NoError(err)

	s.Equal(int64(4), zcardCmd.Val())
	s.Equal(int64(3), zcountCmd.Val())
	s.Equal(20.0, zscoreCmd.Val())
	s.Equal([]string{"m0", "m1"}, zrangeCmd.Val())
	s.Equal(int64(2), zrankCmd.Val())
	s.Equal(int64(0), zrevrankCmd.Val())
	s.Equal([]string{"m3", "m2"}, zrevrangeCmd.Val())

	// ZIncrBy
	pipe3 := client.Pipeline()
	zincrbyCmd := pipe3.ZIncrBy(ctx, "pzset", 100, "m0")
	_, err = pipe3.Exec(ctx)
	s.NoError(err)
	s.Equal(110.0, zincrbyCmd.Val())

	// ZRem
	pipe4 := client.Pipeline()
	zremCmd := pipe4.ZRem(ctx, "pzset", "m3")
	_, err = pipe4.Exec(ctx)
	s.NoError(err)
	s.Equal(int64(1), zremCmd.Val())

	// ZRangeArgs
	pipe5 := client.Pipeline()
	zrangeArgsCmd := pipe5.ZRangeArgs(ctx, baseRedis.ZRangeArgs{
		Key:     "pzset",
		Start:   "20",
		Stop:    "40",
		ByScore: true,
	})
	_, err = pipe5.Exec(ctx)
	s.NoError(err)
	s.Equal([]string{"m1", "m2"}, zrangeArgsCmd.Val())

	// ZRangeArgsWithScores
	pipe6 := client.Pipeline()
	zrangeScoresCmd := pipe6.ZRangeArgsWithScores(ctx, baseRedis.ZRangeArgs{
		Key:   "pzset",
		Start: 0,
		Stop:  0,
	})
	_, err = pipe6.Exec(ctx)
	s.NoError(err)
	s.Len(zrangeScoresCmd.Val(), 1)
	s.Equal("m1", zrangeScoresCmd.Val()[0].Member)
	s.Equal(20.0, zrangeScoresCmd.Val()[0].Score)

	// ZAddArgs + ZAddArgsIncr
	pipe7 := client.Pipeline()
	zaddArgsCmd := pipe7.ZAddArgs(ctx, "pzset", baseRedis.ZAddArgs{
		Members: []baseRedis.Z{{Score: 50, Member: "m4"}},
	})
	_, err = pipe7.Exec(ctx)
	s.NoError(err)
	s.Equal(int64(1), zaddArgsCmd.Val())

	pipe8 := client.Pipeline()
	zaddArgsIncrCmd := pipe8.ZAddArgsIncr(ctx, "pzset", baseRedis.ZAddArgs{
		Members: []baseRedis.Z{{Score: 5, Member: "m4"}},
	})
	_, err = pipe8.Exec(ctx)
	s.NoError(err)
	s.Equal(55.0, zaddArgsIncrCmd.Val())

	// ZRandMember
	pipe9 := client.Pipeline()
	zrandCmd := pipe9.ZRandMember(ctx, "pzset", 2)
	_, err = pipe9.Exec(ctx)
	s.NoError(err)
	s.Len(zrandCmd.Val(), 2)
}

func (s *ClientWithMiniRedisTestSuite) TestClient_Prefixing_Pipeline_Nested() {
	client := s.newPrefixedClient()
	ctx := s.T().Context()

	// Verify that Pipeline().Pipeline() and Pipeline().TxPipeline() also prefix
	pipe := client.Pipeline()
	nestedPipe := pipe.Pipeline()

	nestedPipe.Set(ctx, "nested-key", "nested-val", 0)
	_, err := nestedPipe.Exec(ctx)
	s.NoError(err)

	s.True(s.server.Exists("my-prefix-nested-key"))
	rawVal, err := s.server.Get("my-prefix-nested-key")
	s.NoError(err)
	s.Equal("nested-val", rawVal)

	txPipe := pipe.TxPipeline()
	txPipe.Set(ctx, "tx-key", "tx-val", 0)
	_, err = txPipe.Exec(ctx)
	s.NoError(err)

	s.True(s.server.Exists("my-prefix-tx-key"))
	rawVal, err = s.server.Get("my-prefix-tx-key")
	s.NoError(err)
	s.Equal("tx-val", rawVal)
}

func (s *ClientWithMiniRedisTestSuite) TestSetNX() {
	ctx := s.T().Context()

	ok, err := s.client.SetNX(ctx, "nx-key", "value1", time.Minute)
	s.NoError(err)
	s.True(ok)

	ok, err = s.client.SetNX(ctx, "nx-key", "value2", time.Minute)
	s.NoError(err)
	s.False(ok)

	val, err := s.client.Get(ctx, "nx-key")
	s.NoError(err)
	s.Equal("value1", val)
}

func (s *ClientWithMiniRedisTestSuite) TestMSetMGet() {
	ctx := s.T().Context()

	err := s.client.MSet(ctx, "k1", "v1", "k2", "v2", "k3", "v3")
	s.NoError(err)

	vals, err := s.client.MGet(ctx, "k1", "k2", "k3", "nonexistent")
	s.NoError(err)
	s.Equal([]any{"v1", "v2", "v3", nil}, vals)
}

func (s *ClientWithMiniRedisTestSuite) TestHExists() {
	ctx := s.T().Context()

	err := s.client.HSet(ctx, "hash", "field", "value")
	s.NoError(err)

	exists, err := s.client.HExists(ctx, "hash", "field")
	s.NoError(err)
	s.True(exists)

	exists, err = s.client.HExists(ctx, "hash", "nonexistent")
	s.NoError(err)
	s.False(exists)
}

func (s *ClientWithMiniRedisTestSuite) TestHKeys() {
	ctx := s.T().Context()

	err := s.client.HSet(ctx, "hash", "f1", "v1")
	s.NoError(err)
	err = s.client.HSet(ctx, "hash", "f2", "v2")
	s.NoError(err)

	keys, err := s.client.HKeys(ctx, "hash")
	s.NoError(err)
	s.ElementsMatch([]string{"f1", "f2"}, keys)
}

func (s *ClientWithMiniRedisTestSuite) TestHGet() {
	ctx := s.T().Context()

	err := s.client.HSet(ctx, "hash", "field", "value")
	s.NoError(err)

	val, err := s.client.HGet(ctx, "hash", "field")
	s.NoError(err)
	s.Equal("value", val)

	_, err = s.client.HGet(ctx, "hash", "nonexistent")
	s.Equal(redis.Nil, err)
}

func (s *ClientWithMiniRedisTestSuite) TestZAddArgsIncr() {
	ctx := s.T().Context()

	_, err := s.client.ZAdd(ctx, "zset", 10, "member")
	s.NoError(err)

	args := redis.ZAddArgs{
		Key:     "zset",
		Members: []redis.Z{{Member: "member", Score: 5}},
	}
	newScore, err := s.client.ZAddArgsIncr(ctx, args)
	s.NoError(err)
	s.Equal(15.0, newScore)
}

func (s *ClientWithMiniRedisTestSuite) TestPipeline() {
	ctx := s.T().Context()

	pipe := s.client.Pipeline()
	pipe.Set(ctx, "pipe-key", "pipe-val", 0)
	getCmd := pipe.Get(ctx, "pipe-key")

	_, err := pipe.Exec(ctx)
	s.NoError(err)

	s.Equal("pipe-val", getCmd.Val())
	s.True(s.server.Exists("pipe-key"))
}

func (s *ClientWithMiniRedisTestSuite) TestGetNotFound() {
	// the logger should fail the test as soon as any logger.Warn or anything gets called
	// because we want to test the executor not doing that
	logger := logMocks.NewLoggerMock(logMocks.WithTestingT(s.T()))
	logger.EXPECT().WithFields(mock.AnythingOfType("log.Fields")).Return(logger).Twice()
	executor := redis.NewBackoffExecutor(logger, exec.BackoffSettings{
		CancelDelay:     time.Second,
		InitialInterval: time.Millisecond,
		MaxAttempts:     0,
		MaxInterval:     time.Second * 3,
		MaxElapsedTime:  0,
	}, "test")
	s.client = redis.NewClientWithInterfaces(logger, s.baseClient, executor, s.settings, "")

	res, err := s.client.Get(s.T().Context(), "missing")

	s.Equal(redis.Nil, err)
	s.Equal("", res)
}

func (s *ClientWithMiniRedisTestSuite) TestBLPop() {
	if _, err := s.server.Lpush("list", "value"); err != nil {
		s.FailNow(err.Error(), "can not setup miniredis server")
	}

	res, err := s.client.BLPop(s.T().Context(), 1*time.Second, "list")

	s.NoError(err, "there should be no error on blpop")
	s.Equal("value", res[1])
}

func (s *ClientWithMiniRedisTestSuite) TestDel() {
	count, err := s.client.Del(s.T().Context(), "test")
	s.NoError(err, "there should be no error on Del")
	s.Equal(0, int(count))

	var ttl time.Duration
	err = s.client.Set(s.T().Context(), "key", "value", ttl)
	s.NoError(err, "there should be no error on Del")

	count, err = s.client.Del(s.T().Context(), "key")
	s.NoError(err, "there should be no error on Del")
	s.Equal(1, int(count))
}

func (s *ClientWithMiniRedisTestSuite) TestLLen() {
	for i := 0; i < 3; i++ {
		if _, err := s.server.Lpush("list", "value"); err != nil {
			s.FailNow(err.Error(), "can not setup miniredis server")
		}
	}

	res, err := s.client.LLen(s.T().Context(), "list")

	s.NoError(err, "there should be no error on LLen")
	s.Equal(int64(3), res)
}

func (s *ClientWithMiniRedisTestSuite) TestRPush() {
	count, err := s.client.RPush(s.T().Context(), "list", "v1", "v2", "v3")
	s.NoError(err, "there should be no error on RPush")
	s.Equal(int64(3), count)
}

func (s *ClientWithMiniRedisTestSuite) TestSet() {
	var ttl time.Duration
	err := s.client.Set(s.T().Context(), "key", "value", ttl)
	s.NoError(err, "there should be no error on Set")

	ttl, err = time.ParseDuration("1m")
	s.NoError(err, "there should be no error on ParseDuration")

	err = s.client.Set(s.T().Context(), "key", "value", ttl)
	s.NoError(err, "there should be no error on Set with expiration date")
}

func (s *ClientWithMiniRedisTestSuite) TestHSet() {
	err := s.client.HSet(s.T().Context(), "key", "field", "value")
	s.NoError(err, "there should be no error on HSet")
}

func (s *ClientWithMiniRedisTestSuite) TestHSetNX() {
	isNewlySet, err := s.client.HSetNX(s.T().Context(), "key", "field", "value")
	s.True(isNewlySet, "the field should be set the first time")
	s.NoError(err, "there should be no error on HSet")

	isNewlySet, err = s.client.HSetNX(s.T().Context(), "key", "field", "value")
	s.False(isNewlySet, "the field should NOT be set the first time")
	s.NoError(err, "there should be no error on HSet")
}

func (s *ClientWithMiniRedisTestSuite) TestHMSet() {
	err := s.client.HMSet(s.T().Context(), "key", map[string]any{"field": "value"})
	s.NoError(err, "there should be no error on HSet")
}

func (s *ClientWithMiniRedisTestSuite) TestHMGet() {
	vals, err := s.client.HMGet(s.T().Context(), "key", "field", "value")
	s.NoError(err, "there should be no error on HSet")
	s.Equal([]any{nil, nil}, vals, "there should be no error on HSet")

	err = s.client.HMSet(s.T().Context(), "key", map[string]any{"value": "1"})
	s.NoError(err, "there should be no error on HSet")

	vals, err = s.client.HMGet(s.T().Context(), "key", "field", "value")
	s.NoError(err, "there should be no error on HSet")
	s.Equal([]any{nil, "1"}, vals, "there should be no error on HSet")
}

func (s *ClientWithMiniRedisTestSuite) TestHGetAll() {
	err := s.client.HSet(s.T().Context(), "key", "field1", "value1")
	s.NoError(err, "there should be no error on HSet")
	serr := s.client.HSet(s.T().Context(), "key", "field2", "value2")
	s.NoError(serr, "there should be no error on HSet")

	vals, err := s.client.HGetAll(s.T().Context(), "key")
	s.NoError(err, "there should be no error on HGetAll")
	s.Equal(map[string]string{"field1": "value1", "field2": "value2"}, vals)
}

func (s *ClientWithMiniRedisTestSuite) TestGetDel() {
	var ttl time.Duration
	err := s.client.Set(s.T().Context(), "key", "value1", ttl)
	s.NoError(err, "there should be no error on HSet")

	val, err := s.client.GetDel(s.T().Context(), "key")
	s.NoError(err, "there should be no error on HGetAll")
	s.Equal("value1", val)

	_, err = s.client.GetDel(s.T().Context(), "key")
	s.Equal(redis.Nil, err)
}

func (s *ClientWithMiniRedisTestSuite) TestGetSet() {
	val, err := s.client.GetSet(s.T().Context(), "key", "value1")
	s.Equal(redis.Nil, err)
	s.Equal("", val)

	val, err = s.client.GetSet(s.T().Context(), "key", "value2")
	s.NoError(err, "there should be no error on GetSet")
	s.Equal("value1", val)
}

func (s *ClientWithMiniRedisTestSuite) TestHDel() {
	err := s.client.HSet(s.T().Context(), "key", "field", "value1")
	s.NoError(err, "there should be no error on HSet")
	serr := s.client.HSet(s.T().Context(), "key", "field2", "value2")
	s.NoError(serr, "there should be no error on HSet")

	vals, err := s.client.HDel(s.T().Context(), "key", "field2")
	s.NoError(err, "there should be no error on HDel")
	s.Equal(int64(1), vals)

	valuesFromMap, err := s.client.HGetAll(s.T().Context(), "key")
	s.NoError(err, "there should be no error on HGetAll")
	s.Equal(map[string]string{"field": "value1"}, valuesFromMap)
}

func (s *ClientWithMiniRedisTestSuite) TestSAdd() {
	_, err := s.client.SAdd(s.T().Context(), "key", "value")
	s.NoError(err, "there should be no error on SAdd")
}

func (s *ClientWithMiniRedisTestSuite) TestSCard() {
	_, err := s.client.SAdd(s.T().Context(), "key", "value1")
	s.NoError(err, "there should be no error on SAdd")
	_, err = s.client.SAdd(s.T().Context(), "key", "value2")
	s.NoError(err, "there should be no error on SAdd")
	_, err = s.client.SAdd(s.T().Context(), "key", "value2")
	s.NoError(err, "there should be no error on SAdd")

	amount, err := s.client.SCard(s.T().Context(), "key")
	s.Equal(int64(2), amount)
	s.NoError(err, "there should be no error on SCard")
}

func (s *ClientWithMiniRedisTestSuite) TestSDiff() {
	_, err := s.client.SAdd(s.T().Context(), "set1", "1", "2", "3", "4")
	s.NoError(err, "there should be no error on SAdd")
	_, err = s.client.SAdd(s.T().Context(), "set2", "2", "3")
	s.NoError(err, "there should be no error on SAdd")
	_, err = s.client.SAdd(s.T().Context(), "set3", "2", "4")
	s.NoError(err, "there should be no error on SAdd")

	diffs, err := s.client.SDiff(s.T().Context(), "set1", "set2", "set3")
	s.Equal(1, len(diffs))
	s.True(slices.Contains(diffs, "1"))
	s.NoError(err, "there should be no error on SDiff")
}

func (s *ClientWithMiniRedisTestSuite) TestSDiffStore() {
	_, err := s.client.SAdd(s.T().Context(), "set1", "1", "2", "3", "4")
	s.NoError(err, "there should be no error on SAdd")
	_, err = s.client.SAdd(s.T().Context(), "set2", "2", "3")
	s.NoError(err, "there should be no error on SAdd")
	_, err = s.client.SAdd(s.T().Context(), "set3", "2", "4")
	s.NoError(err, "there should be no error on SAdd")

	numDiffs, err := s.client.SDiffStore(s.T().Context(), "set4", "set1", "set2", "set3")
	s.Equal(int64(1), numDiffs)
	s.NoError(err, "there should be no error on SDiffStore")
}

func (s *ClientWithMiniRedisTestSuite) TestSInter() {
	_, err := s.client.SAdd(s.T().Context(), "set1", "1", "2", "3", "4")
	s.NoError(err, "there should be no error on SAdd")
	_, err = s.client.SAdd(s.T().Context(), "set2", "2", "3")
	s.NoError(err, "there should be no error on SAdd")
	_, err = s.client.SAdd(s.T().Context(), "set3", "2", "4")
	s.NoError(err, "there should be no error on SAdd")

	inters, err := s.client.SInter(s.T().Context(), "set1", "set2", "set3")
	s.Equal(1, len(inters))
	s.True(slices.Contains(inters, "2"))
	s.NoError(err, "there should be no error on SInter")
}

func (s *ClientWithMiniRedisTestSuite) TestSInterStore() {
	_, err := s.client.SAdd(s.T().Context(), "set1", "1", "2", "3", "4")
	s.NoError(err, "there should be no error on SAdd")
	_, err = s.client.SAdd(s.T().Context(), "set2", "2", "3")
	s.NoError(err, "there should be no error on SAdd")
	_, err = s.client.SAdd(s.T().Context(), "set3", "2", "4")
	s.NoError(err, "there should be no error on SAdd")

	numDiffs, err := s.client.SInterStore(s.T().Context(), "set4", "set1", "set2", "set3")
	s.Equal(int64(1), numDiffs)
	s.NoError(err, "there should be no error on SInterStore")
}

func (s *ClientWithMiniRedisTestSuite) TestSMembers() {
	_, err := s.client.SAdd(s.T().Context(), "set", "0", "1", "2")
	s.NoError(err, "there should be no error on SAdd")

	members, err := s.client.SMembers(s.T().Context(), "set")
	s.Equal(3, len(members))
	s.True(slices.Contains(members, "0"))
	s.True(slices.Contains(members, "1"))
	s.True(slices.Contains(members, "2"))
	s.NoError(err, "there should be no error on SMembers")
}

func (s *ClientWithMiniRedisTestSuite) TestSIsMember() {
	_, err := s.client.SAdd(s.T().Context(), "key", "value1")
	s.NoError(err, "there should be no error on SAdd")

	isMember, err := s.client.SIsMember(s.T().Context(), "key", "value1")
	s.Equal(true, isMember)
	s.NoError(err, "there should be no error on SIsMember")

	isMember, err = s.client.SIsMember(s.T().Context(), "key", "value2")
	s.Equal(false, isMember)
	s.NoError(err, "there should be no error on SIsMember")
}

func (s *ClientWithMiniRedisTestSuite) TestSMove() {
	_, err := s.client.SAdd(s.T().Context(), "set1", "0", "1", "2")
	s.NoError(err, "there should be no error on SAdd")

	_, err = s.client.SMove(s.T().Context(), "set1", "set2", "1")
	s.NoError(err, "there should be no error on SMove")

	members, err := s.client.SMembers(s.T().Context(), "set2")
	s.Equal(1, len(members))
	s.True(slices.Contains(members, "1"))
	s.NoError(err, "there should be no error on SMembers")
}

func (s *ClientWithMiniRedisTestSuite) TestSPop() {
	_, err := s.client.SAdd(s.T().Context(), "set1", "0", "1", "2")
	s.NoError(err, "there should be no error on SAdd")

	_, err = s.client.SPop(s.T().Context(), "set1")
	s.NoError(err, "there should be no error on SPop")
}

func (s *ClientWithMiniRedisTestSuite) TestSRem() {
	count, err := s.client.SAdd(s.T().Context(), "set", "v1", "v2", "v3")
	s.NoError(err, "there should be no error on SAdd")
	s.Equal(int64(3), count)

	numRemoved, err := s.client.SRem(s.T().Context(), "set", "v1", "v3")
	s.NoError(err, "there should be no error on SRem")
	s.Equal(int64(2), numRemoved)
}

func (s *ClientWithMiniRedisTestSuite) TestSRandMember() {
	count, err := s.client.SAdd(s.T().Context(), "set", "v1", "v2", "v3")
	s.NoError(err, "there should be no error on SAdd")
	s.Equal(int64(3), count)

	item, err := s.client.SRandMember(s.T().Context(), "set")
	s.NoError(err, "there should be no error on SRandMember")

	contained, err := s.client.SIsMember(s.T().Context(), "set", item)
	s.Equal(true, contained)
	s.NoError(err, "there should be no error on SIsMember")
}

func (s *ClientWithMiniRedisTestSuite) TestSUnion() {
	_, err := s.client.SAdd(s.T().Context(), "set1", "1", "2", "3", "4")
	s.NoError(err, "there should be no error on SAdd")
	_, err = s.client.SAdd(s.T().Context(), "set2", "2", "3")
	s.NoError(err, "there should be no error on SAdd")
	_, err = s.client.SAdd(s.T().Context(), "set3", "2", "4")
	s.NoError(err, "there should be no error on SAdd")

	union, err := s.client.SUnion(s.T().Context(), "set1", "set2", "set3")
	s.Equal(4, len(union))
	s.NoError(err, "there should be no error on SUnion")
}

func (s *ClientWithMiniRedisTestSuite) TestSUnionStore() {
	_, err := s.client.SAdd(s.T().Context(), "set1", "1", "2", "3", "4")
	s.NoError(err, "there should be no error on SAdd")
	_, err = s.client.SAdd(s.T().Context(), "set2", "2", "3")
	s.NoError(err, "there should be no error on SAdd")
	_, err = s.client.SAdd(s.T().Context(), "set3", "2", "4")
	s.NoError(err, "there should be no error on SAdd")

	numUnion, err := s.client.SUnionStore(s.T().Context(), "set4", "set1", "set2", "set3")
	s.Equal(int64(4), numUnion)
	s.NoError(err, "there should be no error on SUnionStore")
}

func (s *ClientWithMiniRedisTestSuite) TestIncr() {
	val, err := s.client.Incr(s.T().Context(), "key")
	s.NoError(err, "there should be no error on Incr")
	s.Equal(int64(1), val)

	val, err = s.client.Incr(s.T().Context(), "key")
	s.NoError(err, "there should be no error on Incr")
	s.Equal(int64(2), val)

	val, err = s.client.IncrBy(s.T().Context(), "key", int64(3))
	s.NoError(err, "there should be no error on IncrBy")
	s.Equal(int64(5), val)
}

func (s *ClientWithMiniRedisTestSuite) TestDecr() {
	err := s.client.Set(s.T().Context(), "key", 10, time.Minute*10)
	s.NoError(err, "there should be no error on Set")

	val, err := s.client.Decr(s.T().Context(), "key")
	s.NoError(err, "there should be no error on Decr")
	s.Equal(int64(9), val)

	val, err = s.client.Decr(s.T().Context(), "key")
	s.NoError(err, "there should be no error on Decr")
	s.Equal(int64(8), val)

	val, err = s.client.DecrBy(s.T().Context(), "key", int64(5))
	s.NoError(err, "there should be no error on DecrBy")
	s.Equal(int64(3), val)
}

func (s *ClientWithMiniRedisTestSuite) TestZAdd() {
	inserts, err := s.client.ZAdd(s.T().Context(), "key", 10.0, "member")
	s.NoError(err, "there should be no error on ZAdd")
	s.EqualValues(1, inserts)
}

func (s *ClientWithMiniRedisTestSuite) TestZAddArgs() {
	args := redis.ZAddArgs{
		Key: "key",
		Members: []redis.Z{
			{Member: "member1", Score: 20},
			{Member: "member2", Score: 50},
			{Member: "member3", Score: 110},
			{Member: "member4", Score: 160},
		},
	}
	inserts, err := s.client.ZAddArgs(s.T().Context(), args)
	s.NoError(err, "there should be no error on ZAdd")
	s.EqualValues(4, inserts)
}

func (s *ClientWithMiniRedisTestSuite) TestZCard() {
	cardinality, err := s.client.ZCard(s.T().Context(), "key")
	s.NoError(err, "there should be no error on ZCard")
	s.EqualValues(0, cardinality)

	_, err = s.client.ZAdd(s.T().Context(), "key", 10.0, "member0")
	s.NoError(err, "there should be no error on ZAdd")

	_, err = s.client.ZAdd(s.T().Context(), "key", 10.0, "member1")
	s.NoError(err, "there should be no error on ZAdd")

	_, err = s.client.ZAdd(s.T().Context(), "key", 11.0, "member2")
	s.NoError(err, "there should be no error on ZAdd")

	_, err = s.client.ZAdd(s.T().Context(), "key", 12.0, "member3")
	s.NoError(err, "there should be no error on ZAdd")

	cardinality, err = s.client.ZCard(s.T().Context(), "key")
	s.NoError(err, "there should be no error on ZCard")
	s.EqualValues(4, cardinality)
}

func (s *ClientWithMiniRedisTestSuite) TestZCount() {
	_, err := s.client.ZAdd(s.T().Context(), "key", 10, "member0")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 50, "member1")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 90, "member2")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 100, "member3")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 120, "member4")
	s.NoError(err)

	count, err := s.client.ZCount(s.T().Context(), "key", "(50", "100")
	s.NoError(err, "there should be no error on ZCount")
	s.EqualValues(2, count)
}

func (s *ClientWithMiniRedisTestSuite) TestZIncrBy() {
	_, err := s.client.ZIncrBy(s.T().Context(), "key", 10.0, "member")
	s.NoError(err)
	_, err = s.client.ZIncrBy(s.T().Context(), "key", 50.0, "member")
	s.NoError(err)

	score, err := s.client.ZScore(s.T().Context(), "key", "member")
	s.NoError(err, "there should be no error on ZIncrBy")
	s.EqualValues(60.0, score)
}

func (s *ClientWithMiniRedisTestSuite) TestZScore() {
	_, err := s.client.ZAdd(s.T().Context(), "key", 10.0, "member")
	s.NoError(err, "there should be no error on ZAdd")

	score, err := s.client.ZScore(s.T().Context(), "key", "member")
	s.NoError(err, "there should be no error on ZScore")
	s.EqualValues(10.0, score)
}

func (s *ClientWithMiniRedisTestSuite) TestZRange() {
	_, err := s.client.ZAdd(s.T().Context(), "key", 10, "member0")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 50, "member1")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 90, "member2")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 100, "member3")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 120, "member4")
	s.NoError(err)

	members, err := s.client.ZRange(s.T().Context(), "key", 1, 2)
	s.NoError(err, "there should be no error on ZRange")
	s.EqualValues(2, len(members))
	s.Equal("member1", members[0])
	s.Equal("member2", members[1])
}

func (s *ClientWithMiniRedisTestSuite) TestZRangeArgsWithScore() {
	_, err := s.client.ZAdd(s.T().Context(), "key", 10, "member0")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 50, "member1")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 90, "member2")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 100, "member3")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 120, "member4")
	s.NoError(err)

	args := redis.ZRangeArgs{
		Key:   "key",
		Start: 1,
		Stop:  2,
	}

	zs, err := s.client.ZRangeArgsWithScore(s.T().Context(), args)
	s.NoError(err, "there should be no error on ZRangeArgsWithScore")
	s.EqualValues(2, len(zs))
	s.Equal("member1", zs[0].Member)
	s.Equal(float64(50), zs[0].Score)
	s.Equal("member2", zs[1].Member)
	s.Equal(float64(90), zs[1].Score)
}

func (s *ClientWithMiniRedisTestSuite) TestZRangeArgsWithScoreRev() {
	_, err := s.client.ZAdd(s.T().Context(), "key", 10, "member0")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 50, "member1")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 90, "member2")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 100, "member3")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 120, "member4")
	s.NoError(err)

	args := redis.ZRangeArgs{
		Key:   "key",
		Start: 1,
		Stop:  2,
		Rev:   true,
	}

	zs, err := s.client.ZRangeArgsWithScore(s.T().Context(), args)
	s.NoError(err, "there should be no error on ZRangeArgsWithScore")
	s.EqualValues(2, len(zs))
	s.Equal("member3", zs[0].Member)
	s.Equal(float64(100), zs[0].Score)
	s.Equal("member2", zs[1].Member)
	s.Equal(float64(90), zs[1].Score)
}

func (s *ClientWithMiniRedisTestSuite) TestZRangeArgsByScoreWithLimit() {
	_, err := s.client.ZAdd(s.T().Context(), "key", 10, "member0")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 50, "member1")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 90, "member2")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 100, "member3")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 120, "member4")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 150, "member5")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 170, "member6")
	s.NoError(err)

	args := redis.ZRangeArgs{
		Key:     "key",
		Start:   "(10",
		Stop:    "200",
		Rev:     false,
		ByScore: true,
		Offset:  1,
		Count:   3,
	}

	zs, err := s.client.ZRangeArgs(s.T().Context(), args)
	s.NoError(err, "there should be no error on ZRangeArgs")
	s.EqualValues(3, len(zs))
	s.Equal("member2", zs[0])
	s.Equal("member3", zs[1])
	s.Equal("member4", zs[2])
}

func (s *ClientWithMiniRedisTestSuite) TestZRank() {
	_, err := s.client.ZAdd(s.T().Context(), "key", 10, "member0")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 50, "member1")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 90, "member2")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 100, "member3")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 120, "member4")
	s.NoError(err)

	rank, err := s.client.ZRank(s.T().Context(), "key", "member3")
	s.NoError(err, "there should be no error on ZRank")
	s.EqualValues(3, rank)
}

func (s *ClientWithMiniRedisTestSuite) TestZRem() {
	_, err := s.client.ZAdd(s.T().Context(), "key", 10, "member0")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 50, "member1")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 90, "member2")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 100, "member3")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 120, "member4")
	s.NoError(err)

	removes, err := s.client.ZRem(s.T().Context(), "key", "member2", "member3", "memberX")
	s.NoError(err, "there should be no error on ZRem")
	s.EqualValues(2, removes)
}

func (s *ClientWithMiniRedisTestSuite) TestZRevRank() {
	_, err := s.client.ZAdd(s.T().Context(), "key", 10, "member0")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 50, "member1")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 90, "member2")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 100, "member3")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 120, "member4")
	s.NoError(err)

	rank, err := s.client.ZRevRank(s.T().Context(), "key", "member3")
	s.NoError(err, "there should be no error on ZRevRank")
	s.EqualValues(1, rank)
}

func (s *ClientWithMiniRedisTestSuite) TestZRandMember() {
	_, err := s.client.ZAdd(s.T().Context(), "key", 10, "member0")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 50, "member1")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 90, "member2")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 100, "member3")
	s.NoError(err)
	_, err = s.client.ZAdd(s.T().Context(), "key", 120, "member4")
	s.NoError(err)

	members, err := s.client.ZRandMember(s.T().Context(), "key", 2)
	s.NoError(err, "there should be no error on ZRandMember")
	s.EqualValues(2, len(members))
}

func (s *ClientWithMiniRedisTestSuite) TestLPush() {
	count, err := s.client.LPush(s.T().Context(), "list", "v1", "v2", "v3")
	s.NoError(err, "there should be no error on LPush")
	s.Equal(int64(3), count)
}

func (s *ClientWithMiniRedisTestSuite) TestLRem() {
	count, err := s.client.LPush(s.T().Context(), "list", "v1", "v2", "v3")
	s.NoError(err, "there should be no error on LPush")
	s.Equal(int64(3), count)

	numRemoved, err := s.client.LRem(s.T().Context(), "list", 0, "v1")
	s.NoError(err, "there should be no error on LRem")
	s.Equal(int64(1), numRemoved)
}

func (s *ClientWithMiniRedisTestSuite) TestLPop() {
	count, err := s.client.RPush(s.T().Context(), "list", "v1", "v2", "v3")
	s.NoError(err, "there should be no error on RPush")
	s.Equal(int64(3), count)

	item, err := s.client.LPop(s.T().Context(), "list")
	s.NoError(err, "there should be no error on LPop")
	s.Equal("v1", item)
}

func (s *ClientWithMiniRedisTestSuite) TestRPop() {
	count, err := s.client.LPush(s.T().Context(), "list", "v1", "v2", "v3")
	s.NoError(err, "there should be no error on LPush")
	s.Equal(int64(3), count)

	item, err := s.client.RPop(s.T().Context(), "list")
	s.NoError(err, "there should be no error on RPop")
	s.Equal("v1", item)
}

func (s *ClientWithMiniRedisTestSuite) TestExpire() {
	_, err := s.client.Incr(s.T().Context(), "key")
	s.NoError(err)

	result, err := s.client.Expire(s.T().Context(), "key", time.Second)
	s.NoError(err, "there should be no error on Expire")
	s.True(result)

	s.server.FastForward(time.Second)

	amount, err := s.client.Exists(s.T().Context(), "key")
	s.Equal(int64(0), amount)
	s.NoError(err, "there should be no error on Exists")
}

func (s *ClientWithMiniRedisTestSuite) TestIsAlive() {
	alive := s.client.IsAlive(s.T().Context())
	s.True(alive)
}

func TestClientWithMiniRedisTestSuite(t *testing.T) {
	suite.Run(t, new(ClientWithMiniRedisTestSuite))
}

type ClientWithMockTestSuite struct {
	suite.Suite
	client    redis.Client
	redisMock *redismock.ClientMock
}

func (s *ClientWithMockTestSuite) SetupTest() {
	settings := &redis.Settings{}
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(s.T()))
	executor := redis.NewBackoffExecutor(logger, settings.BackoffSettings, "test")

	s.redisMock = redismock.NewMock()
	s.client = redis.NewClientWithInterfaces(logger, s.redisMock, executor, settings, "")
}

func (s *ClientWithMockTestSuite) TestSetWithOOM() {
	s.redisMock.On("Set", s.T().Context(), "key", "value", time.Duration(1000000000)).Return(baseRedis.NewStatusResult("", errors.New("OOM command not allowed when used memory > 'maxmemory'"))).Once()
	s.redisMock.On("Set", s.T().Context(), "key", "value", time.Duration(1000000000)).Return(baseRedis.NewStatusResult("", nil)).Once()

	err := s.client.Set(s.T().Context(), "key", "value", time.Second)

	s.NoError(err, "there should be no error on Set with backoff")
	s.redisMock.AssertExpectations(s.T())
}

func (s *ClientWithMockTestSuite) TestSetWithError() {
	s.redisMock.On("Set", s.T().Context(), "key", "value", time.Duration(1000000000)).Return(baseRedis.NewStatusResult("", errors.New("random redis error"))).Once()
	s.redisMock.On("Set", s.T().Context(), "key", "value", time.Duration(1000000000)).Return(baseRedis.NewStatusResult("", nil)).Times(0)

	err := s.client.Set(s.T().Context(), "key", "value", time.Second)

	s.NotNil(err, "there should be an error on Set")
	s.redisMock.AssertExpectations(s.T())
}

func (s *ClientWithMockTestSuite) TestPFAddCountMerge() {
	// miniredis doesn't support HyperLogLogs, so we need to mock these
	s.redisMock.On("PFAdd", s.T().Context(), "key1", []any{"a", "b", "c"}).Return(baseRedis.NewIntResult(1, nil)).Once()
	s.redisMock.On("PFAdd", s.T().Context(), "key2", []any{"d"}).Return(baseRedis.NewIntResult(1, nil)).Once()
	s.redisMock.On("PFAdd", s.T().Context(), "key2", []any{"e", "f"}).Return(baseRedis.NewIntResult(1, nil)).Once()
	s.redisMock.On("PFMerge", s.T().Context(), "key", []string{"key1", "key2"}).Return(baseRedis.NewStatusResult("OK", nil)).Once()
	s.redisMock.On("PFCount", s.T().Context(), []string{"key"}).Return(baseRedis.NewIntResult(6, nil)).Once()

	result, err := s.client.PFAdd(s.T().Context(), "key1", "a", "b", "c")
	s.NoError(err, "There should be no error on PFAdd")
	s.Equal(int64(1), result)

	result, err = s.client.PFAdd(s.T().Context(), "key2", "d")
	s.NoError(err, "There should be no error on PFAdd")
	s.Equal(int64(1), result)

	result, err = s.client.PFAdd(s.T().Context(), "key2", "e", "f")
	s.NoError(err, "There should be no error on PFAdd")
	s.Equal(int64(1), result)

	_, err = s.client.PFMerge(s.T().Context(), "key", "key1", "key2")
	s.NoError(err, "There should be no error on PFMerge")

	result, err = s.client.PFCount(s.T().Context(), "key")
	s.NoError(err, "There should be no error on PFCount")
	s.Equal(int64(6), result)
}

func (s *ClientWithMockTestSuite) TestPrefixing_PFAddCountMerge() {
	// Create a prefixed client using the same mock
	settings := &redis.Settings{}
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(s.T()))
	executor := redis.NewBackoffExecutor(logger, settings.BackoffSettings, "test")
	prefixedClient := redis.NewClientWithInterfaces(logger, s.redisMock, executor, settings, "pfx-")

	// Expect all keys to have prefix
	s.redisMock.On("PFAdd", s.T().Context(), "pfx-hll1", []any{"a", "b"}).Return(baseRedis.NewIntResult(1, nil)).Once()
	s.redisMock.On("PFAdd", s.T().Context(), "pfx-hll2", []any{"c"}).Return(baseRedis.NewIntResult(1, nil)).Once()
	s.redisMock.On("PFMerge", s.T().Context(), "pfx-dest", []string{"pfx-hll1", "pfx-hll2"}).Return(baseRedis.NewStatusResult("OK", nil)).Once()
	s.redisMock.On("PFCount", s.T().Context(), []string{"pfx-dest"}).Return(baseRedis.NewIntResult(3, nil)).Once()

	result, err := prefixedClient.PFAdd(s.T().Context(), "hll1", "a", "b")
	s.NoError(err)
	s.Equal(int64(1), result)

	result, err = prefixedClient.PFAdd(s.T().Context(), "hll2", "c")
	s.NoError(err)
	s.Equal(int64(1), result)

	_, err = prefixedClient.PFMerge(s.T().Context(), "dest", "hll1", "hll2")
	s.NoError(err)

	result, err = prefixedClient.PFCount(s.T().Context(), "dest")
	s.NoError(err)
	s.Equal(int64(3), result)

	s.redisMock.AssertExpectations(s.T())
}

func TestClientWithMockTestSuite(t *testing.T) {
	suite.Run(t, new(ClientWithMockTestSuite))
}
