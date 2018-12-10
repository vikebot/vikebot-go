package vikebot

// MapEntity is a struct with the width and height
// of the map
type MapEntity struct {
	height int
	width  int
}

// Width returns an int of the width of the map entity
func (me *MapEntity) Width() int {
	return me.width
}

// Height returns an int of the Height of the map
// entity
func (me *MapEntity) Height() int {
	return me.height
}

// Block is not implemented yet
func (me *MapEntity) Block(x int, y int) *BlockEntity {
	return nil
}
