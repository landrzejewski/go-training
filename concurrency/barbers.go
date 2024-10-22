// https://en.wikipedia.org/wiki/Sleeping_barber_problem

// The Sleeping Barber dilemma, a classic computer science problem
// which illustrates the complexities that arise when there are multiple operating system processes. Here, we have
// a finite number of barbers, a finite number of seats in a waiting room, a fixed length of time the barbershop is
// open, and clients arriving at (roughly) regular intervals. When a barber has nothing to do, he or she checks the
// waiting room for new clients, and if one or more is there, a haircut takes place. Otherwise, the barber goes to
// sleep until a new client arrives. So the rules are as follows:
//
//		- if there are no customers, the barber falls asleep in the chair
//		- a customer must wake the barber if he is asleep a customer arrives while the barber is working, t
//		- the customer leaves if all chairs are occupied and
//		  sits in an empty chair if it's available
//		- when the barber finishes a haircut, he inspects the waiting room to see if there are any waiting customers
//		  and falls asleep if there are none
// 		- shop can stop accepting new clients at closing time, but the barbers cannot leave until the waiting room is
//	      empty
//		- after the shop is closed and there are no clients left in the waiting area, the barber
//		  goes home
// The Sleeping Barber was originally proposed in 1965 by computer science pioneer Edsger Dijkstra.

package concurrency

import (
	"fmt"
	"github.com/fatih/color"
	"math/rand"
	"sync"
	"time"
)

var lock = sync.Mutex{}

type BarberShop struct {
	HairCutDuration   time.Duration
	NumberOfBarbers   int
	BarberDoneChannel chan bool
	ClientsChannel    chan string
	Open              bool
	Counter           int
}

func (shop *BarberShop) addBarber(barber string) {
	lock.Lock()
	shop.NumberOfBarbers++
	lock.Unlock()
	go func() {
		isSleeping := false
		color.Yellow("%s goes to the waiting room to check for clients", barber)
		for {
			lock.Lock()
			if len(shop.ClientsChannel) == 0 && shop.Open {
				color.Yellow("%s goes to sleep", barber)
				isSleeping = true
			}
			lock.Unlock()
			client, isOpen := <-shop.ClientsChannel
			if isOpen {
				if isSleeping {
					color.Yellow("%s client wakes %s barber up", client, barber)
					isSleeping = false
				}
				shop.cutHair(barber, client)
			} else {
				for range shop.ClientsChannel {
					shop.cutHair(barber, client)
				}
				shop.sendBarberHome(barber)
				return
			}
		}
	}()
}

func (shop *BarberShop) cutHair(barber, client string) {
	color.Yellow("%s is cutting %s's hair", barber, client)
	time.Sleep(shop.HairCutDuration)
	color.Yellow("%s is finished cutting %s's hair", barber, client)
	mutex.Lock()
	shop.Counter++
	mutex.Unlock()
}

func (shop *BarberShop) sendBarberHome(barber string) {
	color.Yellow("%s is going home", barber)
	shop.BarberDoneChannel <- true
}

func (shop *BarberShop) closeShop() {
	color.Cyan("Closing shop")
	close(shop.ClientsChannel)
	lock.Lock()
	shop.Open = false
	lock.Unlock()
	for a := 1; a <= shop.NumberOfBarbers; a++ {
		<-shop.BarberDoneChannel
	}
	color.Cyan("The barbershop is now closed")
}

func (shop *BarberShop) addClient(client string) {
	color.Yellow("%s arrives", client)
	lock.Lock()
	if shop.Open {
		lock.Unlock()
		select {
		case shop.ClientsChannel <- client:
			color.Yellow("%s takes a seat in the wating room", client)
		default:
			color.Red("The waiting room is full, so %s leaves", client)
		}
	} else {
		lock.Unlock()
		color.Red("The shop is already closed, so %s leaves", client)
	}
}

const timeOpen = 10 * time.Second

func Barbers() {
	shopClosing := make(chan bool)
	closed := make(chan bool)

	shop := BarberShop{
		HairCutDuration:   1000 * time.Millisecond,
		NumberOfBarbers:   0,
		BarberDoneChannel: make(chan bool),
		ClientsChannel:    make(chan string, 10),
		Open:              true,
	}
	color.Cyan("Barber shop is open")

	shop.addBarber("Jan")
	shop.addBarber("Marek")

	go func() {
		<-time.After(timeOpen)
		shopClosing <- true
		shop.closeShop()
		closed <- true
	}()

	index := 1
	go func() {
		for {
			randomTime := rand.Int() % (200)
			select {
			case <-shopClosing:
				return
			case <-time.After(time.Millisecond * time.Duration(randomTime)):
				shop.addClient(fmt.Sprintf("Client #%d", index))
				index++
			}
		}
	}()

	<-closed
	fmt.Println(shop.Counter)
}
