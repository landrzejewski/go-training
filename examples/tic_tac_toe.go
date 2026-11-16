package examples

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

const (
	empty     = "-"
	playerX   = "X"
	playerO   = "O"
	boardSize = 3
	maxIndex  = boardSize - 1
)

// game holds the whole state of one tic-tac-toe match. Keeping it in a struct
// rather than in package-level variables means TicTacToe can be called more
// than once in a process and each match starts from an empty board.
type game struct {
	board         [boardSize][boardSize]string
	currentPlayer string
	movesCount    int
}

func newGame() *game {
	g := &game{currentPlayer: playerX}
	for row := range g.board {
		for col := range g.board[row] {
			g.board[row][col] = empty
		}
	}
	return g
}

func (g *game) printBoard() {
	for _, row := range g.board {
		fmt.Println(strings.Join(row[:], " "))
	}
}

func (g *game) makeMove(row, col int) bool {
	if isFieldOutsideBoard(row, col) || g.isFieldTaken(row, col) {
		return false
	}
	g.board[row][col] = g.currentPlayer
	g.movesCount++
	return true
}

func (g *game) changePlayer() {
	if g.currentPlayer == playerX {
		g.currentPlayer = playerO
	} else {
		g.currentPlayer = playerX
	}
}

// The name has to match the return value: this reports true when the field is
// OUTSIDE the board. It used to be called isFieldOnBoard, which claimed exactly
// the opposite - the code only worked because the call site compensated for the
// mistake, and the obvious "fix" to !isFieldOnBoard would have given an index
// out of range.
func isFieldOutsideBoard(row, col int) bool {
	return row < 0 || row > maxIndex || col < 0 || col > maxIndex
}

func (g *game) isFieldTaken(row, col int) bool {
	return g.board[row][col] != empty
}

// winner returns the symbol that ACTUALLY formed a winning line, or empty when
// there is none. It deliberately does not rely on currentPlayer, so it is
// correct no matter when it is called.
func (g *game) winner() string {
	b := &g.board
	for i := 0; i <= maxIndex; i++ {
		if b[i][0] != empty && b[i][0] == b[i][1] && b[i][1] == b[i][2] {
			return b[i][0]
		}
		if b[0][i] != empty && b[0][i] == b[1][i] && b[1][i] == b[2][i] {
			return b[0][i]
		}
	}

	if b[0][0] != empty && b[0][0] == b[1][1] && b[1][1] == b[2][2] {
		return b[0][0]
	}
	if b[0][2] != empty && b[0][2] == b[1][1] && b[1][1] == b[2][0] {
		return b[0][2]
	}

	return empty
}

func (g *game) isBoardFull() bool {
	return g.movesCount >= boardSize*boardSize
}

// TicTacToe plays one game of tic-tac-toe on the terminal. Players take turns
// typing "column row" (zero-based) until somebody completes a row, column or
// diagonal, or the board is full.
func TicTacToe() {
	g := newGame()

	// Read the WHOLE line and only then parse numbers out of it. fmt.Scanln
	// on a parse error does not consume the offending token, so typing "abc" made
	// the next call trip over the same bytes and the program printed
	// "Invalid move. Try again" forever.
	input := bufio.NewScanner(os.Stdin)
	g.printBoard()
	for {
		fmt.Printf("Player %s, enter move (column, row): ", g.currentPlayer)
		if !input.Scan() {
			fmt.Println("\nEnd of input, quitting.")
			return
		}

		var row, col int
		if _, err := fmt.Sscanf(input.Text(), "%d %d", &col, &row); err != nil || !g.makeMove(row, col) {
			fmt.Println("Invalid move. Try again")
			continue
		}

		g.printBoard()

		if winner := g.winner(); winner != empty {
			fmt.Printf("Player %s wins\n", winner)
			return
		}

		if g.isBoardFull() {
			fmt.Println("The game is a draw!")
			return
		}

		g.changePlayer()
	}
}
