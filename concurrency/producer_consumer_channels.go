package concurrency

import (
	"fmt"
	"math/rand"
	"sync/atomic"
	"time"
	"github.com/fatih/color"
)

const MAX_ORDER_COUNT int = 1000

var totalOrders, successOrders, failureOrders int
var currentOrderNumber int64 = 0

type Producer struct {
	orders chan Order
}

func (p *Producer) Close() {
	close(p.orders)
}

type Order struct {
	number  int
	message string
	success bool
}

func createOrder(orderNumber int) *Order {
	if orderNumber <= MAX_ORDER_COUNT {
		delay := rand.Intn(2) + 1
		successRate := rand.Intn(30)
		
		fmt.Printf("Creating order with number %d\n", orderNumber)
		time.Sleep(time.Duration(delay) * time.Millisecond)
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
	for {
		atomic.AddInt64(&currentOrderNumber, 1)
		order := createOrder(int(currentOrderNumber))
		if order != nil {
			producer.orders <- *order
		} else {
			break
		}
	}
}

func Orders() {
	producer := &Producer{
		orders: make(chan Order),
	}
	go produce(producer)
	go produce(producer)
	go produce(producer)

	for order := range producer.orders {
		if order.number < MAX_ORDER_COUNT {
			if order.success {
				color.Green("Order with number %d deliverd", order.number)
			} else {
				color.Red("Failed to deliver order with number %d (%v)", order.number, order.message)
			}
		} else {
			time.Sleep(5 * time.Second)
			producer.Close()
		}
	}
	color.Yellow("Stats successCount: %d, failureCount: %d, total: %d", successOrders, failureOrders, totalOrders)
}