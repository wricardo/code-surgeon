package main

import "fmt"

type TwilioProvider struct {
	Id   int
	Name string
}

func (p *TwilioProvider) SendSMS(to, body string) error {
	fmt.Printf("Sending SMS to %s using %s provider\n", to, p.Name)
	return nil

}

type NexmoProvider struct {
	Id int

	Name string
}

func (p *NexmoProvider) SendSMS(to, body string) error {
	fmt.Printf("Sending SMS to %s using %s provider\n",

		to,
		p.Name)

	return nil
}

type Person struct{ Name string }
