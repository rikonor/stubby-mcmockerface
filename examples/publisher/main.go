package main

import (
	"fmt"
	"log"
	"strings"
	"time"
)

func main() {
	// Create a regular publisher to dest-1
	p := NewPublisher("dest-1")

	// publish some test message
	if err := p.Publish("hello"); err != nil {
		log.Fatal(err)
	}

	// Mock a Publisher
	mp := &MockPublisher{
		PublishFn: func(msg string) error {
			// Lets create a mock publisher that sends messages to nowhere
			// so we're just not doing anything
			return nil
		},
	}

	if err := mp.Publish("this-will-not-go-anywhere"); err != nil {
		log.Fatal(err)
	}

	// Now lets try and create a Publisher that will transform our messages before sending them out
	tp := TransformPublisher(p, func(msg string) string {
		// as an example, lets capitalize the message
		return strings.Title(msg)
	})

	tp.Publish("hello")

	// Now lets create a publisher that will wrap a series of existing publishers
	// and sends any message given to it to all wrapped publishers
	p2 := NewPublisher("dest-2")

	// wrap all of the preceding Publishers into one
	mulp := MultiPublisher(p, mp, tp, p2)

	if err := mulp.Publish("test"); err != nil {
		log.Fatal(err)
	}

	// Another useful thing is to batch messages and send them in bulk
	bp := BatchPublisher(p, 3)
	for i := 0; i < 3; i++ {
		msg := fmt.Sprintf("msg-%d", i)
		if err := bp.Publish(msg); err != nil {
			log.Fatal(err)
		}
	}

	// We can also mock a Publisher such that it always fails (which is handy in tests)
	errp := &MockPublisher{
		PublishFn: func(msg string) error {
			return fmt.Errorf("failed to send msg: %s", msg)
		},
	}

	if err := errp.Publish("test"); err != nil {
		fmt.Printf("Received error as expected: %s\n", err)
	}
}

// Publisher publishes basic string messages
type Publisher interface {
	Publish(msg string) error
}

type publisher struct {
	destination string
}

// NewPublisher creates a new Publisher
func NewPublisher(dest string) Publisher {
	return &publisher{
		destination: dest,
	}
}

func (p *publisher) Publish(msg string) error {
	// Pretend publishing is just printing to stdout
	fmt.Printf("[%d] Publishing message to %s: %s\n", time.Now().UnixNano(), p.destination, msg)
	return nil
}

// MockPublisher is a mockable Publisher
type MockPublisher struct {
	PublishFn func(msg string) error
}

// Publish calls the underlying Publish method
func (p *MockPublisher) Publish(msg string) error {
	return p.PublishFn(msg)
}

// TransformFunc is a function that changes a message and returns the changed version
type TransformFunc func(msg string) string

// TransformPublisher wraps a given Publisher with a message TransformFunc
func TransformPublisher(p Publisher, tfn TransformFunc) Publisher {
	return &MockPublisher{
		PublishFn: func(msg string) error {
			// transform the message using the given transform function, then send it along
			return p.Publish(tfn(msg))
		},
	}
}

// MultiPublisher wraps all given Publishers into one Publisher
func MultiPublisher(ps ...Publisher) Publisher {
	return &MockPublisher{
		PublishFn: func(msg string) error {
			// iterate over all publishers and send to each in turn
			for _, p := range ps {
				// there's multiple possible error handling strategies here
				// in this case we'll just return the first encountered error
				if err := p.Publish(msg); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

// BatchPublisher batches messages together before sending them out
func BatchPublisher(p Publisher, batchSize int) Publisher {
	// hold our batched msgs somewhere
	msgs := []string{}

	return &MockPublisher{
		PublishFn: func(msg string) error {
			msgs = append(msgs, msg)

			// if enough messages have been batched, we can send them out
			if len(msgs) == batchSize {
				// there's multiple ways to batch the messages
				// in this case we'll just concatenate them
				batchMsg := strings.Join(msgs, ",")
				return p.Publish(batchMsg)
			}

			// Note: It's also possible to flush the batch publisher after some pre-defined time duration
			// but to keep the example simple we will not do so

			// still waiting for batch buffer to fill up
			return nil
		},
	}
}
