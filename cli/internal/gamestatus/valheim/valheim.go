package valheim

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
	defaultTimeout = 2 * time.Second
	queryPort      = 2457
	a2sInfoHeader  = 0x54
	a2sInfoReply   = 0x49
	a2sChallenge   = 0x41
)

var baseInfoRequest = []byte{
	0xFF, 0xFF, 0xFF, 0xFF,
	a2sInfoHeader,
	'S', 'o', 'u', 'r', 'c', 'e', ' ', 'E', 'n', 'g', 'i', 'n', 'e', ' ', 'Q', 'u', 'e', 'r', 'y', 0x00,
}

type Checker struct {
	timeout time.Duration
	dial    func(network, address string, timeout time.Duration) (net.Conn, error)
}

type infoResponse struct {
	Name       string
	Map        string
	Players    int
	MaxPlayers int
	Version    string
}

func NewChecker() *Checker {
	return &Checker{
		timeout: defaultTimeout,
		dial:    net.DialTimeout,
	}
}

func (c *Checker) GameName() string {
	return "valheim"
}

func (c *Checker) Check(target gamestatus.Target) (*gamestatus.Status, error) {
	port, err := primaryPort(target)
	if err != nil {
		return &gamestatus.Status{State: gamestatus.StateUnknown, Detail: err.Error()}, nil
	}
	if target.PublicIP == "" {
		return &gamestatus.Status{State: gamestatus.StateUnknown, Detail: "valheim status skipped because no public IP is available"}, nil
	}

	address := net.JoinHostPort(target.PublicIP, strconv.Itoa(port))
	conn, err := c.dial("udp", address, c.timeout)
	if err != nil {
		return &gamestatus.Status{State: gamestatus.StateUnreachable, Detail: fmt.Sprintf("valheim A2S dial failed: %v", err)}, nil
	}
	defer func() {
		_ = conn.Close()
	}()

	_ = conn.SetDeadline(time.Now().Add(c.timeout))

	response, err := queryInfo(conn)
	if err != nil {
		if isTimeout(err) {
			return &gamestatus.Status{State: gamestatus.StateUnreachable, Detail: fmt.Sprintf("valheim A2S query timed out for %s", address)}, nil
		}
		return &gamestatus.Status{State: gamestatus.StateQueryFailed, Detail: fmt.Sprintf("valheim A2S query failed: %v", err)}, nil
	}

	players := response.Players
	return &gamestatus.Status{
		State:       gamestatus.StateReady,
		Detail:      fmt.Sprintf("valheim A2S query succeeded: %s (%d/%d players, map=%s, version=%s)", response.Name, response.Players, response.MaxPlayers, response.Map, response.Version),
		Reachable:   true,
		PlayerCount: &players,
	}, nil
}

func primaryPort(target gamestatus.Target) (int, error) {
	if len(target.Ports) == 0 {
		return 0, fmt.Errorf("valheim status skipped because no ports are configured")
	}

	for _, portRange := range target.Ports {
		if portRange.From <= queryPort && queryPort <= portRange.To {
			return queryPort, nil
		}
	}

	// Valheim uses basePort+1 for queries
	return target.Ports[0].From + 1, nil
}

func queryInfo(conn net.Conn) (*infoResponse, error) {
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
	case a2sInfoReply:
		return parseInfoResponse(payload[5:])
	case a2sChallenge:
		if len(payload) < 9 {
			return nil, fmt.Errorf("short A2S challenge response")
		}
		request := append(append([]byte{}, baseInfoRequest...), payload[5:9]...)
		payload, err = sendInfoRequest(conn, request)
		if err != nil {
			return nil, err
		}
		if len(payload) < 5 || payload[4] != a2sInfoReply {
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

func parseInfoResponse(payload []byte) (*infoResponse, error) {
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

	return &infoResponse{
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

func init() {
	gamestatus.Register(NewChecker())
}
