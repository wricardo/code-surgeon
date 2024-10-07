package main

type Person struct {
	Id   string
	Name string
}

func (p Person) GetID() string {
	return p.Id
}
func (p Person) SetID(id string) {
	p.Id = id // nolint
}
func (_ Person) GetLabel() string {
	return "Person"
}
func (p Person) GetProps() map[string]any {
	return map[string]any{
		"Name": p.Name,
	}
}
func (p Person) SetProps(props map[string]any) {
	if val, ok := props["Name"]; ok {
		p.Name = val.(string) // nolint
	}
}
