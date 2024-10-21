/*
Napisz grę w kółko i krzyżyk
Plansza ma wymiary 3 x 3 pola
Gracze na zmianę zajmują wolne pola, umieszczając na nich swój znak (kółko lub krzyżyk)
Gra kończy się, gdy wszystkie pola zostaną zajęte lub jeden z graczy zajmie zwycięską sekwencję (kolumnę, rząd lub przekątną)
Interfejs gry powinien opierać się na wierszu poleceń/terminalu
*/

package exercises

import (
	"fmt"
)

const (
	empty   = " "
	playerX = "X"
	playerO = "O"
)

var board = [][]string{
	{empty, empty, empty},
	{empty, empty, empty},
	{empty, empty, empty},
}

var currentPlayer = playerX

func printBoard() {
	for _, row := range board {
		fmt.Println(row)
	}
}

func checkWinner() string {
	for i := 0; i < 3; i++ {
		if board[i][0] != empty && board[i][0] == board[i][1] && board[i][1] == board[i][2] {
			return currentPlayer
		}
		if board[0][i] != empty && board[0][i] == board[1][i] && board[1][i] == board[2][i] {
			return currentPlayer
		}
	}

	if board[0][0] != empty && board[0][0] == board[1][1] && board[1][1] == board[2][2] {
		return currentPlayer
	}
	if board[0][2] != empty && board[0][2] == board[1][1] && board[1][1] == board[2][0] {
		return currentPlayer
	}

	return empty
}

func isBoardFull() bool {
	for _, row := range board {
		for _, cell := range row {
			if cell == empty {
				return false
			}
		}
	}
	return true
}

func makeMove(player string, row int, col int) bool {
	if row < 0 || row > 2 || col < 0 || col > 2 || board[row][col] != empty {
		return false
	}
	board[row][col] = player
	return true
}

func TicTacToeExercise() {
	var row, col int
	for {
		printBoard()
		fmt.Printf("Player %s, enter your move (column row): ", currentPlayer)
		_, err := fmt.Scanln(&col, &row)

		if err != nil || !makeMove(currentPlayer, row, col) {
			fmt.Println("Invalid move. Try again.")
			empty := ""
			fmt.Scanln(&empty)
			continue
		}

		winner := checkWinner()
		if winner != empty {
			printBoard()
			fmt.Printf("Player %s wins!\n", winner)
			break
		}

		if isBoardFull() {
			printBoard()
			fmt.Println("The game is a draw!")
			break
		}

		if currentPlayer == playerX {
			currentPlayer = playerO
		} else {
			currentPlayer = playerX
		}
	}
}
