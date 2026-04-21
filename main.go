package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {

	listener, err := net.Listen("tcp", "localhost:8000")
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Server started: Listening on localhost:8000")
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("connect failure", err)
			continue
		}
		go handleConn2(conn)
	}

	/*app, err := config.NewApplication("conf.yaml")

	if err != nil {
		return
	}

	db := app.DB
	ctx := context.Background()
	u := entity.User{}
	db.WithContext(ctx).Find(&u, "name = ?", "test1")
	fmt.Println(u)
	ctx := context.Background()
	db := config.DB
	db.AutoMigrate(&entity.User{})
	err := gorm.G[entity.User](db).Create(ctx, &entity.User{Name: "test1", Age: 18, Gender: "male", Email: "test1@gmail.com"})
	if err != nil {
		log.Fatal(err)
		panic("failed to create user")

	}
	defer app.Close()*/

}

func echo(c net.Conn, shout string, timeout time.Duration) {
	fmt.Fprintln(c, "\t", strings.ToUpper(shout))
	time.Sleep(timeout)
	fmt.Fprintln(c, "\t", shout)
	time.Sleep(timeout)
	fmt.Fprintln(c, "\t", strings.ToLower(shout))

}

type client struct {
	conn    io.ReadWriteCloser
	writer  *bufio.Writer
	scanner *bufio.Scanner
	dir     string //当前工作目录
}

func handleConn2(c net.Conn) {
	defer c.Close()
	input := bufio.NewScanner(c)
	for input.Scan() {
		go echo(c, input.Text(), 1*time.Second)
	}

}
func handleConn(c io.ReadWriteCloser) {

	defer c.Close()
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}
	cli := &client{
		conn:    c,
		writer:  bufio.NewWriter(c),
		scanner: bufio.NewScanner(c),
		dir:     cwd,
	}
	cli.send("欢迎来到 Go FTP 服务器\n支持命令：ls, cd [dir], get [file], send [file], close\n")

	for cli.scanner.Scan() {
		cmdline := cli.scanner.Text()
		args := strings.Fields(cmdline)
		if len(args) == 0 {
			continue
		}
		switch args[0] {
		case "ls":
			cli.ls()
		case "cd":
			if len(args) >= 2 {
				cli.cd(args[1])
			} else {
				cli.send("用法：cd 目录名\n")
			}
		case "get":
			if len(args) >= 2 {
				cli.getFile(args[1])
			} else {
				cli.send("用法：get 文件名\n")
			}
		case "send":
			if len(args) > 1 {
				cli.sendFile(args[1])
			} else {
				cli.send("用法：send 文件名\n")
			}
		case "close":
			cli.send("bye\n")
			return
		default:
			cli.send("未知命令：" + args[0] + "\n")
		}
	}
}

func (cli *client) send(msg string) {
	_, err := cli.writer.WriteString(msg)
	if err != nil {
		log.Fatal(err)
	}
	err = cli.writer.Flush()
	if err != nil {
		log.Fatal(err)
	}
}
func handleBrowser(c net.Conn) {
	defer func(c net.Conn) {
		err := c.Close()
		if err != nil {
			log.Printf("Failed to close connection: %v", err)
		}
	}(c)
	buf := make([]byte, 1024)
	_, _ = c.Read(buf)
	now := time.Now().Format("15:04:05\n")
	response := "HTTP/1.1 200 OK\r\n" +
		"Content-Type: text/html; charset=utf-8\r\n" +
		"\r\n" + // 空行分隔头和内容
		"<h1>当前服务器时间</h1>" +
		"<p>" + now + "</p>"
	_, _ = io.WriteString(c, response)
}

func (cli *client) ls() {
	files, err := os.ReadDir(cli.dir)
	if err != nil {
		cli.send("ls error: " + err.Error() + "\n")
		return
	}
	for _, file := range files {
		cli.send(file.Name() + "\t")
		if file.IsDir() {
			cli.send("(目录)\n")
		} else {
			cli.send("(文件)\n")
		}
	}
}

func (cli *client) cd(path string) {
	target := filepath.Join(cli.dir, path)
	stat, err := os.Stat(target)
	if err != nil {
		cli.send("dir not exist: " + err.Error() + "\n")
		return
	}
	if !stat.IsDir() {
		cli.send("not a dir: " + target + "\n")
		return
	}
	cli.dir = target
	cli.send("cd to " + cli.dir + "\n")
}

func (cli *client) getFile(fileName string) {
	file, err := os.Open(filepath.Join(cli.dir, fileName))
	if err != nil {
		cli.send("cannot open target file: " + err.Error() + "\n")
		return
	}
	defer file.Close()
	cli.send("begin downloading file: " + fileName + "\n")
	_, _ = io.Copy(cli.writer, file)
	cli.send("file transfer complete\n")
}

func (cli *client) sendFile(fileName string) {
	file, err := os.Create(filepath.Join(cli.dir, fileName))
	if err != nil {
		cli.send("cannot create target file: " + err.Error() + "\n")
		return
	}
	defer file.Close()
	cli.send("ready to receive file")
	cli.writer.Flush()
	cli.send("begin uploading file: " + fileName + "\n")
	_, _ = io.Copy(file, cli.conn)
	cli.send("file transfer complete\n")
}
