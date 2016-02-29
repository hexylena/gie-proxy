package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
)

func shouldUpgradeWebsocket(r *http.Request) bool {
	connHdr := ""
	connHdrs := r.Header["Connection"]
	if len(connHdrs) > 0 {
		connHdr = connHdrs[0]
	}
	upgradeWebsocket := false
	if strings.ToLower(connHdr) == "upgrade" {
		upgradeHdrs := r.Header["Upgrade"]
		if len(upgradeHdrs) > 0 {
			upgradeWebsocket = (strings.ToLower(upgradeHdrs[0]) == "websocket")
		}
	}
	return upgradeWebsocket
}

func plumbWebsocket(w http.ResponseWriter, r *http.Request, route **Route) error {
	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "webserver doesn't support hijacking", http.StatusInternalServerError)
		return errors.New("no-hijack")
	}
	conn, bufrw, err := hj.Hijack()
	conn2, err := net.Dial("tcp", r.URL.Host)
	if err != nil {
		http.Error(w, "couldn't connect to backend server", http.StatusServiceUnavailable)
		return errors.New("dead-backend")
	}
	err = r.Write(conn2)
	if err != nil {
		log.Warning("writing WebSocket request to backend server failed: %v", err)
		return errors.New("dead-backend")
	}
	CopyBidir(conn, bufrw, conn2, bufio.NewReadWriter(bufio.NewReader(conn2), bufio.NewWriter(conn2)), route)
	err = conn.Close()

	if err != nil {
		log.Warning("Could not close stream", err)
	}
	err = conn2.Close()
	if err != nil {
		log.Warning("Could not close stream", err)
	}
	return nil
}

func plumbHTTP(h *requestHandler, w http.ResponseWriter, r *http.Request, route **Route) error {
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
	(*route).Seen()
	w.WriteHeader(resp.StatusCode)
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		log.Warning("Could not copy bytes", err)
	}
	err = resp.Body.Close()
	if err != nil {
		log.Warning("Could not close response", err)
	}
	return nil
}

// Copy from src buffer to destination buffer. One way.
func Copy(dest *bufio.ReadWriter, src *bufio.ReadWriter, route **Route) {
	buf := make([]byte, 40*1024)
	for {
		n, err := src.Read(buf)
		if err != nil && err != io.EOF {
			//log.Error("Read failed: %v", err)
			return
		}
		if n == 0 {
			return
		}
		(*route).Seen()
		_, err = dest.Write(buf[0:n])
		if err != nil && err != io.EOF {
			log.Warning("Could not write to dest", err)
		}
		err = dest.Flush()

		if err != nil && err != io.EOF {
			log.Warning("Could not flush to dest", err)
		}
	}
}

// CopyBidir copies the first buffer to the second and vice versa.
func CopyBidir(conn1 io.ReadWriteCloser, rw1 *bufio.ReadWriter, conn2 io.ReadWriteCloser, rw2 *bufio.ReadWriter, route **Route) {
	finished := make(chan bool)
	go func() {
		Copy(rw2, rw1, route)
		_ = conn2.Close()
		finished <- true
	}()
	go func() {
		Copy(rw1, rw2, route)
		_ = conn1.Close()
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
