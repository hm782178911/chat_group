package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

// Config 配置文件结构
type Config struct {
	ServerURL   string `json:"server_url"`
	Username    string `json:"username"`
}

// Message 消息结构
type Message struct {
	Sender    string    `json:"sender"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"`
}

// loadConfig 加载配置文件
func loadConfig() (*Config, error) {
	file, err := os.Open("config.json")
	if err != nil {
		if os.IsNotExist(err) {
			defaultConfig := &Config{
				ServerURL: "http://localhost:8080",
				Username:  "匿名用户",
			}
			saveConfig(defaultConfig)
			return defaultConfig, nil
		}
		return nil, err
	}
	defer file.Close()

	var config Config
	err = json.NewDecoder(file).Decode(&config)
	return &config, err
}

func saveConfig(config *Config) error {
	file, err := os.Create("config.json")
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(config)
}

// joinChat 加入聊天室
func joinChat(serverURL, username string) error {
	data := url.Values{}
	data.Set("username", username)

	resp, err := http.PostForm(serverURL+"/join", data)
	if err != nil {
		return fmt.Errorf("网络错误: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	// 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusConflict {
			return fmt.Errorf("用户名 '%s' 已存在，请选择其他用户名", username)
		}
		return fmt.Errorf("服务器错误: %s", resp.Status)
	}

	var response map[string]interface{}
	json.Unmarshal(body, &response)

	if response["status"] == "success" {
		fmt.Printf("✅ 成功加入聊天室 as %s\n", username)
	} else {
		return fmt.Errorf("加入失败: %v", response["message"])
	}
	return nil
}

// leaveChat 离开聊天室
func leaveChat(serverURL, username string) error {
	data := url.Values{}
	data.Set("username", username)

	resp, err := http.PostForm(serverURL+"/leave", data)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// sendMessage 发送消息
func sendMessage(serverURL, sender, content string) error {
	data := url.Values{}
	data.Set("sender", sender)
	data.Set("content", content)

	resp, err := http.PostForm(serverURL+"/send", data)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// startRealTimeChat 启动实时群聊
func startRealTimeChat(serverURL, username string) {
	// 先加入聊天室
	if err := joinChat(serverURL, username); err != nil {
		fmt.Printf("❌ 加入失败: %v\n", err)
		return
	}
	defer leaveChat(serverURL, username)

	fmt.Printf("\n🚀 进入群聊模式（用户: %s）\n", username)
	fmt.Println("💬 输入消息开始聊天")
	fmt.Println("👥 输入 '/users' 查看在线用户")
	fmt.Println("📜 输入 '/history' 查看历史消息")
	fmt.Println("❌ 输入 'exit' 退出聊天")
	fmt.Println("🔔 开始接收消息...\n")

	// 启动SSE连接
	events := make(chan Message)
	go func() {
		resp, err := http.Get(serverURL + "/stream?user=" + url.QueryEscape(username))
		if err != nil {
			fmt.Printf("❌ 连接失败: %v\n", err)
			return
		}
		defer resp.Body.Close()

		reader := bufio.NewReader(resp.Body)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				fmt.Printf("❌ 连接断开: %v\n", err)
				return
			}

			if strings.HasPrefix(line, "data: ") {
				var msg Message
				json.Unmarshal([]byte(line[6:]), &msg)
				events <- msg
			}
		}
	}()

	// 处理信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 处理输入
	inputChan := make(chan string)
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			input := strings.TrimSpace(scanner.Text())
			inputChan <- input
		}
	}()

	for {
		select {
		case msg := <-events:
			timestamp := msg.Timestamp.Format("15:04:05")
			switch msg.Type {
			case "join", "leave":
				fmt.Printf("📢 [%s] %s\n", timestamp, msg.Content)
			default:
				fmt.Printf("💬 [%s] %s: %s\n", timestamp, msg.Sender, msg.Content)
			}

		case input := <-inputChan:
			if input == "exit" {
				fmt.Println("👋 退出聊天")
				return
			}

			switch input {
			case "/users":
				resp, err := http.Get(serverURL + "/users")
				if err == nil {
					var result map[string]interface{}
					json.NewDecoder(resp.Body).Decode(&result)
					if users, ok := result["users"].([]interface{}); ok {
						fmt.Println("👥 在线用户:")
						for _, user := range users {
							if u, ok := user.(map[string]interface{}); ok {
								fmt.Printf("  • %s\n", u["name"])
							}
						}
					}
				}

			case "/history":
				resp, err := http.Get(serverURL + "/history?limit=10")
				if err == nil {
					var result map[string]interface{}
					json.NewDecoder(resp.Body).Decode(&result)
					if messages, ok := result["messages"].([]interface{}); ok {
						fmt.Println("📜 最近消息:")
						for _, msg := range messages {
							if m, ok := msg.(map[string]interface{}); ok {
								ts, _ := time.Parse(time.RFC3339, m["timestamp"].(string))
								fmt.Printf("  [%s] %s: %s\n", 
									ts.Format("15:04"), m["sender"].(string), m["content"].(string))
							}
						}
					}
				}

			default:
				if input != "" {
					if err := sendMessage(serverURL, username, input); err != nil {
						fmt.Printf("❌ 发送失败: %v\n", err)
					}
				}
			}

		case <-sigChan:
			fmt.Println("\n👋 收到退出信号")
			return
		}
	}
}

func main() {
	config, err := loadConfig()
	if err != nil {
		fmt.Printf("❌ 加载配置失败: %v\n", err)
		os.Exit(1)
	}

	// if len(os.Args) < 2 {
	// 	fmt.Println("使用方法: chat start [用户名]")
	// 	os.Exit(1)
	// }

	// if os.Args[1] == "start" {
	// 	username := config.Username
	// 	if len(os.Args) >= 3 {
	// 		username = os.Args[2]
	// 		config.Username = username
	// 		saveConfig(config)
	// 	}
	// 	startRealTimeChat(config.ServerURL, username)
	// } else {
	// 	fmt.Println("未知命令，使用: chat start [用户名]")
	// }

	command := os.Args[1]

	switch command {
	case "start":
		username := config.Username
		if len(os.Args) >= 3 {
			username = os.Args[2]
			config.Username = username
			saveConfig(config)
		}
		startRealTimeChat(config.ServerURL, username)

	case "set-server":
		if len(os.Args) < 3 {
			color.Red("使用方法: chat set-server <服务器地址>")
			os.Exit(1)
		}
		config.ServerURL = os.Args[2]
		err = saveConfig(config)
		if err == nil {
			color.Green("✓ 服务器地址已设置为: %s", config.ServerURL)
		}

	default:
		color.Red("未知命令: %s", command)
		os.Exit(1)
	}
}