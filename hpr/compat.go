package hpr

type Client struct {
	manager *Manager
}

func NewClient() *Client {
	return &Client{manager: NewManager(WithDefaultDrivers())}
}

func (c *Client) FindPedals() ([]PedalInfo, error) {
	return c.manager.Scan()
}

func (c *Client) Open(info PedalInfo) (Device, error) {
	return c.manager.Open(info)
}

func (c *Client) OpenFirst() (Device, error) {
	return c.manager.OpenFirst()
}

func OpenFirst() (Device, error) {
	return NewManager(WithDefaultDrivers()).OpenFirst()
}

func FindPedals() ([]PedalInfo, error) {
	return NewManager(WithDefaultDrivers()).Scan()
}

func FindSimagicPedals() ([]PedalInfo, error) {
	return FindPedals()
}
