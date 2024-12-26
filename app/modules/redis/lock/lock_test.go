package lock

//
//import (
//	"context"
//	srv "github.com/garyburd/redigo/redis"
//	"github.com/stretchr/testify/assert"
//	"testing"
//	"willshark/app/modules/redis"
//	"willshark/utils/strings"
//)
//
//func runOnRedis(t *testing.T, fn func(client *redis.Redis)) {
//	host := "172.16.27.142:16379"
//	rs, err := srv.Dial("tcp", host)
//	defer rs.Close()
//	assert.Nil(t, err)
//	client := redis.Redis{
//		Cache: rs,
//	}
//	fn(&client)
//}
//
//func TestRedisLock(t *testing.T) {
//	testFn := func(ctx context.Context) func(client *redis.Redis) {
//		return func(client *redis.RedisPool) {
//			key := strings.Rand()
//			firstLock := NewRedisLock(client, key)
//			// Todo： Adjust the expiration time as needed
//			firstLock.SetExpire(5)
//			firstAcquire, err := firstLock.Acquire()
//			assert.Nil(t, err)
//			assert.True(t, firstAcquire)
//
//			secondLock := NewRedisLock(client, key)
//			secondLock.SetExpire(5)
//			againAcquire, err := secondLock.Acquire()
//			assert.Nil(t, err)
//			assert.False(t, againAcquire)
//
//			release, err := firstLock.Release()
//			assert.Nil(t, err)
//			assert.True(t, release)
//
//			endAcquire, err := secondLock.Acquire()
//			assert.Nil(t, err)
//			assert.True(t, endAcquire)
//		}
//	}
//
//	t.Run("normal", func(t *testing.T) {
//		runOnRedis(t, testFn(nil))
//	})
//
//	t.Run("context", func(t *testing.T) {
//		runOnRedis(t, testFn(context.Background()))
//	})
//}
