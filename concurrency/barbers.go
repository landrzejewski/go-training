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
