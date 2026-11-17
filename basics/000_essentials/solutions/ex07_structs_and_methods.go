package solutions

import "fmt"

// ---- Exercise 7.1: value and pointer receivers -------------------------------------------------

// Rectangle is a width and a height.
type Rectangle struct {
	Width, Height float64
}

// Area, Perimeter and String only READ the rectangle, so a value receiver (a copy) is enough.
// Scale must CHANGE it, so it needs a pointer receiver. Section 007's rule "one pointer
// receiver means all pointer receivers" is the safe default; a small value type that is mostly
// read, like this one (or time.Time in the standard library), is the common exception.

func (r Rectangle) Area() float64      { return r.Width * r.Height }
func (r Rectangle) Perimeter() float64 { return 2 * (r.Width + r.Height) }

// Scale multiplies both sides by factor.
func (r *Rectangle) Scale(factor float64) {
	r.Width *= factor
	r.Height *= factor
}

// String implements fmt.Stringer, so fmt prints a Rectangle as "3x4".
func (r Rectangle) String() string {
	return fmt.Sprintf("%gx%g", r.Width, r.Height)
}

// ---- Exercise 7.2: a constructor and methods ---------------------------------------------------

// CartItem is one line of a shopping cart.
type CartItem struct {
	Price    float64
	Quantity int
}

// ShoppingCart maps product names to cart lines. Create one with NewCart.
type ShoppingCart struct {
	items map[string]CartItem // unexported: change it only through the methods
}

// NewCart returns an empty cart that is ready to use. The constructor is needed because the
// zero value would hold a nil map, and writing to a nil map panics.
func NewCart() *ShoppingCart {
	return &ShoppingCart{items: make(map[string]CartItem)}
}

// Add puts quantity units of a product into the cart. Adding a product that is already there
// increases its quantity and uses the new price.
func (c *ShoppingCart) Add(name string, price float64, quantity int) {
	item := c.items[name] // the zero CartItem when missing
	item.Price = price
	item.Quantity += quantity
	c.items[name] = item // a map entry cannot be changed in place: read, modify, write back
}

// Remove deletes a product and reports whether it was in the cart.
func (c *ShoppingCart) Remove(name string) bool {
	if _, ok := c.items[name]; !ok {
		return false
	}
	delete(c.items, name)
	return true
}

// Total returns the sum of price * quantity over all products.
func (c *ShoppingCart) Total() float64 {
	total := 0.0
	for _, item := range c.items {
		total += item.Price * float64(item.Quantity)
	}
	return total
}

// ---- Exercise 7.3: embedding -------------------------------------------------------------------

// Vehicle is the embedded type.
type Vehicle struct {
	Brand  string
	Wheels int
}

// Describe returns e.g. "Fiat with 4 wheels".
func (v Vehicle) Describe() string {
	return fmt.Sprintf("%s with %d wheels", v.Brand, v.Wheels)
}

// Car embeds Vehicle, so Brand, Wheels and Describe are promoted to Car.
type Car struct {
	Vehicle
	Doors int
}

// Describe on Car hides the promoted Vehicle.Describe, but can still call it explicitly.
func (c Car) Describe() string {
	return c.Vehicle.Describe() + fmt.Sprintf(" and %d doors", c.Doors)
}
