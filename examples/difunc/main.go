package main

import (
	"fmt"
	"strings"
)

func main() {
	p := &Person{
		Name:  "Kip",
		SayFn: SayFunc(sayLoud),
	}

	p.IntroduceYourself()
}

type Person struct {
	Name  string
	SayFn SayFunc
}

func (p *Person) IntroduceYourself() {
	p.SayFn("Hi, my name is " + p.Name + ".")
}

type SayFunc func(msg string)

func say(msg string) {
	fmt.Println(msg)
}

func sayLoud(msg string) {
	fmt.Println(strings.ToUpper(msg))
}

func sayMute(msg string) {
	// Do nothing because you're mute
}
