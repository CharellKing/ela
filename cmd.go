package main

type Cmd struct {
	Tasks      bool   `long:"tasks" description:"run all tasks"`
	Gateway    bool   `long:"gateway" description:"run gateway server"`
	ConfigFile string `long:"config" description:"load config file"`
	Task       string `long:"task" description:"run a specific task"`
}
