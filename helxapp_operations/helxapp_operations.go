package helxapp_operations

import (
	helxv1 "github.com/helxplatform/helxapp/api/v1"
)

var apps *map[string]helxv1.HelxAppSpec

// AddToMap adds the given HelxAppSpec instance to the map
func addAppToMap(m *map[string]helxv1.HelxAppSpec, app *helxv1.HelxAppSpec) {
	(*m)[app.Name] = *app
}

func AddApp(app *helxv1.HelxAppSpec) {
	addAppToMap(apps, app)
}
