// https://en.wikipedia.org/wiki/Sleeping_barber_problem

// This is a simple demonstration of how to solve the Sleeping Barber dilemma, a classic computer science problem
// which illustrates the complexities that arise when there are multiple operating system processes. Here, we have
// a finite number of barbers, a finite number of seats in a waiting room, a fixed length of time the barbershop is
// open, and clients arriving at (roughly) regular intervals. When a barber has nothing to do, he or she checks the
// waiting room for new clients, and if one or more is there, a haircut takes place. Otherwise, the barber goes to
// sleep until a new client arrives. So the rules are as follows:
//
//		- if there are no customers, the barber falls asleep in the chair
//		- a customer must wake the barber if he is asleepf a customer arrives while the barber is working, t
//		- ihe customer leaves if all chairs are occupied and
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
	"math/rand"
	"time"

	"github.com/fatih/color"
)

type BarberShop struct {
	Capacity int
	HairCutDuration time.Duration
	NumberOfBarbers int
	BarberDoneChannel chan bool
	CleintsChannel chan string
	Open bool
}

func (shop *BarberShop) addBarber(barber string) {
	shop.NumberOfBarbers++
	go func ()  {
		isSleeping := false
		color.Yellow("%s goes to the waiting room to check for clients", barber)
		for {
			if (len(shop.CleintsChannel) == 0) {
				color.Yellow("%s goes to sleep", barber)
				isSleeping = true
			}
			client, isOpen := <- shop.CleintsChannel
			if isOpen {
				if isSleeping {
					color.Yellow("%s clien t wakes %s barber up", client, barber)
					isSleeping = false
				}
				shop.cutHait(barber, client)
			} else {
				shop.sendBarberHome(barber)
				return
			}
		}
	}()
}

func (shop *BarberShop) cutHait(barber, client string) {
	color.Green("%s is cutting %s's hair", barber, client)
	time.Sleep(shop.HairCutDuration)
	color.Green("%s is finished cutting %s's hair", barber, client)
}

func (shop *BarberShop) sendBarberHome(barber string) {
	color.Cyan("%s is going home", barber)
	shop.BarberDoneChannel <- true 
}

func (shop *BarberShop) closeShop() {
	color.Cyan("Closing shop")
	close(shop.CleintsChannel)
	shop.Open = false
	for a := 1; a <= shop.NumberOfBarbers; a++ {
		<- shop.BarberDoneChannel
	}
	color.Green("The barbershop is now closed")
}

func (shop *BarberShop) addClient(client string) {
	color.Green("%s arrives", client)
	if shop.Open {
		select {
		case shop.CleintsChannel <- client:
			color.Yellow("%s takes a seat in the wating room", client)
		default:
			color.Red("The waiting room is full, so %s leaves", client)
		}
	} else {
		color.Red("The shop is already closed, so %s leaves", client)
	}
}

const timeOpen = 10 * time.Second

func Run() {
	shopClosing := make(chan bool)
	closed := make(chan bool)

	shop := BarberShop{
		Capacity: 10,
		HairCutDuration: 1000 * time.Millisecond,
		NumberOfBarbers: 0,
		BarberDoneChannel: make(chan bool),
		CleintsChannel: make(chan string),
		Open: true,
	}
	color.Green("Barber shop is open")

	shop.addBarber("Jan")
	shop.addBarber("Marek")

	go func ()  {
		<- time.After(timeOpen)
		shopClosing <- true
		shop.closeShop()
		closed <- true
	}()

	index := 1
	go func ()  {
		for {
			randomTime := rand.Int() % (2 * 800)
			select {
			case <- shopClosing:
				return	
			case <-time.After(time.Millisecond * time.Duration(randomTime)):
				shop.addClient(fmt.Sprintf("Client #%d", index))
				index++
			}
		}
	}()

	<- closed
}
