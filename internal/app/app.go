package app

// App is the Wails application service, exposed to the frontend via bindings.
type App struct{}

func NewApp() *App {
	return &App{}
}
