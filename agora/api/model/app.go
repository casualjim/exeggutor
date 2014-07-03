package model

import "github.com/astaxie/beego/validation"

// App the app controller, which deals with our applications
type App struct {
	// Name represents the name of the application
	Name string `json:"name" valid:"Required;MinSize(3);MaxSize(50);AlphaDash"`
	// Components represent the components this app exists out of
	Components map[string]AppComponent `json:"components"`
}

// Valid validates this struct
func (a App) Valid(v *validation.Validation) {
	if len(a.Components) == 0 {
		v.SetError("components", "requires at least 1 entry")
	}
}
