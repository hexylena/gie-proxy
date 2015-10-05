package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
)

func shouldUpgradeWebsocket(r *http.Request) bool {
	connHdr := ""
	connHdrs := r.Header["Connection"]
	log.Printf("Connection headers: %v", connHdrs)
	if len(connHdrs) > 0 {
		connHdr = connHdrs[0]
	}
	upgradeWebsocket := false
	if strings.ToLower(connHdr) == "upgrade" {
		log.Printf("got Connection: Upgrade")
		upgradeHdrs := r.Header["Upgrade"]
		log.Printf("Upgrade headers: %v", upgradeHdrs)
		if len(upgradeHdrs) > 0 {
			upgradeWebsocket = (strings.ToLower(upgradeHdrs[0]) == "websocket")
		}
	}
	return upgradeWebsocket
}

func plumbWebsocket(w http.ResponseWriter, r *http.Request) error {
	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "webserver doesn't support hijacking", http.StatusInternalServerError)
		return errors.New("no-hijack")
	}
	conn, bufrw, err := hj.Hijack()
	defer conn.Close()
	conn2, err := net.Dial("tcp", r.URL.Host)
	if err != nil {
		http.Error(w, "couldn't connect to backend server", http.StatusServiceUnavailable)
		return errors.New("dead-backend")
	}
	defer conn2.Close()
	err = r.Write(conn2)
	if err != nil {
		log.Printf("writing WebSocket request to backend server failed: %v", err)
		return errors.New("dead-backend")
	}
	CopyBidir(conn, bufrw, conn2, bufio.NewReadWriter(bufio.NewReader(conn2), bufio.NewWriter(conn2)))
	return nil
}

func plumbHTTP(h *requestHandler, w http.ResponseWriter, r *http.Request) error {
	resp, err := h.Transport.RoundTrip(r)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(w, "Error: %v", err)
		return errors.New("dead-backend")
	}
	for k, v := range resp.Header {
		for _, vv := range v {
			w.Header().Add(k, vv)
		}
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
	resp.Body.Close()
	return nil
}

// Copy from src buffer to destination buffer. One way.
func Copy(dest *bufio.ReadWriter, src *bufio.ReadWriter) {
	buf := make([]byte, 40*1024)
	for {
		n, err := src.Read(buf)
		if err != nil && err != io.EOF {
			log.Printf("Read failed: %v", err)
			return
		}
		if n == 0 {
			return
		}
		dest.Write(buf[0:n])
		dest.Flush()
	}
}

// CopyBidir copies the first buffer to the second and vice versa.
func CopyBidir(conn1 io.ReadWriteCloser, rw1 *bufio.ReadWriter, conn2 io.ReadWriteCloser, rw2 *bufio.ReadWriter) {
	finished := make(chan bool)
	go func() {
		Copy(rw2, rw1)
		conn2.Close()
		finished <- true
	}()
	go func() {
		Copy(rw1, rw2)
		conn1.Close()
		finished <- true
	}()
	<-finished
	<-finished
}

func addForwardedFor(r *http.Request) {
	remoteAddr := r.RemoteAddr
	idx := strings.LastIndex(remoteAddr, ":")
	if idx != -1 {
		remoteAddr = remoteAddr[0:idx]
		if remoteAddr[0] == '[' && remoteAddr[len(remoteAddr)-1] == ']' {
			remoteAddr = remoteAddr[1 : len(remoteAddr)-1]
		}
	}
	r.Header.Add("X-Forwarded-For", remoteAddr)
}
