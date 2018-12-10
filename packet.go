package vikebot

import "fmt"

type typePacket struct {
	Type string `json:"type"`
}

type response struct {
	Type  string  `json:"type"`
	Pc    *uint32 `json:"pc"`
	Error *string `json:"error"`
}

type serverhelloObj struct {
	Cipher *string `json:"cipher"`
}
type serverhelloPacket struct {
	Type string         `json:"type"`
	Obj  serverhelloObj `json:"obj"`
}

func loginPacket(roundticket string) []byte {
	return []byte(fmt.Sprintf(`{"type":"login","obj":{"roundticket":"%s"}}`, roundticket))
}

func clienthelloPacket(cipher string) []byte {
	return []byte(fmt.Sprintf(`{"type":"clienthello","obj":{"cipher":"%s"}}`, cipher))
}

func agreeconnPacket(g *Game) []byte {
	g.pc++
	return []byte(fmt.Sprintf(`{"type":"agreeconn","pc":%d,"obj":{}}`, g.pc))
}
