package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/xiaoming-software/p2pssh-server/internal/callback"
	"github.com/xiaoming-software/p2pssh-server/internal/webrpc"
)

func usage() {
	fmt.Fprintf(os.Stderr, `P2P SSH 转发服务端（webrpc）

只负责把对端 SSH 隧道流量转发到本机 SSH 端口，不做鉴权与终端。

用法:
  p2pssh-server --token TOKEN --passwd PASSWD --permission PERMISSION [选项]

参数:
  --token         webrpc token（必填）
  --passwd        webrpc token 密码（必填）
  --permission    webrpc 口令，对端 OpenSession 时需提供（必填）
  --ssh-host      本机 SSH 监听地址（默认 127.0.0.1）
  --ssh-port      本机 SSH 端口（默认 22）
`)
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
	log.SetPrefix("p2pssh-server ")

	token := flag.String("token", "", "webrpc token")
	passwd := flag.String("passwd", "", "webrpc password")
	permission := flag.String("permission", "", "webrpc permission")
	sshHost := flag.String("ssh-host", "127.0.0.1", "local SSH host")
	sshPort := flag.Int("ssh-port", 22, "local SSH port")
	flag.Usage = usage
	flag.Parse()

	if *token == "" || *passwd == "" || *permission == "" {
		usage()
		os.Exit(2)
	}
	if *sshPort <= 0 || *sshPort > 65535 {
		log.Fatalf("非法 ssh-port: %d", *sshPort)
	}
	if *sshHost == "" {
		*sshHost = "127.0.0.1"
	}

	client, err := webrpc.Open(*token, *passwd, *permission)
	if err != nil {
		log.Fatalf("webrpc: %v", err)
	}
	defer client.Close()

	log.Printf("webrpc 客户端已创建，等待登录...")
	if err := client.WaitLogin(2*time.Second, 30*time.Second); err != nil {
		log.Fatal(err)
	}
	log.Printf("webrpc 登录成功 status=%d", client.LoginStatus())

	tcpPort := client.ReceivePort()
	if tcpPort <= 0 {
		log.Fatalf("无效的回调端口: %d", tcpPort)
	}

	mgr := callback.NewManager(*sshHost, *sshPort, func(sessionID uint32, data []byte) int {
		return client.SendData(sessionID, data, 5000)
	})
	defer mgr.CloseAll()

	go callback.Listen(tcpPort, mgr)

	log.Printf("转发目标(tcp模式): %s:%d", *sshHost, *sshPort)
	log.Printf("回调端口: %d", tcpPort)
	log.Printf("服务已就绪：应用内终端走 shell PTY；外部 ssh -p 走本机端口转发")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-sig:
			log.Println("正在退出")
			return
		case <-ticker.C:
			log.Printf("当前会话数: %d", client.SessionSize())
		}
	}
}
