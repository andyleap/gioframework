package gioframework

type Cell struct {
	Armies  int
	Type    CellType
	Faction int
}

type CellType int

const (
	Plain CellType = iota
	City
	General
)
