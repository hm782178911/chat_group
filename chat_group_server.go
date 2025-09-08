package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
	// "strings"
)

// Message 结构体表示一条聊天消息
type Message struct {
	Sender    string    `json:"sender"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"` // "message", "join", "leave"
}

// User 用户信息
type User struct {
	Name      string    `json:"name"`
	LastSeen  time.Time `json:"last_seen"`
	IsOnline  bool      `json:"is_online"`
}

// ChatServer 管理聊天状态
type ChatServer struct {
	messages      []Message           // 所有消息历史
	users         map[string]*User    // 在线用户
	clients       map[chan Message]bool // 客户端连接通道
	mutex         sync.RWMutex        // 保护共享数据的读写锁
}

// NewChatServer 创建新的聊天服务器实例
func NewChatServer() *ChatServer {
	return &ChatServer{
		messages: make([]Message, 0),
		users:    make(map[string]*User),
		clients:  make(map[chan Message]bool),
	}
}

// broadcast 广播消息给所有客户端
func (cs *ChatServer) broadcast(message Message) {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	// 添加到消息历史
	cs.messages = append(cs.messages, message)

	// 广播给所有连接的客户端
	for client := range cs.clients {
		select {
		case client <- message:
		default:
			// 如果客户端无法接收，跳过
		}
	}
}

// JoinHandler 处理用户加入
func (cs *ChatServer) JoinHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "只支持POST方法", http.StatusMethodNotAllowed)
		return
	}

	username := r.FormValue("username")
	if username == "" {
		http.Error(w, "username 参数是必需的", http.StatusBadRequest)
		return
	}

	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	// 检查用户名是否已存在
	if _, exists := cs.users[username]; exists {
		// http.Error(w, "用户名已存在", http.StatusConflict)
		// return
	}

	// 添加新用户
	cs.users[username] = &User{
		Name:     username,
		LastSeen: time.Now(),
		IsOnline: true,
	}

	// 广播用户加入消息
	joinMsg := Message{
		Sender:    "系统",
		Content:   fmt.Sprintf("用户 %s 加入了聊天室", username),
		Timestamp: time.Now(),
		Type:      "join",
	}
	go cs.broadcast(joinMsg)

	log.Printf("用户 %s 加入了聊天室", username)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "success",
		"message":  "加入成功",
		"username": username,
	})
}

// LeaveHandler 处理用户离开
func (cs *ChatServer) LeaveHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "只支持POST方法", http.StatusMethodNotAllowed)
		return
	}

	username := r.FormValue("username")
	if username == "" {
		http.Error(w, "username 参数是必需的", http.StatusBadRequest)
		return
	}

	cs.mutex.Lock()
	defer cs.mutex.Unlock()

	// 移除用户
	if user, exists := cs.users[username]; exists {
		user.IsOnline = false
		user.LastSeen = time.Now()
	}

	// 广播用户离开消息
	leaveMsg := Message{
		Sender:    "系统",
		Content:   fmt.Sprintf("用户 %s 离开了聊天室", username),
		Timestamp: time.Now(),
		Type:      "leave",
	}
	go cs.broadcast(leaveMsg)

	log.Printf("用户 %s 离开了聊天室", username)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "离开成功",
	})
}

// SendHandler 处理发送消息
func (cs *ChatServer) SendHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "只支持POST方法", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "解析表单数据失败", http.StatusBadRequest)
		return
	}

	sender := r.FormValue("sender")
	content := r.FormValue("content")

	if sender == "" || content == "" {
		http.Error(w, "sender 和 content 参数都是必需的", http.StatusBadRequest)
		return
	}

	// 创建新消息
	message := Message{
		Sender:    sender,
		Content:   content,
		Timestamp: time.Now(),
		Type:      "message",
	}

	// 广播消息
	go cs.broadcast(message)

	log.Printf("消息: %s -> %s", sender, content)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "消息发送成功",
	})
}

// StreamHandler 处理实时消息流
func (cs *ChatServer) StreamHandler(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("user")
	if username == "" {
		http.Error(w, "user 参数是必需的", http.StatusBadRequest)
		return
	}

	// 设置SSE头
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// 创建客户端通道
	messageChan := make(chan Message, 10)

	cs.mutex.Lock()
	cs.clients[messageChan] = true
	cs.mutex.Unlock()

	// 连接关闭时清理
	defer func() {
		cs.mutex.Lock()
		delete(cs.clients, messageChan)
		cs.mutex.Unlock()
		close(messageChan)
	}()

	// 发送历史消息
	cs.mutex.RLock()
	history := make([]Message, len(cs.messages))
	copy(history, cs.messages)
	cs.mutex.RUnlock()

	for _, msg := range history {
		data, _ := json.Marshal(msg)
		fmt.Fprintf(w, "data: %s\n\n", data)
		w.(http.Flusher).Flush()
	}

	// 实时推送新消息
	for {
		select {
		case msg := <-messageChan:
			data, err := json.Marshal(msg)
			if err != nil {
				log.Printf("JSON编码错误: %v", err)
				continue
			}
			fmt.Fprintf(w, "data: %s\n\n", data)
			w.(http.Flusher).Flush()

		case <-r.Context().Done():
			// 客户端断开连接
			log.Printf("用户 %s 的SSE连接已关闭", username)
			return
		}
	}
}

// UsersHandler 获取在线用户列表
func (cs *ChatServer) UsersHandler(w http.ResponseWriter, r *http.Request) {
	cs.mutex.RLock()
	defer cs.mutex.RUnlock()

	onlineUsers := make([]*User, 0)
	for _, user := range cs.users {
		if user.IsOnline {
			onlineUsers = append(onlineUsers, user)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":      "success",
		"users":       onlineUsers,
		"total_count": len(onlineUsers),
	})
}

// HistoryHandler 获取消息历史
func (cs *ChatServer) HistoryHandler(w http.ResponseWriter, r *http.Request) {
	cs.mutex.RLock()
	defer cs.mutex.RUnlock()

	limit := 50 // 默认返回最近50条消息
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		fmt.Sscanf(limitStr, "%d", &limit)
	}

	start := len(cs.messages) - limit
	if start < 0 {
		start = 0
	}

	history := cs.messages[start:]

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "success",
		"messages": history,
		"count":    len(history),
		"total":    len(cs.messages),
	})
}

// StatusHandler 服务器状态
func (cs *ChatServer) StatusHandler(w http.ResponseWriter, r *http.Request) {
	cs.mutex.RLock()
	defer cs.mutex.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":        "online",
		"users_online":  len(cs.clients),
		"users_total":   len(cs.users),
		"messages_total": len(cs.messages),
		"timestamp":     time.Now(),
	})
}

func main() {
	chatServer := NewChatServer()

	// 设置路由
	http.HandleFunc("/join", chatServer.JoinHandler)
	http.HandleFunc("/leave", chatServer.LeaveHandler)
	http.HandleFunc("/send", chatServer.SendHandler)
	http.HandleFunc("/stream", chatServer.StreamHandler)
	http.HandleFunc("/users", chatServer.UsersHandler)
	http.HandleFunc("/history", chatServer.HistoryHandler)
	http.HandleFunc("/status", chatServer.StatusHandler)

	// 启动服务器
	port := ":8080"
	log.Printf("🚀 群聊服务器启动在 http://localhost%s", port)
	log.Printf("📝 API端点:")
	log.Printf("  POST /join - 加入聊天室")
	log.Printf("  POST /leave - 离开聊天室")
	log.Printf("  POST /send - 发送消息")
	log.Printf("  GET  /stream - 实时消息流 (SSE)")
	log.Printf("  GET  /users - 在线用户列表")
	log.Printf("  GET  /history - 消息历史")
	log.Printf("  GET  /status - 服务器状态")
	
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}