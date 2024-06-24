package main

import (
	"fmt"
)

const (
	EMPTY = " "
	PLAYER_X = "X"
	PLAYER_O = "O"
)

var board = [3][3]string{
	{EMPTY, EMPTY, EMPTY},
	{EMPTY, EMPTY, EMPTY},
	{EMPTY, EMPTY, EMPTY},
}

func printBoard() {
	for _, row := range board {
		fmt.Println(row)
	}
}

func checkWinner() string {
	for i := 0; i < 3; i++ {
		if board[i][0] != EMPTY && board[i][0] == board[i][1] && board[i][1] == board[i][2] {
			return board[i][0]
		}
		if board[0][i] != EMPTY && board[0][i] == board[1][i] && board[1][i] == board[2][i] {
			return board[0][i]
		}
	}

	if board[0][0] != EMPTY && board[0][0] == board[1][1] && board[1][1] == board[2][2] {
		return board[0][0]
	}
	if board[0][2] != EMPTY && board[0][2] == board[1][1] && board[1][1] == board[2][0] {
		return board[0][2]
	}

	return EMPTY
}

func isBoardFull() bool {
	for _, row := range board {
		for _, cell := range row {
			if cell == EMPTY {
				return false
			}
		}
	}
	return true
}

func makeMove(player string, row int, col int) bool {
	if row < 0 || row > 2 || col < 0 || col > 2 || board[row][col] != EMPTY {
		return false
	}
	board[row][col] = player
	return true
}

func main() {
	currentPlayer := PLAYER_X
	var row, col int
	for {
		printBoard()
		fmt.Printf("Player %s, enter your move (column row): ", currentPlayer)
		fmt.Scan(&col, &row)
		if !makeMove(currentPlayer, row, col) {
			fmt.Println("Invalid move. Try again.")
			continue
		}

		winner := checkWinner()
		if winner != EMPTY {
			printBoard()
			fmt.Printf("Player %s wins!\n", winner)
			break
		}

		if isBoardFull() {
			printBoard()
			fmt.Println("The game is a draw!")
			break
		}

		if currentPlayer == PLAYER_X {
			currentPlayer = PLAYER_O
		} else {
			currentPlayer = PLAYER_X
		}
	}
}
