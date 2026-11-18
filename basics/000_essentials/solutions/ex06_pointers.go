package solutions

// ---- Exercise 6.1: & and * ---------------------------------------------------------------------

// SwapInts swaps the values that a and b point to.
func SwapInts(a, b *int) {
	*a, *b = *b, *a
}

// ---- Exercise 6.2: copies and pointers into a slice --------------------------------------------

// Player is a name with a score.
type Player struct {
	Name  string
	Score int
}

// AddBonus adds bonus to the score of every player in the slice.
func AddBonus(players []Player, bonus int) {
	// "for _, p := range players { p.Score += bonus }" compiles but does nothing:
	// p is a COPY of the element. Index into the slice to change the element itself.
	for i := range players {
		players[i].Score += bonus
	}
}

// FindPlayer returns a pointer to the player with the given name, or nil if there is none.
// The pointer points INTO the slice, so a change made through it shows up in the slice.
func FindPlayer(players []Player, name string) *Player {
	for i := range players {
		if players[i].Name == name {
			return &players[i]
		}
	}
	return nil
}

// ---- Exercise 6.3: a linked list ---------------------------------------------------------------

// Node is one element of a singly linked list. A nil *Node is the empty list.
type Node struct {
	Value int
	Next  *Node
}

// Prepend returns a new list with v in front of head. head may be nil.
func Prepend(head *Node, v int) *Node {
	return &Node{Value: v, Next: head} // safe: the Node lives on after Prepend returns
}

// Values returns the values of the list from head to tail.
func Values(head *Node) []int {
	var out []int
	for n := head; n != nil; n = n.Next { // follow the pointers until the end (nil)
		out = append(out, n.Value)
	}
	return out
}
