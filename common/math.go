package common

/*func Add(a, b int) int {
	return a + b
}*/

/*func addInt(a, b int) int {
	return a + b
}

func addFloat(a, b float64) float64 {
	return a + b
}*/

/*func Add[T any](a, b T) T {
	return a + b
}*/

/*func Add[T int | float64 | uint](a, b T) T {
	return a + b
}*/

type number interface {
	int | float64 | uint
}

func Add[T number](a, b T) T {
	return a + b
}
