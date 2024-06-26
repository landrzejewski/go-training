package concurrency

import (
	"fmt"
	"math/rand"
	"time"
)

const MAX_ORDER_COUNT int = 20

var totalOrders, successOrders, failureOrders int

type Producer struct {
	success chan Order
	failure chan chan error
}

func (p *Producer) Close() error {
	channel := make(chan error)
	p.failure <- channel
	return <- channel
}

type Order struct {
	number  int
	message string
	success bool
}

func createOrder(orderNumber int) *Order {
	if orderNumber <= MAX_ORDER_COUNT {
		delay := rand.Intn(5) + 1
		successRate := rand.Intn(15)
		
		fmt.Printf("Creating order with number %d\n", orderNumber)
		time.Sleep(time.Duration(delay) * time.Second)
		message := ""

		success := false
		if successRate <= 2{
			message = "We ran out of products"
			failureOrders++
		} else if successRate <= 5 {
			message = "Machine is broken"
			failureOrders++
		} else {
			message = fmt.Sprintf("Order with number %d created", orderNumber)
			successOrders++
			success = true
		}
		totalOrders++
		
		return &Order{
			orderNumber,
			message,
			success,
		}
	}
	return nil
}

func produce(producer *Producer) {
	orderNumber := 0
	for {
		orderNumber++
		order := createOrder(orderNumber)
		if order != nil {
			select {
			case producer.success <- *order:
			case errorChannel := <- producer.failure:
				close(producer.success)
				close(errorChannel)
			}
		}
	}
}

func Orders() {
	producer := &Producer{
		success: make(chan Order),
		failure: make(chan chan error),
	}
	go produce(producer)

	for order := range producer.success {
		if order.number <= MAX_ORDER_COUNT {
			if order.success {
				fmt.Printf("Order with number %d deliverd\n", order.number)
			} else {
				fmt.Printf("Failed to deliver order with number %d (%v)\n", order.number, order.message)
			}
		} else {
			err := producer.Close()
			if (err != nil) {
				fmt.Println("Closing channel failed")
			}
		}
	}
}