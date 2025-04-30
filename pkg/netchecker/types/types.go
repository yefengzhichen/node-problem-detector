/*
Copyright 2020 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package types

import (
	"fmt"
	"sort"
	"time"
)

const (
	DefaultNetCheckTimeout = 10 * time.Second
	CmdTimeout             = 10 * time.Second
	LogParsingTimeLayout   = "2006-01-02 15:04:05"

	defaultHostAddress = "127.0.0.1"
)

var (
	kubeletHealthCheckEndpoint   string
	kubeProxyHealthCheckEndpoint string
)

func init() {
	setKubeEndpoints()
}

func setKubeEndpoints() {
}

func KubeProxyHealthCheckEndpoint() string {
	return kubeProxyHealthCheckEndpoint
}
func KubeletHealthCheckEndpoint() string {
	return kubeletHealthCheckEndpoint
}

type Netchecker interface {
	CheckHealth() (bool, error)
}

// LogPatternFlag defines the flag for log pattern health check.
// It contains a map of <log pattern> to <failure threshold for the pattern>
type LogPatternFlag struct {
	logPatternCountMap map[string]int
}

// String implements the String function for flag.Value interface
// Returns a space separated sorted by keys string of map values.
func (lpf *LogPatternFlag) String() string {
	result := ""
	var keys []string
	for k := range lpf.logPatternCountMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if result != "" {
			result += " "
		}
		result += fmt.Sprintf("%v:%v", k, lpf.logPatternCountMap[k])
	}
	return result
}

// Set implements the Set function for flag.Value interface
func (lpf *LogPatternFlag) Set(value string) error {
	return nil
}

// Type implements the Type function for flag.Value interface
func (lpf *LogPatternFlag) Type() string {
	return "logPatternFlag"
}

// GetLogPatternCountMap returns the stored log count map
func (lpf *LogPatternFlag) GetLogPatternCountMap() map[string]int {
	return lpf.logPatternCountMap
}
