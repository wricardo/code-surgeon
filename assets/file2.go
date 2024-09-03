package assets

import "fmt"

type File2 struct {
	Name string
}

func DeleteFiles2(id int) {

}

type TwilioProvider struct {
	Id   int
	Name string
}

func (p *TwilioProvider) SendSMS(to, body string) error {
	Debug()
	fmt.Printf("Sending SMS to %s using %s provider\n", to, p.Name)
	return nil
}

type NexmoProvider struct {
	Id   int
	Name string
}

func (p *NexmoProvider) SendSMS(to, body string) error {
	fmt.Printf("Sending SMS to %s using %s provider\n", to, p.Name)
	return nil
}

type Poop string
