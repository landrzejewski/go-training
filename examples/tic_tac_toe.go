package examples

import "fmt"

const (
	empty   = "-"
	playerX = "X"
	playerO = "O"
	maxIndex = 2
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

func makeMove(row, col int) bool {
	if isFieldOnBoard(row, col) || isFieldTaken(row, col) {
		return false
	}
	board[row][col] = currentPlayer
	return true
}

func changePlayer() {
	if currentPlayer == playerX {
		currentPlayer = playerO
	} else {
		currentPlayer = playerX
	}
}

func isFieldOnBoard(row, col int) bool {
	return row < 0 || row > maxIndex || col < 0 || col > maxIndex
}

func isFieldTaken(row, col int) bool {
	return board[row][col] != empty
}

func checkWinner() string {
	for i := 0; i <= maxIndex; i++ {
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
		for _, field := range row {
			if field == empty {
				return false
			}
		}
	}
	return true
}

func TicTacToeExercise() {
	var row, col int
	for {
		fmt.Printf("Player %s, enter move (column, row): ", currentPlayer)
		_, err := fmt.Scanln(&col, &row)

		if err != nil || !makeMove(row, col) {
			fmt.Printf("Invalid move. Try again")
			continue
		}

		printBoard()

		winner := checkWinner()

		if winner != empty {
			fmt.Printf("Player %s wins\n", winner)
			break
		}

		if isBoardFull() {
			fmt.Println("The game is a draw!")
			break
		}

		changePlayer()
	}
}