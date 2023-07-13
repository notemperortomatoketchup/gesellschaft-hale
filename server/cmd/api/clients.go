package main

func (app *Application) GetClientsList() []*Client {
	var clients []*Client
	app.clients.Range(func(key, value any) bool {
		clients = append(clients, value.(*Client))
		return true
	})

	return clients
}

func (app *Application) GetAvailableClient(capacity int32) (*Client, bool) {
	clients := app.GetClientsList()
	for _, c := range clients {
		if c.slots >= capacity {
			return c, true
		}
	}
	return nil, false
}
