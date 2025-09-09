# a chat-apps based on golang

## 0.Description

local online-chatroom


## 1.quick start

```bash
#server
./chat_server_linux_arm64


#client
cd ./chat_group_client
./chat_client_windows_x86.exe set-server http://your_server_ip:8080

./chat_group_client/chat_client_windows_x86.exe start user_name

```

## 2.build server with docke-compose

```bash

#server
cd chat_group_server_docker
sudo docker-compose up --build

#client
cd ./chat_group_client
./chat_client_windows_x86.exe set-server http://your_server_ip:8080

./chat_group_client/chat_client_windows_x86.exe start user_name

```