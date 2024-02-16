package facter

import "github.com/hansmi/paperminer"

type pluginWrapper struct {
	name string
	inst paperminer.DocumentFacter
}

func newPluginWrapper(df paperminer.DocumentFacter) *pluginWrapper {
	return &pluginWrapper{
		name: df.PluginInfo().Name,
		inst: df,
	}
}
