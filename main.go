package main

import (
	"io"
	"io/ioutil"
	"os"

	//"syscall"
	"fmt"
	"log"

	//"code.google.com/p/go.crypto/ssh"

	"net/http"

	"github.com/gorilla/websocket"
	//"code.google.com/p/go.crypto/ssh/terminal"
	"golang.org/x/crypto/ssh"
	//"github.com/fatih/color"
)

type password string

func (p password) Password(user string) (password string, err error) {
	return string(p), nil
}

func PublicKeyFile(file string) (ssh.AuthMethod, error) {
	buffer, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	key, err := ssh.ParsePrivateKey(buffer)
	if err != nil {
		return nil, err
	}
	return ssh.PublicKeys(key), nil
}

func init() {
	// Verbose logging with file name and line number
	log.SetFlags(log.Lshortfile)
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,

	// We'll need to check the origin of our connection
	// this will allow us to make requests from our React
	// development server to here.
	// For now, we'll do no checking and just allow any connection
	CheckOrigin: func(r *http.Request) bool { return true },
}

func reader(conn *websocket.Conn, w *io.PipeWriter) {
	for {
		// read in a message
		_, p, err := conn.NextReader()

		if err != nil {
			log.Println(err)
			return
		}
		if _, err = io.Copy(w, p); err != nil {
			fmt.Println(err)
		}
	}
}

func writer(conn *websocket.Conn, r *io.PipeReader) {
	for {
		w, err := conn.NextWriter(websocket.TextMessage)
		if err != nil {
			fmt.Println(err)
		}
		//buf := make([]byte, 100)
		//r.Read(buf)
		//fmt.Println(string(buf))
		_, err = io.CopyN(w, r, int64(1))
		if err != nil {
			fmt.Println(err)
		}
		w.Close()
		//r.Close()
	}
}

func serveWs(rr *io.PipeReader, wr *io.PipeWriter) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// upgrade this connection to a WebSocket
		// connection
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println(err)
		}
		// listen indefinitely for new messages coming
		// through on our WebSocket connection
		go writer(ws, rr)
		go reader(ws, wr)
	}
}

func startServer(r *io.PipeReader, w *io.PipeWriter) {

	go http.ListenAndServe(":8080", nil)

	http.HandleFunc("/ws", serveWs(r, w))
}

func runSSH(w *io.PipeWriter, r *io.PipeReader) {
	server := "10.0.127.230"
	port := "22"
	server = server + ":" + port
	user := "ec2-user"

	publicKey, err := PublicKeyFile(`D:\leadschool\ssh-keys\AWS\mykeys\jumpbox\vpc#01-jumpbox-root-key.pem`)
	if err != nil {
		log.Println(err)
		return
	}

	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			//ssh.Password(p),
			publicKey,
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	client, err := ssh.Dial("tcp", server, config)
	if err != nil {
		fmt.Println(err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		fmt.Println(err)
	}
	defer session.Close()

	session.Stdout = w
	session.Stderr = w
	session.Stdin = r

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,     // enable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}

	// fileDescriptor := int(os.Stdin.Fd())

	// if terminal.IsTerminal(fileDescriptor) {
	// 	originalState, err := terminal.MakeRaw(fileDescriptor)
	// 	if err != nil {
	// 		fmt.Println(err)
	// 	}
	// 	defer terminal.Restore(fileDescriptor, originalState)

	// 	termWidth, termHeight, err := terminal.GetSize(fileDescriptor)
	// 	if err != nil {
	// 		fmt.Println("HRTR",err)
	// 	}

	// 	err = session.RequestPty("xterm", termHeight, termWidth, modes)
	// 	if err != nil {
	// 		fmt.Println(err)
	// 	}
	// }l

	err = session.RequestPty("xterm", 1600, 500, modes)
	if err != nil {
		fmt.Println(err)
	}

	err = session.Shell()
	if err != nil {
		fmt.Println(err)
	}

	// You should now be connected via SSH with a fully-interactive terminal
	// This call blocks until the user exits the session (e.g. via CTRL + D)
	fmt.Println("ssh wait")
	session.Wait()
}

func dummyreader(rr *io.PipeReader, wr *io.PipeWriter) {
	go io.Copy(os.Stdout, rr)
	go io.Copy(wr, os.Stdin)
}

func main() {
	inr, inw := io.Pipe()
	outr, outw := io.Pipe()
	startServer(inr, outw)
	//time.Sleep(10 * time.Second)
	//go dummyreader(inr, outw)
	runSSH(inw, outr)
}
