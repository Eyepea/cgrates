/*
Real-time Charging System for Telecom & ISP environments
Copyright (C) 2012-2015 ITsysCOM GmbH

This program is free software: you can Storagetribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITH*out ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>
*/
package utils

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"net/rpc/jsonrpc"
	"sync"

	"github.com/cenkalti/rpc2"
	"github.com/gorilla/websocket"

	_ "net/http/pprof"
)

type Server struct {
	rpcEnabled  bool
	httpEnabled bool
	bijsonSrv   *rpc2.Server
}

func (s *Server) RpcRegister(rcvr interface{}) {
	rpc.Register(rcvr)
	s.rpcEnabled = true
}

func (s *Server) RpcRegisterName(name string, rcvr interface{}) {
	rpc.RegisterName(name, rcvr)
	s.rpcEnabled = true
}

func (s *Server) RegisterHttpFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	http.HandleFunc(pattern, handler)
	s.httpEnabled = true
}

// Registers a new BiJsonRpc name
func (s *Server) BijsonRegisterName(method string, handlerFunc interface{}) {
	if s.bijsonSrv == nil {
		s.bijsonSrv = rpc2.NewServer()
	}
	s.bijsonSrv.Handle(method, handlerFunc)
}

//Registers a new handler for OnConnect event
func (s *Server) BijsonRegisterOnConnect(f func(*rpc2.Client)) {
	s.bijsonSrv.OnConnect(f)
}

//Registers a new handler for OnDisconnect event
func (s *Server) BijsonRegisterOnDisconnect(f func(*rpc2.Client)) {
	s.bijsonSrv.OnDisconnect(f)
}

func (s *Server) ServeJSON(addr string) {
	if !s.rpcEnabled {
		return
	}
	lJSON, e := net.Listen("tcp", addr)
	if e != nil {
		log.Fatal("ServeJSON listen error:", e)
	}
	Logger.Info(fmt.Sprintf("Starting CGRateS JSON server at %s.", addr))
	for {
		conn, err := lJSON.Accept()
		if err != nil {
			Logger.Err(fmt.Sprintf("<CGRServer> Accept error: %v", conn))
			continue
		}
		//utils.Logger.Info(fmt.Sprintf("<CGRServer> New incoming connection: %v", conn.RemoteAddr()))
		go jsonrpc.ServeConn(conn)
	}

}

func (s *Server) ServeGOB(addr string) {
	if !s.rpcEnabled {
		return
	}
	lGOB, e := net.Listen("tcp", addr)
	if e != nil {
		log.Fatal("ServeGOB listen error:", e)
	}
	Logger.Info(fmt.Sprintf("Starting CGRateS GOB server at %s.", addr))
	for {
		conn, err := lGOB.Accept()
		if err != nil {
			Logger.Err(fmt.Sprintf("<CGRServer> Accept error: %v", conn))
			continue
		}

		//utils.Logger.Info(fmt.Sprintf("<CGRServer> New incoming connection: %v", conn.RemoteAddr()))
		go rpc.ServeConn(conn)
	}
}

type WebsocketReadWriteCloser struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

func (wrwc *WebsocketReadWriteCloser) Read(p []byte) (n int, err error) {
	wrwc.mu.Lock()
	defer wrwc.mu.Unlock()
	_, r, err := wrwc.conn.NextReader()
	if err != nil {
		return 0, err
	}
	b := bytes.NewBuffer(p)
	read, err := io.Copy(b, r)
	log.Print("Read: ", string(b.Bytes()))
	return int(read), err
}

func (wrwc *WebsocketReadWriteCloser) Write(p []byte) (n int, err error) {
	wrwc.mu.Lock()
	defer wrwc.mu.Unlock()
	w, err := wrwc.conn.NextWriter(websocket.TextMessage)
	if err != nil {
		return 0, err
	}
	log.Print("Write: ", string(p))
	b := bytes.NewBuffer(p)
	written, err := io.Copy(w, b)
	if err != nil {
		return int(written), err
	}
	err = w.Close()
	return int(written), err
}

func (wrwc *WebsocketReadWriteCloser) Close() error {
	wrwc.mu.Lock()
	defer wrwc.mu.Unlock()
	return wrwc.conn.Close()
}

func (s *Server) ServeHTTP(addr string) {
	if s.rpcEnabled {
		http.HandleFunc("/jsonrpc", func(w http.ResponseWriter, req *http.Request) {
			defer req.Body.Close()
			w.Header().Set("Content-Type", "application/json")
			res := NewRPCRequest(req.Body).Call()
			io.Copy(w, res)
		})
		upgrader := websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		}

		http.HandleFunc("/ws", func(w http.ResponseWriter, req *http.Request) {
			ws, err := upgrader.Upgrade(w, req, nil)
			if err != nil {
				log.Println(err)
				return
			}
			defer ws.Close()
			jsonrpc.ServeConn(ws.UnderlyingConn())

			//wrapper := &WebsocketReadWriteCloser{conn: ws}
			//jsonrpc.ServeConn(wrapper)
		})
		s.httpEnabled = true
	}
	if !s.httpEnabled {
		return
	}
	Logger.Info(fmt.Sprintf("Starting CGRateS HTTP server at %s.", addr))
	http.ListenAndServe(addr, nil)
}

func (s *Server) ServeBiJSON(addr string) {
	if s.bijsonSrv == nil {
		return
	}
	lBiJSON, e := net.Listen("tcp", addr)
	if e != nil {
		log.Fatal("ServeBiJSON listen error:", e)
	}
	Logger.Info(fmt.Sprintf("Starting CGRateS BiJSON server at %s.", addr))
	s.bijsonSrv.Accept(lBiJSON)
}

// rpcRequest represents a RPC request.
// rpcRequest implements the io.ReadWriteCloser interface.
type rpcRequest struct {
	r    io.Reader     // holds the JSON formated RPC request
	rw   io.ReadWriter // holds the JSON formated RPC response
	done chan bool     // signals then end of the RPC request
}

// NewRPCRequest returns a new rpcRequest.
func NewRPCRequest(r io.Reader) *rpcRequest {
	var buf bytes.Buffer
	done := make(chan bool)
	return &rpcRequest{r, &buf, done}
}

func (r *rpcRequest) Read(p []byte) (n int, err error) {
	return r.r.Read(p)
}

func (r *rpcRequest) Write(p []byte) (n int, err error) {
	n, err = r.rw.Write(p)
	r.done <- true
	return
}

func (r *rpcRequest) Close() error {
	//r.done <- true // seem to be called sometimes before the write command finishes!
	return nil
}

// Call invokes the RPC request, waits for it to complete, and returns the results.
func (r *rpcRequest) Call() io.Reader {
	go jsonrpc.ServeConn(r)
	<-r.done
	return r.rw
}
