// Copyright The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"os"
	"os/exec"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDefaultEnvars(t *testing.T) {
	if mode := os.Getenv("PROMETHEUS_MCP_TEST_PROCESS"); mode != "" {
		os.Args = os.Args[:1]
		if mode == "help" {
			os.Args = append(os.Args, "--help")
		} else {
			// Stop before starting the server if the timeout variable is ignored.
			os.Args = append(os.Args, "--http.config=.")
		}
		main()
		return
	}

	executable, err := os.Executable()
	require.NoError(t, err)
	var env []string
	for _, variable := range os.Environ() {
		if !strings.HasPrefix(strings.ToUpper(variable), "PROMETHEUS_MCP_") {
			env = append(env, variable)
		}
	}

	for _, name := range []string{"prometheus-mcp", "prometheus-mcp.exe"} {
		t.Run(name, func(t *testing.T) {
			for _, mode := range []string{"help", "timeout"} {
				t.Run(mode, func(t *testing.T) {
					ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
					defer cancel()
					// Kingpin reads os.Args[0] during package initialization.
					cmd := exec.CommandContext(ctx, executable, "-test.run=^TestDefaultEnvars$")
					cmd.Args[0] = name
					cmd.Env = slices.Clone(env)
					cmd.Env = append(cmd.Env, "PROMETHEUS_MCP_TEST_PROCESS="+mode)
					if mode == "timeout" {
						cmd.Env = append(cmd.Env, "PROMETHEUS_MCP_PROMETHEUS_TIMEOUT=invalid-duration")
					}
					output, err := cmd.CombinedOutput()
					require.NoError(t, ctx.Err(), "subprocess timed out: %s", output)
					if mode == "help" {
						require.NoError(t, err, "%s", output)
						for _, variable := range []string{
							"PROMETHEUS_MCP_PROMETHEUS_URL",
							"PROMETHEUS_MCP_MCP_TRANSPORT",
							"PROMETHEUS_MCP_WEB_LISTEN_ADDRESS",
						} {
							require.Contains(t, string(output), variable)
						}
						require.NotContains(t, string(output), "PROMETHEUS_MCP_EXE_")
					} else {
						require.Error(t, err)
						require.Contains(t, string(output), `invalid duration "invalid-duration"`)
					}
				})
			}
		})
	}
}
