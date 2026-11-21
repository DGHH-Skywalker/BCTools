package launcher

import (
	"errors"
	"io"
	"net"
	"time"
)

var ErrSwitching = errors.New("switching to desktop")

func serveControl(l net.Listener, mode string, commands chan<- string) {
	for {
		conn, err := l.Accept()
		if err != nil {
			return
		}
		go func() {
			defer conn.Close()
			_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
			var command [1]byte
			if _, err := io.ReadFull(conn, command[:]); err != nil {
				return
			}
			if mode == "update" {
				// The helper owns the data directory while no UI or HTTP backend is running.
				return
			}
			switch command[0] {
			case 's':
				_, _ = conn.Write([]byte{'o'})
			case 'q':
				select {
				case commands <- "shutdown":
					_, _ = conn.Write([]byte{'o'})
				default:
				}
			case 'd':
				request, reply := "activate", byte('o')
				if mode == "browser" {
					request, reply = "shutdown", 'r'
				}
				select {
				case commands <- request:
					_, _ = conn.Write([]byte{reply})
				default:
				}
			}
		}()
	}
}

func sendControl(conn net.Conn, command string) error {
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
	op := map[string]byte{"status": 's', "shutdown": 'q', "desktop": 'd'}[command]
	if op == 0 {
		return errors.New("invalid control command")
	}
	if _, err := conn.Write([]byte{op}); err != nil {
		return err
	}
	var status [1]byte
	if _, err := io.ReadFull(conn, status[:]); err != nil {
		return err
	}
	switch status[0] {
	case 'o':
		return nil
	case 'r':
		return ErrSwitching
	default:
		return errors.New("BCTools did not accept the command")
	}
}

type ownership struct {
	net.Listener
	lock io.Closer
}

func (o *ownership) Close() error {
	return errors.Join(o.Listener.Close(), o.lock.Close())
}
