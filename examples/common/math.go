package common

// Number is the constraint accepted by Add. It is the end of a progression
// that started with a plain `func Add(a, b int) int`, then one copy per type
// (addInt, addFloat), then `[T any]` - which does not compile, because `any`
// does not support the + operator - and finally a union of the types that do.
//
// The ~ prefix means "any type whose UNDERLYING type is this". Without it the
// constraint accepts only int, float64 and uint exactly, and rejects defined
// types such as `type Celsius float64`.
type Number interface {
	~int | ~float64 | ~uint
}

// Add returns a + b for any Number type.
func Add[T Number](a, b T) T {
	return a + b
}
