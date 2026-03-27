package minecraft

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/openhost/cli/internal/gamestatus"
)

const (
	defaultTimeout   = 3 * time.Second
	defaultQueryPort = 25565
)

type Checker struct {
	timeout time.Duration
	dial    func(network, address string, timeout time.Duration) (net.Conn, error)
}

func NewChecker() *Checker {
	return &Checker{
		timeout: defaultTimeout,
		dial:    net.DialTimeout,
	}
}

func (c *Checker) GameName() string {
	return "minecraft"
}

func (c *Checker) Check(target gamestatus.Target) (*gamestatus.Status, error) {
	port := defaultQueryPort
	if len(target.Ports) > 0 {
		port = target.Ports[0].From
	}

	if target.PublicIP == "" {
		return &gamestatus.Status{
			State:  gamestatus.StateUnknown,
			Detail: "minecraft status skipped because no public IP is available",
		}, nil
	}

	address := net.JoinHostPort(target.PublicIP, strconv.Itoa(port))
	conn, err := c.dial("tcp", address, c.timeout)
	if err != nil {
		return &gamestatus.Status{
			State:  gamestatus.StateUnreachable,
			Detail: fmt.Sprintf("minecraft SLP dial failed: %v", err),
		}, nil
	}
	defer func() { _ = conn.Close() }()

	_ = conn.SetDeadline(time.Now().Add(c.timeout))

	// --- Minecraft Server List Ping (SLP) protocol ---
	// 1) Send Handshake packet (packet ID 0x00, next state = 1 / Status)
	handshake := buildHandshakePacket(target.PublicIP, port)
	if _, err := conn.Write(handshake); err != nil {
		return &gamestatus.Status{
			State:  gamestatus.StateUnreachable,
			Detail: fmt.Sprintf("minecraft SLP handshake write failed: %v", err),
		}, nil
	}

	// 2) Send Status Request packet (packet ID 0x00, empty payload)
	statusRequest := encodePacket(0x00, nil)
	if _, err := conn.Write(statusRequest); err != nil {
		return &gamestatus.Status{
			State:  gamestatus.StateUnreachable,
			Detail: fmt.Sprintf("minecraft SLP status request write failed: %v", err),
		}, nil
	}

	// 3) Read Status Response
	response, err := readSLPResponse(conn)
	if err != nil {
		return &gamestatus.Status{
			State:  gamestatus.StateQueryFailed,
			Detail: fmt.Sprintf("minecraft SLP response read failed: %v", err),
		}, nil
	}

	players := response.Players.Online
	detail := fmt.Sprintf("minecraft server %q — %d/%d players, version %s",
		response.Description.Text, players, response.Players.Max, response.Version.Name)

	return &gamestatus.Status{
		State:       gamestatus.StateReady,
		Detail:      detail,
		Reachable:   true,
		PlayerCount: &players,
	}, nil
}

func init() {
	gamestatus.Register(NewChecker())
}

// --- SLP protocol helpers ---

// slpResponse is the JSON payload returned by a Minecraft server in the SLP
// Status Response packet.
type slpResponse struct {
	Version struct {
		Name     string `json:"name"`
		Protocol int    `json:"protocol"`
	} `json:"version"`
	Players struct {
		Max    int `json:"max"`
		Online int `json:"online"`
	} `json:"players"`
	Description struct {
		Text string `json:"text"`
	} `json:"description"`
}

// buildHandshakePacket builds the Minecraft SLP handshake packet.
func buildHandshakePacket(host string, port int) []byte {
	var payload []byte
	payload = appendVarInt(payload, -1)                            // protocol version (-1 = ping)
	payload = appendVarInt(payload, len(host))                     // host string length
	payload = append(payload, []byte(host)...)                     // host string
	payload = binary.BigEndian.AppendUint16(payload, uint16(port)) // port
	payload = appendVarInt(payload, 1)                             // next state: 1 = Status

	return encodePacket(0x00, payload)
}

// encodePacket wraps a payload with a VarInt packet-ID and length prefix.
func encodePacket(packetID int, payload []byte) []byte {
	var inner []byte
	inner = appendVarInt(inner, packetID)
	inner = append(inner, payload...)

	var packet []byte
	packet = appendVarInt(packet, len(inner))
	packet = append(packet, inner...)
	return packet
}

// readSLPResponse reads a single SLP Status Response from the connection.
func readSLPResponse(conn net.Conn) (*slpResponse, error) {
	// Read packet length
	packetLen, err := readVarInt(conn)
	if err != nil {
		return nil, fmt.Errorf("read packet length: %w", err)
	}
	if packetLen < 0 || packetLen > 1<<20 {
		return nil, fmt.Errorf("invalid packet length %d", packetLen)
	}

	// Read packet ID
	packetID, err := readVarInt(conn)
	if err != nil {
		return nil, fmt.Errorf("read packet id: %w", err)
	}
	if packetID != 0x00 {
		return nil, fmt.Errorf("unexpected packet id 0x%02x", packetID)
	}

	// Read JSON string length
	jsonLen, err := readVarInt(conn)
	if err != nil {
		return nil, fmt.Errorf("read json length: %w", err)
	}
	if jsonLen < 0 || jsonLen > 1<<20 {
		return nil, fmt.Errorf("invalid json length %d", jsonLen)
	}

	// Read JSON bytes
	jsonBuf := make([]byte, jsonLen)
	n := 0
	for n < jsonLen {
		read, err := conn.Read(jsonBuf[n:])
		if err != nil {
			return nil, fmt.Errorf("read json body: %w", err)
		}
		n += read
	}

	var resp slpResponse
	if err := json.Unmarshal(jsonBuf, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal SLP json: %w", err)
	}

	return &resp, nil
}

// appendVarInt appends a VarInt-encoded integer to buf.
func appendVarInt(buf []byte, value int) []byte {
	uval := uint32(value)
	for {
		b := byte(uval & 0x7F)
		uval >>= 7
		if uval != 0 {
			b |= 0x80
		}
		buf = append(buf, b)
		if uval == 0 {
			break
		}
	}
	return buf
}

// readVarInt reads a VarInt from a byte-at-a-time reader (net.Conn).
func readVarInt(conn net.Conn) (int, error) {
	var result int
	var shift uint
	buf := make([]byte, 1)
	for i := 0; i < 5; i++ {
		if _, err := conn.Read(buf); err != nil {
			return 0, err
		}
		b := buf[0]
		result |= int(b&0x7F) << shift
		if b&0x80 == 0 {
			return result, nil
		}
		shift += 7
	}
	return 0, fmt.Errorf("varint too long")
}
