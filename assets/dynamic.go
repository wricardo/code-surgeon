package assets

import "fmt"

func (p *TwilioProvider) SendSMS(to, body string) error {
	fmt.Printf("Sending SMS to %s using %s provider\n", to, p.Name)
	return nil
}

func (p *NexmoProvider) SendSMS(to, body string) error {
	fmt.Printf("Sending SMS to %s using %s provider\n", to, p.Name)
	return nil
}

type Person struct{ Name string }
