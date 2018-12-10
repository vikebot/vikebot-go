package vikebot

import (
	"bufio"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
)

type roundInformation struct {
	Ticket string `json:"ticket"`
	AesKey string `json:"aes_key"`
	IPV4   string `json:"ipv4"`
	IPV6   string `json:"ipv6"`
	Port   int    `json:"port"`

	Error *string `json:"error"`
}

// Game manages all connections and authorizations for the client. Also holds
// the state of the active player
type Game struct {
	conn *net.Conn
	buf  *bufio.Reader
	gcm  cipher.AEAD
	pc   uint32

	Encrypted bool
	Player    *Player
}

// Close frees all local infos about the game and closes all remote connections
// to any servers or APIs.
func (g *Game) Close() error {
	g.buf = nil
	g.gcm = nil
	g.Encrypted = false
	g.Player = nil
	conn := g.conn
	g.conn = nil
	return (*conn).Close()
}

func (g Game) encrypt(plain []byte) (cipher []byte, err error) {
	// Generate random nonce value for this encryption round
	nonce := make([]byte, g.gcm.NonceSize())
	_, err = io.ReadFull(rand.Reader, nonce)
	if err != nil {
		return nil, fmt.Errorf("vikebot: unable to create nonce value for encryption, err = %v", err)
	}

	// Seal our plaintext
	cipherBuf := g.gcm.Seal(nil, nonce, plain, nil)

	// Append nonce at the front
	cipherBuf = append(nonce, cipherBuf...)

	return cipherBuf, nil
}

func (g Game) encrypt64(plain []byte) (cipher64 []byte, err error) {
	// encrypt plain content
	cipher, err := g.encrypt(plain)
	if err != nil {
		return nil, err
	}

	// convert the cipher to it's base64 equivalent using raw base64 encoding
	base64Cipher := make([]byte, base64.RawStdEncoding.EncodedLen(len(cipher)))
	base64.RawStdEncoding.Encode(base64Cipher, cipher)

	return base64Cipher, nil
}

func (g Game) encryptStr(plain string) (cipher string, err error) {
	cipherBuf, err := g.encrypt([]byte(plain))
	if err != nil {
		return "", err
	}
	return base64.RawStdEncoding.EncodeToString(cipherBuf), err
}

func (g Game) decrypt(cipher []byte) (plain []byte, err error) {
	nonce := cipher[0:g.gcm.NonceSize()]
	ciphertext := cipher[g.gcm.NonceSize():]

	plain, err = g.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return
}

func (g Game) decrypt64(cipher64 []byte) (plain []byte, err error) {
	cipher := make([]byte, base64.RawStdEncoding.DecodedLen(len(cipher64)))
	_, err = base64.RawStdEncoding.Decode(cipher, cipher64)
	if err != nil {
		return nil, err
	}

	return g.decrypt(cipher)
}

func (g Game) decryptStr(cipher string) (plain string, err error) {
	cipherBuf, err := base64.RawStdEncoding.DecodeString(cipher)
	if err != nil {
		return "", err
	}
	plainBuf, err := g.decrypt(cipherBuf)
	if err != nil {
		return "", err
	}
	return string(plainBuf), nil
}

func (g *Game) write(buf []byte) error {
	if g.Encrypted {
		cipher, err := g.encrypt64(buf)
		if err != nil {
			return fmt.Errorf("vikebot: encryption failed - %s", err.Error())
		}
		buf = cipher
	}

	buf = append(buf, '\n')
	_, err := (*g.conn).Write(buf)
	if err != nil {
		return fmt.Errorf("vikebot: %s", err.Error())
	}
	return nil
}

func (g *Game) read(extractPt bool) (pt string, buf []byte, err error) {
	buf, err = g.buf.ReadBytes('\n')
	if err != nil {
		return "", nil, err
	}

	if g.Encrypted {
		buf = buf[:len(buf)-1]
		plain, err := g.decrypt64(buf)
		if err != nil {
			return "", nil, fmt.Errorf("vikebot: unsecure connection - %s", err.Error())
		}
		buf = plain
	}

	if !extractPt {
		return "", buf, nil
	}

	var t typePacket
	err = json.Unmarshal(buf, &t)
	if err != nil {
		return "", nil, fmt.Errorf("vikebot: %s", err.Error())
	}

	return t.Type, buf, nil
}

func (g *Game) trivialAction(pt string, packet []byte) error {
	_, err := g.trivialActionResp(pt, packet)
	return err
}

func (g *Game) trivialActionResp(pt string, packet []byte) (buf []byte, err error) {
	// Send packet
	err = g.write(packet)
	if err != nil {
		return nil, err
	}

	// Read response, unmarshal it and extract error message
	_, buf, err = g.read(false)
	if err != nil {
		return nil, err
	}
	var resp response
	err = json.Unmarshal(buf, &resp)
	if err != nil {
		return nil, fmt.Errorf("vikebot: %s", err.Error())
	}

	// Check for server errors or forbidden messages
	if resp.Type != pt {
		if resp.Error != nil && (resp.Type == "unknown" || resp.Type == "forbidden") {
			return nil, fmt.Errorf("vikebot: %s", *resp.Error)
		}
		return nil, errors.New("vikebot: invalid server response. unexpected packet")
	}

	// Check for pc increase if connection is encrypted
	if g.Encrypted {
		g.pc++
		if resp.Pc == nil {
			return nil, errors.New("vikebot: invalid server response. missing pc")
		} else if *resp.Pc != g.pc {
			return nil, errors.New("vikebot: invalid server response. pc mismatch")
		}
	}

	return buf, nil
}

// Join exchanges the `authtoken` for server credentials and establishes a
// secure connection (`AES256-GCM`) to the game-server. Afterwards it returns
// a game object containing basic infos and the player's state.
func Join(authtoken string) (g *Game, err error) {
	production := true

	// Exchange authtoken for round information
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: !production},
	}
	httpclient := &http.Client{Transport: tr}
	get, err := httpclient.Get("https://api.vikebot.com/v1/roundentry/connectinfo/" + authtoken)
	if err != nil {
		return nil, fmt.Errorf("vikebot: %s", err.Error())
	}
	defer get.Body.Close()

	var ri roundInformation
	err = json.NewDecoder(get.Body).Decode(&ri)
	if err != nil {
		return nil, fmt.Errorf("vikebot: %s", err.Error())
	}

	if ri.Error != nil {
		return nil, fmt.Errorf("vikebot: %v", *ri.Error)
	}

	// Get aes key and iv byte slices
	keyBuf, err := base64.StdEncoding.DecodeString(ri.AesKey)
	if err != nil {
		return nil, fmt.Errorf("vikebot: %s", err.Error())
	}

	// Create aesblock
	block, err := aes.NewCipher(keyBuf)
	if err != nil {
		return nil, fmt.Errorf("vikebot: %s", err.Error())
	}
	// Create gcm cipher
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("vikebot: %s", err.Error())
	}

	// Open connection to game server
	client, err := net.Dial("tcp", fmt.Sprintf("%s:%d", ri.IPV4, ri.Port))
	if err != nil {
		return nil, fmt.Errorf("vikebot: %s", err.Error())
	}

	// Create game object
	g = &Game{
		conn: &client,
		buf:  bufio.NewReader(client),
		gcm:  gcm,
	}

	//
	// Start login process
	//

	// Login packet
	err = g.trivialAction("login", loginPacket(ri.Ticket))
	if err != nil {
		return nil, err
	}

	// Client hello
	challengeBuf := make([]byte, 8)
	_, err = io.ReadFull(rand.Reader, challengeBuf)
	if err != nil {
		return nil, fmt.Errorf("vikebot: %s", err.Error())
	}
	challenge := binary.BigEndian.Uint64(challengeBuf)
	challengeStr := strconv.FormatUint(challenge, 10)
	clienthelloCipher, err := g.encryptStr("clienthello:" + challengeStr)
	if err != nil {
		return nil, fmt.Errorf("vikebot: %s", err.Error())
	}
	err = g.write(clienthelloPacket(clienthelloCipher))
	if err != nil {
		return nil, err
	}

	// Server hello
	pt, buf, err := g.read(true)
	if err != nil {
		return nil, err
	} else if pt != "serverhello" {
		var resp response
		err = json.Unmarshal(buf, &resp)
		if err != nil {
			return nil, fmt.Errorf("vikebot: %s", err.Error())
		}
		if resp.Error != nil {
			return nil, fmt.Errorf("vikebot: %s", *resp.Error)
		}
	}
	var serverhello serverhelloPacket
	err = json.Unmarshal(buf, &serverhello)
	if err != nil {
		return nil, fmt.Errorf("vikebot: %s", err.Error())
	}
	if serverhello.Obj.Cipher == nil {
		return nil, errors.New("vikebot: invalid server response serverhello.Obj.Cipher == nil")
	}
	plainServerhello, err := g.decryptStr(*serverhello.Obj.Cipher)
	if err != nil {
		return nil, err
	} else if plainServerhello != "serverhello:"+challengeStr {
		return nil, fmt.Errorf("vikebot: invalid plain text - expecting 'serverhello:%s'", challengeStr)
	}

	// Connection verified -> enable complete encryption
	g.Encrypted = true

	// Initial pc
	pt, buf, err = g.read(true)
	if err != nil {
		return nil, err
	} else if pt != "initialpc" {
		return nil, errors.New("vikebot: invalid server response. expected initialpc packet")
	}
	var resp response
	err = json.Unmarshal(buf, &resp)
	if err != nil {
		return nil, fmt.Errorf("vikebot: %s", err.Error())
	}
	if resp.Pc == nil {
		return nil, errors.New("vikebot: invalid server response. expected pc in initialpc packet")
	}
	g.pc = *resp.Pc

	// Finished login process itself -> agree on connection
	err = g.trivialAction("agreeconn", agreeconnPacket(g))
	if err != nil {
		return nil, err
	}

	// Allocate player struct
	g.Player = &Player{g: g}

	return g, nil
}

// MustJoin is like Join but panics if any errors occur. It simplifies safe
// initialization of the game object.
func MustJoin(authtoken string) *Game {
	g, err := Join(authtoken)
	if err != nil {
		panic(err)
	}
	return g
}
