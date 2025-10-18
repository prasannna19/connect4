package game

import (
	"math/rand"
)

// Cell values for the grid
type Cell int

const (
	Empty  Cell = 0
	Red    Cell = 1 // Player
	Yellow Cell = 2 // Bot
	Rows        = 6
	Cols        = 7
)

// NewGrid creates a fresh 6x7 empty board
func NewGrid() [][]Cell {
	grid := make([][]Cell, Rows)
	for i := range grid {
		grid[i] = make([]Cell, Cols)
	}
	return grid
}

// DropDisc drops a disc into a given column for a player (Red or Yellow)
func DropDisc(g [][]Cell, col int, who Cell) bool {
	if col < 0 || col >= Cols {
		return false
	}
	for r := Rows - 1; r >= 0; r-- {
		if g[r][col] == Empty {
			g[r][col] = who
			return true
		}
	}
	return false
}

// CheckWin verifies whether any player has connected 4
func CheckWin(g [][]Cell) (Cell, bool) {
	dirs := [][2]int{{1, 0}, {0, 1}, {1, 1}, {1, -1}}
	for r := 0; r < Rows; r++ {
		for c := 0; c < Cols; c++ {
			if g[r][c] == Empty {
				continue
			}
			for _, d := range dirs {
				count := 0
				for i := 0; i < 4; i++ {
					nr, nc := r+d[0]*i, c+d[1]*i
					if nr < 0 || nr >= Rows || nc < 0 || nc >= Cols {
						break
					}
					if g[nr][nc] == g[r][c] {
						count++
					}
				}
				if count == 4 {
					return g[r][c], true
				}
			}
		}
	}
	return Empty, false
}

// copyGrid helps simulate bot moves safely
func copyGrid(g [][]Cell) [][]Cell {
	newG := make([][]Cell, Rows)
	for i := range g {
		newG[i] = append([]Cell(nil), g[i]...)
	}
	return newG
}

// BotMove — smart bot: tries to win, block, or pick random valid column
func BotMove(g [][]Cell) int {
	// Try to win
	for c := 0; c < Cols; c++ {
		temp := copyGrid(g)
		if DropDisc(temp, c, Yellow) {
			if w, ok := CheckWin(temp); ok && w == Yellow {
				return c
			}
		}
	}

	// Try to block player win
	for c := 0; c < Cols; c++ {
		temp := copyGrid(g)
		if DropDisc(temp, c, Red) {
			if w, ok := CheckWin(temp); ok && w == Red {
				return c
			}
		}
	}

	// Otherwise pick a random valid move
	for c := 0; c < Cols; c++ {
		if g[0][c] == Empty {
			return c
		}
	}
	return rand.Intn(Cols)
}
