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
	"runtime"
	"strings"
	"syscall"
	"time"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// Config 配置文件结构
type Config struct {
	ServerURL string `json:"server_url"`
	Username  string `json:"username"`
}

// Message 消息结构
type Message struct {
	Sender    string    `json:"sender"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"`
}

// ---------------- 通用配置加载 ----------------
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

// ---------------- CLI 模式（Linux/macOS） ----------------
func joinChat(serverURL, username string) error {
	data := url.Values{}
	data.Set("username", username)

	resp, err := http.PostForm(serverURL+"/join", data)
	if err != nil {
		return fmt.Errorf("网络错误: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
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

func leaveChat(serverURL, username string) error {
	data := url.Values{}
	data.Set("username", username)
	_, _ = http.PostForm(serverURL+"/leave", data)
	return nil
}

func sendMessage(serverURL, sender, content string) error {
	data := url.Values{}
	data.Set("sender", sender)
	data.Set("content", content)

	_, err := http.PostForm(serverURL+"/send", data)
	return err
}

func startRealTimeChat(serverURL, username string) {
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

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

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
			if input != "" {
				if err := sendMessage(serverURL, username, input); err != nil {
					fmt.Printf("❌ 发送失败: %v\n", err)
				}
			}

		case <-sigChan:
			fmt.Println("\n👋 收到退出信号")
			return
		}
	}
}

// ---------------- Windows GUI 模式 ----------------

// 辅助函数：添加带颜色的消息
func addMessage(container *fyne.Container, text string, col color.Color) {
	label := canvas.NewText(text, col)
	label.TextStyle = fyne.TextStyle{}
	container.Add(label)
	container.Refresh()
}

func startGUIChat(serverURL, username string) {
	a := app.New()
	w := a.NewWindow("聊天室 - " + username)

	// 聊天记录容器
	chatVBox := container.NewVBox()
	chatScroll := container.NewVScroll(chatVBox)
	chatScroll.SetMinSize(fyne.NewSize(580, 350))

	// 输入框
	input := widget.NewEntry()
	input.SetPlaceHolder("输入消息...")

	// 发送按钮
	sendBtn := widget.NewButton("发送", func() {
		msg := strings.TrimSpace(input.Text)
		if msg != "" {
			if err := sendMessage(serverURL, username, msg); err != nil {
				addMessage(chatVBox, fmt.Sprintf("❌ 发送失败: %v", err), color.RGBA{255, 0, 0, 255})
			} else {
				input.SetText("")
			}
		}
	})

	// 回车发送
	input.OnSubmitted = func(text string) {
		sendBtn.OnTapped()
	}

	// 底部输入栏
	inputBar := container.NewBorder(nil, nil, nil, sendBtn, input)

	w.SetContent(container.NewBorder(nil, inputBar, nil, nil, chatScroll))
	w.Resize(fyne.NewSize(600, 500))

	// 消息 channel
	msgChan := make(chan Message, 50)

	// 接收消息
	go func() {
		resp, err := http.Get(serverURL + "/stream?user=" + url.QueryEscape(username))
		if err != nil {
			msgChan <- Message{Sender: "系统", Content: fmt.Sprintf("❌ 连接失败: %v", err), Timestamp: time.Now(), Type: "system"}
			close(msgChan)
			return
		}
		defer resp.Body.Close()

		reader := bufio.NewReader(resp.Body)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err != io.EOF {
					msgChan <- Message{Sender: "系统", Content: fmt.Sprintf("❌ 连接断开: %v", err), Timestamp: time.Now(), Type: "system"}
				}
				close(msgChan)
				return
			}
			if strings.HasPrefix(line, "data: ") {
				var msg Message
				if err := json.Unmarshal([]byte(line[6:]), &msg); err == nil {
					msgChan <- msg
				}
			}
		}
	}()

	// 定时器在主线程更新 UI
	ticker := time.NewTicker(100 * time.Millisecond)
	go func() {
		for range ticker.C {
			select {
			case msg, ok := <-msgChan:
				if !ok {
					ticker.Stop()
					return
				}
				var textColor color.Color
				var prefix string
				switch msg.Type {
				case "join":
					textColor = color.RGBA{0, 128, 0, 255}
					prefix = "👥 "
				case "leave":
					textColor = color.RGBA{128, 0, 0, 255}
					prefix = "👋 "
				case "system":
					textColor = color.RGBA{255, 0, 0, 255}
					prefix = "⚠️ "
				default:
					textColor = color.RGBA{0, 0, 0, 255}
					prefix = "💬 "
				}
				display := fmt.Sprintf("[%s] %s%s: %s", msg.Timestamp.Format("15:04:05"), prefix, msg.Sender, msg.Content)
				addMessage(chatVBox, display, textColor)
			default:
			}
		}
	}()

	w.ShowAndRun()
}



// ---------------- 主入口 ----------------
func main() {
	config, err := loadConfig()
	if err != nil {
		fmt.Printf("❌ 加载配置失败: %v\n", err)
		os.Exit(1)
	}

	username := config.Username
	if len(os.Args) >= 3 {
		username = os.Args[2]
		config.Username = username
		saveConfig(config)
	}

	command :="start"
	if len(os.Args) >= 2 {
		command = os.Args[1]
	}
	
	// 根据系统决定模式
	// if runtime.GOOS == "windows" {
	// 	startGUIChat(config.ServerURL, username)
	// } else {
	// 	startRealTimeChat(config.ServerURL, username)
	// }


	switch command {
	case "start":
		username := config.Username
		if len(os.Args) >= 3 {
			username = os.Args[2]
			config.Username = username
			saveConfig(config)
		}
		// startRealTimeChat(config.ServerURL, username)
		if runtime.GOOS == "windows" {
			startGUIChat(config.ServerURL, username)
		} else {
			startRealTimeChat(config.ServerURL, username)
		}

	case "set-server":
		if len(os.Args) < 3 {
			fmt.Println("使用方法: chat set-server <服务器地址>")
			os.Exit(1)
		}
		config.ServerURL = os.Args[2]
		err = saveConfig(config)
		if err == nil {
			fmt.Println("✓ 服务器地址已设置为: %s", config.ServerURL)
		}

	default:
		fmt.Println("未知命令: %s", command)
		os.Exit(1)
	}
}
