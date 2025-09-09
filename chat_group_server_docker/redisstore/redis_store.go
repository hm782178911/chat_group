package redisstore

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisStore struct {
	client *redis.Client
	ctx    context.Context
}

func NewRedisStore(addr, password string, db int) *RedisStore {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	ctx := context.Background()

	// 测试连接
	if err := client.Ping(ctx).Err(); err != nil {
		log.Fatalf("无法连接Redis: %v", err)
	}

	log.Println("✅ Redis连接成功")
	return &RedisStore{
		client: client,
		ctx:    ctx,
	}
}

// SaveMessage 保存消息到Redis
func (r *RedisStore) SaveMessage(message map[string]interface{}) error {
	messageJSON, err := json.Marshal(message)
	if err != nil {
		return err
	}

	// 使用有序集合存储消息，以时间戳作为分数
	timestamp := time.Now().UnixNano()
	score := float64(timestamp) / 1e9 // 转换为秒级精度

	_, err = r.client.ZAdd(r.ctx, "chat:messages", redis.Z{
		Score:  score,
		Member: messageJSON,
	}).Result()

	if err != nil {
		return fmt.Errorf("保存消息到Redis失败: %v", err)
	}

	// 限制消息历史数量（保留最近1000条）
	r.client.ZRemRangeByRank(r.ctx, "chat:messages", 0, -1001)

	return nil
}

// GetRecentMessages 获取最近的消息
func (r *RedisStore) GetRecentMessages(count int64) ([]map[string]interface{}, error) {
	result, err := r.client.ZRevRange(r.ctx, "chat:messages", 0, count-1).Result()
	if err != nil {
		return nil, err
	}

	var messages []map[string]interface{}
	for _, msgJSON := range result {
		var message map[string]interface{}
		if err := json.Unmarshal([]byte(msgJSON), &message); err != nil {
			log.Printf("解析消息JSON失败: %v", err)
			continue
		}
		messages = append(messages, message)
	}

	return messages, nil
}

// SaveUser 保存用户信息到Redis
func (r *RedisStore) SaveUser(user map[string]interface{}) error {
	userJSON, err := json.Marshal(user)
	if err != nil {
		return err
	}

	username, ok := user["username"].(string)
	if !ok {
		return fmt.Errorf("无效的用户名")
	}

	_, err = r.client.HSet(r.ctx, "chat:users", username, userJSON).Result()
	return err
}

// GetAllUsers 获取所有用户
func (r *RedisStore) GetAllUsers() (map[string]map[string]interface{}, error) {
	result, err := r.client.HGetAll(r.ctx, "chat:users").Result()
	if err != nil {
		return nil, err
	}

	users := make(map[string]map[string]interface{})
	for username, userJSON := range result {
		var user map[string]interface{}
		if err := json.Unmarshal([]byte(userJSON), &user); err != nil {
			log.Printf("解析用户JSON失败: %v", err)
			continue
		}
		users[username] = user
	}

	return users, nil
}

// UpdateUserOnlineStatus 更新用户在线状态
func (r *RedisStore) UpdateUserOnlineStatus(username string, isOnline bool) error {
	userJSON, err := r.client.HGet(r.ctx, "chat:users", username).Result()
	if err != nil && err != redis.Nil {
		return err
	}

	var user map[string]interface{}
	if userJSON != "" {
		if err := json.Unmarshal([]byte(userJSON), &user); err != nil {
			return err
		}
	} else {
		user = make(map[string]interface{})
		user["username"] = username
	}

	user["is_online"] = isOnline
	user["last_seen"] = time.Now().Format(time.RFC3339)

	return r.SaveUser(user)
}

// Close 关闭Redis连接
func (r *RedisStore) Close() error {
	return r.client.Close()
}