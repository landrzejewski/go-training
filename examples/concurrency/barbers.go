package concurrency

import (
	"fmt"
	"math/rand/v2"
	"sync"
	"time"
)

// client is what travels through the waiting-room channel. There is deliberately
// no barber type: a barber is a goroutine, and its only state is the id already
// passed to it as a parameter.
type client struct {
	id int
}

// BarberShop is one instance of the sleeping-barber problem: a number of
// barbers, a waiting room with a fixed number of seats, and clients arriving at
// roughly regular intervals while the shop is open.
type BarberShop struct {
	numBarbers      int
	waitingRoomSize int
	// clientsChan IS the waiting room: its buffer holds exactly waitingRoomSize
	// seats. An earlier version also kept a separate currentSize counter in its own
	// goroutine - two sources of truth for one piece of state, and they drifted
	// apart (a client incremented the counter before sending to the channel, while
	// a barber decremented it only after receiving), so clients were turned away
	// with seats still free. Here the `select`/`default` send is both the check and
	// the claim, so there is no window for a race (TOCTOU).
	clientsChan       chan *client
	shopOpenDuration  time.Duration
	haircutDuration   time.Duration
	clientArrivalRate time.Duration
	wg                sync.WaitGroup
}

// NewBarberShop configures a shop; nothing happens until Start is called.
func NewBarberShop(numBarbers, waitingRoomSize int, shopOpenDuration, haircutDuration, clientArrivalRate time.Duration) *BarberShop {
	return &BarberShop{
		numBarbers:        numBarbers,
		waitingRoomSize:   waitingRoomSize,
		clientsChan:       make(chan *client, waitingRoomSize),
		shopOpenDuration:  shopOpenDuration,
		haircutDuration:   haircutDuration,
		clientArrivalRate: clientArrivalRate,
	}
}

func (bs *BarberShop) barber(id int) {
	defer bs.wg.Done()
	fmt.Printf("Barber %d: Ready to work\n", id)

	// range over a channel: the barber sleeps (blocks) while there are no clients,
	// wakes up for a new one, and returns only once the channel is closed AND empty
	// - that is, after the shop has closed and the waiting room has been served.
	for client := range bs.clientsChan {
		fmt.Printf("Barber %d: Cutting hair for client %d (waiting room: %d/%d)\n",
			id, client.id, len(bs.clientsChan), bs.waitingRoomSize)
		time.Sleep(bs.haircutDuration)
		fmt.Printf("Barber %d: Finished cutting hair for client %d\n", id, client.id)
	}

	fmt.Printf("Barber %d: Going home\n", id)
}

func (bs *BarberShop) addClient(id int) {
	newClient := &client{id: id}

	// Check for a free seat and claim it in one indivisible operation.
	select {
	case bs.clientsChan <- newClient:
		fmt.Printf("Client %d: Entered waiting room (seats occupied: %d/%d)\n",
			id, len(bs.clientsChan), bs.waitingRoomSize)
	default:
		fmt.Printf("Client %d: Waiting room full, leaving\n", id)
	}
}

// nextArrival returns the base arrival interval plus up to 50% random jitter.
func (bs *BarberShop) nextArrival() time.Duration {
	// rand.N panics for n <= 0, so only draw when the range is positive.
	if half := bs.clientArrivalRate / 2; half > 0 {
		return bs.clientArrivalRate + rand.N(half)
	}
	return bs.clientArrivalRate
}

// Start opens the shop, runs it for shopOpenDuration, and returns once the
// shop has closed and every barber has gone home. It blocks the caller.
func (bs *BarberShop) Start() {
	fmt.Println("Barber shop is opening!")
	fmt.Printf("Shop configuration: %d barbers, %d waiting room seats\n", bs.numBarbers, bs.waitingRoomSize)

	for i := 1; i <= bs.numBarbers; i++ {
		bs.wg.Add(1)
		go bs.barber(i)
	}

	closingTimer := time.After(bs.shopOpenDuration)
	clientID := 1

	// Arrivals are jittered by resetting a timer to a fresh random interval each
	// round. Sleeping inside the `case` instead would stall the whole select loop,
	// so the closing timer would be serviced late and the shop would stay open past
	// shopOpenDuration.
	arrival := time.NewTimer(bs.nextArrival())
	defer arrival.Stop()

	shopOpen := true

	for shopOpen {
		select {
		case <-arrival.C:
			fmt.Printf("Client %d: Arriving at shop\n", clientID)
			bs.addClient(clientID)
			clientID++
			arrival.Reset(bs.nextArrival())

		case <-closingTimer:
			fmt.Println("\nBarber shop is closing! No new clients accepted.")
			shopOpen = false
		}
	}

	// Closing the channel means "no more admissions". The barbers finish everyone
	// already seated in the waiting room and only then go home - matching the rule
	// "barbers cannot leave until the waiting room is empty".
	close(bs.clientsChan)
	bs.wg.Wait()

	fmt.Println("\nAll barbers have gone home. Shop is closed!")
}

// Barbers runs a short demo of the sleeping-barber problem: two barbers, five
// seats, the shop open for three seconds. It finishes in a few seconds.
func Barbers() {
	numBarbers := 2
	waitingRoomSize := 5
	shopOpenDuration := 3 * time.Second
	haircutDuration := 500 * time.Millisecond
	clientArrivalRate := 100 * time.Millisecond

	shop := NewBarberShop(numBarbers, waitingRoomSize, shopOpenDuration, haircutDuration, clientArrivalRate)
	shop.Start()
}
