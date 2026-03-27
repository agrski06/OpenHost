// Package a2s implements the Valve A2S_INFO query protocol used by Source
// Engine games (Valheim, ARK, Rust, etc.) for server status checking.
// This is the reusable core extracted from the Valheim-specific checker.
package a2s

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/openhost/cli/internal/gamestatus"
)

const (
	infoHeader  = 0x54
	infoReply   = 0x49
	challengeID = 0x41
)

var baseInfoRequest = []byte{
	0xFF, 0xFF, 0xFF, 0xFF,
	infoHeader,
	'S', 'o', 'u', 'r', 'c', 'e', ' ', 'E', 'n', 'g', 'i', 'n', 'e', ' ', 'Q', 'u', 'e', 'r', 'y', 0x00,
}

// Options configures an A2S query.
type Options struct {
	// QueryPort is the UDP port to send the A2S query to.
	QueryPort int

	// Timeout is the maximum time to wait for a response.
	Timeout time.Duration

	// Dial overrides the default net.DialTimeout for testing.
	Dial func(network, address string, timeout time.Duration) (net.Conn, error)
}

// Response holds the parsed A2S_INFO response.
type Response struct {
	Name       string
	Map        string
	Players    int
	MaxPlayers int
	Version    string
}

// Query sends an A2S_INFO query to the target and returns a gamestatus.Status.
// It handles the challenge-response handshake automatically.
func Query(target gamestatus.Target, opts Options) (*gamestatus.Status, error) {
	if opts.Timeout == 0 {
		opts.Timeout = 2 * time.Second
	}
	dial := opts.Dial
	if dial == nil {
		dial = net.DialTimeout
	}

	port := opts.QueryPort
	if port == 0 && len(target.Ports) > 0 {
		port = target.Ports[0].From
	}
	if port == 0 {
		return &gamestatus.Status{
			State:  gamestatus.StateUnknown,
			Detail: "A2S query skipped: no port configured",
		}, nil
	}

	if target.PublicIP == "" {
		return &gamestatus.Status{
			State:  gamestatus.StateUnknown,
			Detail: "A2S query skipped: no public IP available",
		}, nil
	}

	address := net.JoinHostPort(target.PublicIP, strconv.Itoa(port))
	conn, err := dial("udp", address, opts.Timeout)
	if err != nil {
		return &gamestatus.Status{
			State:  gamestatus.StateUnreachable,
			Detail: fmt.Sprintf("A2S dial failed: %v", err),
		}, nil
	}
	defer func() { _ = conn.Close() }()

	_ = conn.SetDeadline(time.Now().Add(opts.Timeout))

	response, err := queryInfo(conn)
	if err != nil {
		if isTimeout(err) {
			return &gamestatus.Status{
				State:  gamestatus.StateUnreachable,
				Detail: fmt.Sprintf("A2S query timed out for %s", address),
			}, nil
		}
		return &gamestatus.Status{
			State:  gamestatus.StateQueryFailed,
			Detail: fmt.Sprintf("A2S query failed: %v", err),
		}, nil
	}

	players := response.Players
	return &gamestatus.Status{
		State:       gamestatus.StateReady,
		Detail:      fmt.Sprintf("A2S query succeeded: %s (%d/%d players, map=%s, version=%s)", response.Name, response.Players, response.MaxPlayers, response.Map, response.Version),
		Reachable:   true,
		PlayerCount: &players,
	}, nil
}

func queryInfo(conn net.Conn) (*Response, error) {
	payload, err := sendInfoRequest(conn, baseInfoRequest)
	if err != nil {
		return nil, err
	}

	if len(payload) < 5 {
		return nil, fmt.Errorf("short A2S response")
	}
	if !bytes.Equal(payload[:4], []byte{0xFF, 0xFF, 0xFF, 0xFF}) {
		return nil, fmt.Errorf("invalid A2S header")
	}

	switch payload[4] {
	case infoReply:
		return parseInfoResponse(payload[5:])
	case challengeID:
		if len(payload) < 9 {
			return nil, fmt.Errorf("short A2S challenge response")
		}
		request := append(append([]byte{}, baseInfoRequest...), payload[5:9]...)
		payload, err = sendInfoRequest(conn, request)
		if err != nil {
			return nil, err
		}
		if len(payload) < 5 || payload[4] != infoReply {
			return nil, fmt.Errorf("unexpected A2S response after challenge")
		}
		return parseInfoResponse(payload[5:])
	default:
		return nil, fmt.Errorf("unexpected A2S response type %#x", payload[4])
	}
}

func sendInfoRequest(conn net.Conn, request []byte) ([]byte, error) {
	if _, err := conn.Write(request); err != nil {
		return nil, err
	}

	buffer := make([]byte, 1400)
	n, err := conn.Read(buffer)
	if err != nil {
		return nil, err
	}
	return buffer[:n], nil
}

func parseInfoResponse(payload []byte) (*Response, error) {
	reader := &byteReader{data: payload}
	if _, err := reader.readByte(); err != nil { // protocol version
		return nil, err
	}

	name, err := reader.readString()
	if err != nil {
		return nil, fmt.Errorf("read server name: %w", err)
	}
	mapName, err := reader.readString()
	if err != nil {
		return nil, fmt.Errorf("read map name: %w", err)
	}
	if _, err := reader.readString(); err != nil { // folder
		return nil, fmt.Errorf("read folder: %w", err)
	}
	if _, err := reader.readString(); err != nil { // game name
		return nil, fmt.Errorf("read game name: %w", err)
	}
	if _, err := reader.readUint16(); err != nil { // app id
		return nil, fmt.Errorf("read app id: %w", err)
	}
	players, err := reader.readByte()
	if err != nil {
		return nil, fmt.Errorf("read players: %w", err)
	}
	maxPlayers, err := reader.readByte()
	if err != nil {
		return nil, fmt.Errorf("read max players: %w", err)
	}
	if _, err := reader.readByte(); err != nil { // bots
		return nil, fmt.Errorf("read bots: %w", err)
	}
	if _, err := reader.readByte(); err != nil { // server type
		return nil, fmt.Errorf("read server type: %w", err)
	}
	if _, err := reader.readByte(); err != nil { // environment
		return nil, fmt.Errorf("read environment: %w", err)
	}
	if _, err := reader.readByte(); err != nil { // visibility
		return nil, fmt.Errorf("read visibility: %w", err)
	}
	if _, err := reader.readByte(); err != nil { // vac
		return nil, fmt.Errorf("read vac: %w", err)
	}
	version, err := reader.readString()
	if err != nil {
		return nil, fmt.Errorf("read version: %w", err)
	}

	return &Response{
		Name:       name,
		Map:        mapName,
		Players:    int(players),
		MaxPlayers: int(maxPlayers),
		Version:    version,
	}, nil
}

func isTimeout(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

type byteReader struct {
	data []byte
	pos  int
}

func (r *byteReader) readByte() (byte, error) {
	if r.pos >= len(r.data) {
		return 0, ioEOF()
	}
	value := r.data[r.pos]
	r.pos++
	return value, nil
}

func (r *byteReader) readUint16() (uint16, error) {
	lo, err := r.readByte()
	if err != nil {
		return 0, err
	}
	hi, err := r.readByte()
	if err != nil {
		return 0, err
	}
	return uint16(lo) | uint16(hi)<<8, nil
}

func (r *byteReader) readString() (string, error) {
	start := r.pos
	for r.pos < len(r.data) {
		if r.data[r.pos] == 0 {
			value := string(r.data[start:r.pos])
			r.pos++
			return value, nil
		}
		r.pos++
	}
	return "", ioEOF()
}

func ioEOF() error {
	return errors.New("unexpected end of A2S payload")
}
