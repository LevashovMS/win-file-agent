package server

type ArgsHandler func(*server)

func (c *server) verification() error {
	if c.port < 80 {
		c.port = 8099
	}

	return nil
}

func Port(port int) ArgsHandler {
	return func(o *server) {
		o.port = port
	}
}

func Handler(method, path string, h routerAction) ArgsHandler {
	return func(o *server) {
		o.router.regHandler(method, path, h)
	}
}
