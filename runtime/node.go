/*
 * Copyright (C) 2015 XLAB, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package runtime

import (
	"fmt"
	"strings"
)

type nodeJsRuntime struct {
	CommonRuntime `yaml:"-,inline"`
	NodeArgs      []string `yaml:"node_args"`
	Main          string   `yaml:"main"`
	Args          []string `yaml:"args"`
}

//
// Interface implementation
//

func (conf nodeJsRuntime) GetRuntimeName() string {
	return string(NodeJS)
}
func (conf nodeJsRuntime) GetRuntimeDescription() string {
	return "Run JavaScript NodeJS 4.4.5 application"
}
func (conf nodeJsRuntime) GetDependencies() []string {
	return []string{"node-4.4.5"}
}
func (conf nodeJsRuntime) Validate() error {
	inherit := conf.Base != ""

	if !inherit {
		if conf.Main == "" {
			return fmt.Errorf("'main' must be provided")
		}
	}

	return conf.CommonRuntime.Validate(inherit)
}
func (conf nodeJsRuntime) GetBootCmd(cmdConfs map[string]*CmdConfig) (string, error) {
	conf.Base = "node-4.4.5:node"
	conf.setDefaultEnv(map[string]string{
		"NODE_ARGS": conf.concatNodeArgs(),
		"MAIN":      conf.Main,
		"ARGS":      strings.Join(conf.Args, " "),
	})
	return conf.CommonRuntime.BuildBootCmd("", cmdConfs)
}
func (conf nodeJsRuntime) GetYamlTemplate() string {
	return `
# REQUIRED
# Filepath of the NodeJS entrypoint (where server is defined).
# Note that package root will correspond to filesystem root (/) in OSv image.
# Example value: /server.js
main: <filepath>

# OPTIONAL
# A list of Node.js args.
# Example value: node_args:
#                   - --require module1
node_args:
   - <list>

# OPTIONAL
# A list of command line args used by the application.
# Example value: args:
#                   - argument1
#                   - argument2
args:
   - <list>
` + conf.CommonRuntime.GetYamlTemplate()
}

//
// Utility
//

func (conf nodeJsRuntime) concatNodeArgs() string {
	res := strings.Join(conf.NodeArgs, " ")

	// This is a workaround since runscript is currently unable to
	// handle empty environment variable as a parameter. So we set
	// dummy value unless user provided some actual value.
	if res == "" {
		return "--no-deprecation"
	}
	return res
}
