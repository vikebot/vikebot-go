package vikebot

import (
	"encoding/json"
	"errors"
	"fmt"
)

const (
	// AngleLeft is a constant for the 'left'
	AngleLeft = "left"
	// AngleRight is a constant for 'right'
	AngleRight = "right"

	// DirectionNorth is a constant for 'north'
	DirectionNorth = "north"
	// DirectionEast is a constant for 'east'
	DirectionEast = "east"
	// DirectionSouth is a constant for 'south'
	DirectionSouth = "south"
	// DirectionWest is a constant for 'west'
	DirectionWest = "west"
)

// Player represents a single controllable game entitiy (also
// called character).
type Player struct {
	g *Game
}

type errorResp struct {
	Error *string `json:"error,omitempty"`
}

// Rotate implements the function of 'sdk-wiki.vikebot.com/#rotate'
func (p *Player) Rotate(angle string) error {
	p.g.pc++
	buf, err := p.g.trivialActionResp("rotate",
		[]byte(fmt.Sprintf(`{"type":"rotate","pc":%d,"obj":{"angle":"%s"}}`, p.g.pc, angle)))
	if err != nil {
		return fmt.Errorf("vikebot: %s", err.Error())
	}

	var er errorResp

	err = json.Unmarshal(buf, &er)
	if err != nil {
		return fmt.Errorf("vikebot: %s", err.Error())
	}

	if er.Error != nil {
		return fmt.Errorf("vikebot: %s", *er.Error)
	}

	return nil
}

// MustRotate is like `Rotate` but panics if any errors occur. It simplifies
// calling if you aren't interested in error handling.
func (p *Player) MustRotate(angle string) {
	err := p.Rotate(angle)
	if err != nil {
		panic(err)
	}
}

// Move instructs the player to run in the neighbor block specified by the
// direction parameter. Directions are cardinal-directions and predefined
// in package constants: `vikebot.Direction*`
func (p *Player) Move(direction string) error {
	p.g.pc++
	buf, err := p.g.trivialActionResp("move",
		[]byte(fmt.Sprintf(`{"type":"move","pc":%d,"obj":{"direction":"%s"}}`, p.g.pc, direction)))
	if err != nil {
		return fmt.Errorf("vikebot: %s", err.Error())
	}

	var er errorResp

	err = json.Unmarshal(buf, &er)
	if err != nil {
		return fmt.Errorf("vikebot: %s", err.Error())
	}

	if er.Error != nil {
		return fmt.Errorf("vikebot: %s", *er.Error)
	}

	return nil
}

// MustMove is like `Move` but panics if any errors occur. It simplifies
// calling if you aren't interested in error handling.
func (p *Player) MustMove(direction string) {
	err := p.Move(direction)
	if err != nil {
		panic(err)
	}
}

type attackResp struct {
	Obj *struct {
		Health *int `json:"health"`
	}
	Error *string `json:"error,omitempty"`
}

// Attack performs an attack into the direction the player is currently
// looking.
func (p *Player) Attack() (enemyHealth int, err error) {
	p.g.pc++
	buf, err := p.g.trivialActionResp("attack",
		[]byte(fmt.Sprintf(`{"type":"attack","pc":%d,"obj":null}`, p.g.pc)))
	if err != nil {
		return -1, err
	}

	var ar attackResp
	err = json.Unmarshal(buf, &ar)
	if err != nil {
		return 0, fmt.Errorf("vikebot: %s", err.Error())
	}

	if ar.Error != nil {
		return 0, fmt.Errorf("vikebot: %s", *ar.Error)
	}

	if ar.Obj == nil || ar.Obj.Health == nil {
		return 0, errors.New("vikebot: invalid attack-response packet")
	}

	return *(*ar.Obj).Health, nil
}

// MustAttack is like `Attack` but panics if any errors occur. It simplifies
// calling if you aren't interested in error handling.
func (p *Player) MustAttack() (enemyHealth int) {
	enemyHealth, err := p.Attack()
	if err != nil {
		panic(err)
	}
	return enemyHealth
}

type radarResp struct {
	Obj *struct {
		Count *int `json:"counter"`
	} `json:"obj"`
	Error *string `json:"error,omitempty"`
}

// Radar implements the function of 'sdk-wiki.vikebot.com/#radar'
func (p *Player) Radar() (count int, err error) {
	p.g.pc++

	buf, err := p.g.trivialActionResp("radar",
		[]byte(fmt.Sprintf(`{"type":"radar","pc":%d,"obj":null}`, p.g.pc)))
	if err != nil {
		return 0, err
	}

	var rr radarResp
	err = json.Unmarshal(buf, &rr)
	if err != nil {
		return 0, fmt.Errorf("vikebot: %s", err.Error())
	}

	if rr.Error != nil {
		return 0, fmt.Errorf("vikebot: %s", *rr.Error)
	}

	if rr.Obj == nil || rr.Obj.Count == nil {
		return 0, errors.New("vikebot: invalid radar-response packet")
	}

	return *(*rr.Obj).Count, nil
}

// MustRadar is like `Radar` but panics if any errors occur. It simplifies
// calling if you aren't interested in error handling.
func (p *Player) MustRadar() (count int) {
	c, err := p.Radar()
	if err != nil {
		panic(err)
	}
	return c
}

type watchResp struct {
	Obj *struct {
		HealthMatrix *[][]int `json:"health_matrix"`
	}
	Error *string `json:"error,omitempty"`
}

// Watch implements the function of 'sdk-wiki.vikebot.com/#watch'
func (p *Player) Watch() (healthMatrix [][]int, err error) {
	p.g.pc++

	buf, err := p.g.trivialActionResp("watch",
		[]byte(fmt.Sprintf(`{"type":"watch","pc":%d,"obj":null}`, p.g.pc)))
	if err != nil {
		return nil, fmt.Errorf("vikebot: %s", err.Error())
	}
	var wr watchResp
	err = json.Unmarshal(buf, &wr)
	if err != nil {
		return nil, fmt.Errorf("vikebot: %s", err.Error())
	}

	if wr.Error != nil {
		return nil, fmt.Errorf("vikebot: %s", *wr.Error)
	}

	if wr.Obj == nil || wr.Obj.HealthMatrix == nil {
		return nil, errors.New("vikebot: invalid watch-response packet")
	}

	return *(*wr.Obj).HealthMatrix, nil
}

// MustWatch is like `Watch` but panics if any errors occur. It simplifies
// calling if you aren't interested in error handling.
func (p *Player) MustWatch() (healthMatrix [][]int) {
	hm, err := p.Watch()
	if err != nil {
		panic(err)
	}
	return hm
}

type scoutResp struct {
	Obj *struct {
		Count *int `json:"counter"`
	} `json:"obj"`
	Error *string `json:"error,omitempty"`
}

// Scout implements the function of 'sdk-wiki.vikebot.com/#scout'
func (p *Player) Scout(distance int) (count int, err error) {
	p.g.pc++
	buf, err := p.g.trivialActionResp("scout",
		[]byte(fmt.Sprintf(`{"type":"scout","pc":%d,"obj":{"distance":%d}}`, p.g.pc, distance)))
	if err != nil {
		return 0, err
	}
	var sr scoutResp
	err = json.Unmarshal(buf, &sr)
	if err != nil {
		return 0, fmt.Errorf("vikebot: %s", err.Error())
	}

	if sr.Error != nil {
		return 0, fmt.Errorf("vikebot: %s", *sr.Error)
	}

	if sr.Obj == nil || sr.Obj.Count == nil {
		return 0, errors.New("vikebot: invalid scout-response packet")
	}

	return *(*sr.Obj).Count, nil
}

// MustScout is like `Scout` but panics if any errors occur. It simplifies
// calling if you aren't interested in error handling.
func (p *Player) MustScout(distance int) (count int) {
	c, err := p.Scout(distance)
	if err != nil {
		panic(err)
	}
	return c
}

// Defend implements the function of
// 'sdk-wiki.vikebot.com/#defend-and-undefend'
func (p *Player) Defend() (err error) {
	p.g.pc++
	return p.g.trivialAction("defend",
		[]byte(fmt.Sprintf(`{"type":"defend","pc":%d,"obj":null}`, p.g.pc)))
}

// MustDefend is like `Defend` but panics if any errors occur. It simplifies
// calling if you aren't interested in error handling.
func (p *Player) MustDefend() {
	err := p.Defend()
	if err != nil {
		panic(err)
	}
}

// Undefend implements the function of
// 'sdk-wiki.vikebot.com/#defend-and-undefend'
func (p *Player) Undefend() (err error) {
	p.g.pc++
	return p.g.trivialAction("undefend",
		[]byte(fmt.Sprintf(`{"type":"undefend","pc":%d,"obj":null}`, p.g.pc)))
}

// MustUndefend is like `Undefend` but panics if any errors occur. It simplifies
// calling if you aren't interested in error handling.
func (p *Player) MustUndefend() {
	err := p.Undefend()
	if err != nil {
		panic(err)
	}
}

type healthResp struct {
	Obj *struct {
		Value *int `json:"value"`
	} `json:"obj"`
}

// GetHealth implements the function of
// 'sdk-wiki.vikebot.com/#GetHealth'
func (p *Player) GetHealth() (health int, err error) {
	p.g.pc++

	buf, err := p.g.trivialActionResp("health",
		[]byte(fmt.Sprintf(`{"type":"health","pc":%d,"obj":null}`, p.g.pc)))
	if err != nil {
		return 0, err
	}
	var hr healthResp
	err = json.Unmarshal(buf, &hr)
	if err != nil {
		return 0, fmt.Errorf("vikebot: %s", err.Error())
	}

	if hr.Obj == nil || hr.Obj.Value == nil {
		return 0, errors.New("vikebot: invalid health-response packet")
	}

	return *(*hr.Obj).Value, nil
}

// MustGetHealth is like `GetHealth` but panics if any errors occur. It simplifies
// calling if you aren't interested in error handling.
func (p *Player) MustGetHealth() (health int) {
	health, err := p.GetHealth()
	if err != nil {
		panic(err)
	}
	return health
}
