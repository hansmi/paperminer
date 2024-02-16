package paperminer

import "github.com/hansmi/staticplug"

var globalPluginRegistry = staticplug.NewRegistry()

// GlobalPluginRegistry returns a pointer to a global plugin registry.
func GlobalPluginRegistry() *staticplug.Registry {
	return globalPluginRegistry
}

func MustRegisterPlugin(p staticplug.Plugin) {
	GlobalPluginRegistry().MustRegister(p)
}
