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

package netchecker

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"k8s.io/node-problem-detector/cmd/netchecker/options"
	"k8s.io/node-problem-detector/pkg/k8sclient"
	"k8s.io/node-problem-detector/pkg/netchecker/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

type netchecker struct {
	networkInterface string
	healthCheckFunc  func() (bool, error)
	client           k8sclient.Client
	// The repair is "best-effort" and ignores the error from the underlying actions.
	// The bash commands to kill the process will fail if the service is down and hence ignore.
	// repairFunc         func()
	// uptimeFunc         func() (time.Duration, error)
	// crictlPath         string
	// netCheckTimeout    time.Duration
	// loopBackTime       time.Duration
	// logPatternsToCheck map[string]int
}

// Newnetchecker returns a new health checker configured with the given options.
func Newnetchecker(hco *options.NetcheckerOptions) (types.Netchecker, error) {
	hc := &netchecker{
		networkInterface: hco.NetworkInterface,
		// netCheckTimeout:    hco.HealthCheckTimeout,
	}
	// k8sclient
	client, err := kubernetes.NewForConfig(ctrl.GetConfigOrDie())
	if err != nil {
		return nil, err
	}
	c, err := k8sclient.NewClient(client)
	if err != nil {
		klog.Errorf("Failed to create k8s client: %v", err)
		return nil, err
	}
	hc.client = c

	hc.healthCheckFunc = getHealthCheckFunc(hc.client, hco)

	return hc, nil
}

// CheckHealth checks for the health of the component and tries to repair if enabled.
// Returns true if healthy, false otherwise.
func (hc *netchecker) CheckHealth() (bool, error) {
	healthy, err := hc.healthCheckFunc()
	return healthy, err
}

// 通过 Flags 判断接口状态
func isInterfaceUp(iface *net.Interface) bool {
	return iface.Flags&net.FlagUp != 0
}

func netCheckMgmt(client k8sclient.Client, iface string) func() (bool, error) {
	return func() (bool, error) {
		// 1 check network interface status
		iface, err := net.InterfaceByName(iface)
		if err != nil {
			return false, fmt.Errorf("Error getting network interfaces: %v\n", err)
		}
		// not found
		if iface == nil {
			return false, fmt.Errorf("Interface %s not found\n", "br-storagepub")
		}
		// check interface status
		if !isInterfaceUp(iface) {
			return false, nil
		}

		// 2 check network connectivity
		cm, err := client.GetConfigMap(context.Background(), "openstack", "keepalived-etc")
		if err != nil {
			return false, fmt.Errorf("Error getting keepalived-etc cm: %v\n", err)
		}
		conf, exists := cm.Data["keepalived.conf"]
		if !exists {
			return false, fmt.Errorf("keepalived.conf not found in cm keepalived-etc")
		}
		// extract vrrp_instance VI_mgmt_vip
		instanceName := "VI_mgmt_vip"
		pattern := fmt.Sprintf(
			`(?s)vrrp_instance %s \{.*?virtual_ipaddress\s*\{([^}]+)\}`,
			regexp.QuoteMeta(instanceName))

		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(conf)
		if len(matches) < 2 {
			klog.Errorf("vrrp_instance not found in keepalived.conf, matches: %v\n", matches)
			return false, fmt.Errorf("not found vrrp_instance %s", instanceName)
		}
		// extract ip
		ipRe := regexp.MustCompile(`\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`)
		ips := ipRe.FindAllString(matches[1], -1)

		var targetIPs []string
		for _, ip := range ips {
			ip = strings.TrimSpace(ip)
			if ip != "" {
				targetIPs = append(targetIPs, ip)
			}
		}
		if len(targetIPs) == 0 {
			return false, fmt.Errorf("no valid IP addresses found for vrrp_instance %s", instanceName)
		}
		// ping random IP address
		ip := targetIPs[0]
		cmd := exec.Command("ping", "-c", "2", ip)
		output, err := cmd.CombinedOutput()
		if err != nil {
			klog.Errorf("ping mgmt vip failed: %v\n", err)
			return false, fmt.Errorf("ping mgmt vip failed: %v\n", err)
		} else if strings.Contains(string(output), "100% packet loss") {
			klog.Infof("ping mgmt vip failed: %s\n", ip)
			return false, nil
		}

		return true, nil
	}
}

// netCheckOKFunc returns a function to check the status of a network interface.
func netCheckStoragepub(client k8sclient.Client, ifaceName string, hco *options.NetcheckerOptions) func() (bool, error) {
	return func() (bool, error) {
		// 1 check network interface status
		iface, err := net.InterfaceByName(ifaceName)
		if err != nil {
			return false, fmt.Errorf("Error getting network interfaces: %v\n", err)
		}
		// not found
		if iface == nil {
			return false, fmt.Errorf("Interface %s not found\n", ifaceName)
		}
		// check interface status
		if !isInterfaceUp(iface) {
			return false, nil
		}

		// 2 check network connectivity
		nodeName := os.Getenv("NODE_NAME")
		if nodeName == "" {
			return false, fmt.Errorf("NODE_NAME is not set")
		}
		ep, err := client.GetEndpoints(context.Background(), hco.CephMonNamespace, hco.CephMonName)
		if err != nil {
			return false, fmt.Errorf("Error getting ceph-mon endpoints: %v\n", err)
		}
		// filter out the IP addresses of the current node
		targetIPs := []string{}
		for _, subset := range ep.Subsets {
			for _, addr := range subset.Addresses {
				if addr.NodeName == nil || *addr.NodeName != nodeName {
					targetIPs = append(targetIPs, addr.IP)
				}
			}
		}
		// ping random IP address
		if len(targetIPs) == 0 {
			return false, fmt.Errorf("No external ceph-mon endpoints found\n")
		}
		ip := targetIPs[0]
		cmd := exec.Command("ping", "-c", "2", ip)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return false, fmt.Errorf("ping ceph-mon failed: %v\n", err)
		} else if strings.Contains(string(output), "100% packet loss") {
			return false, fmt.Errorf("ping ceph-mon failed: %s\n", ip)
		}
		// add ceph blacklist
		// entityaddr := fmt.Sprintf("%s:0/0", ip)
		// err = addCephBlacklist("hostha", "/etc/ceph/ceph.conf", "rbd", entityaddr, 15)

		return true, nil
	}
}

func addCephBlacklist(userID string, configFile string, poolName string, blacklistEntry string, duration int) error {

	cmdArgs := []string{
		"--id", userID,
		"-c", configFile,
		"osd", "blacklist", "pool", poolName,
		"add", blacklistEntry,
		fmt.Sprintf("%d", duration),
	}

	// 创建命令对象
	cmd := exec.Command("ceph", cmdArgs...)

	// 执行命令并捕获输出
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to execute command: %v, output: %s", err, string(output))
	}

	klog.Infof("add ceph blacklist success: %s", string(output))
	return nil
}

// getHealthCheckFunc returns the health check function based on the NetworkInterface.
func getHealthCheckFunc(client k8sclient.Client, hco *options.NetcheckerOptions) func() (bool, error) {
	switch hco.NetworkInterface {
	case k8sclient.IfaceMgmt:
		return netCheckMgmt(client, k8sclient.IfaceMgmt)
	case k8sclient.IfaceStoragepub:
		return netCheckStoragepub(client, k8sclient.IfaceStoragepub, hco)
	default:
		klog.Warningf("Unsupported network interface: %v", hco.NetworkInterface)
	}

	return nil
}

// execCommand executes the bash command and returns the (output, error) from command, error if timeout occurs.
func execCommand(timeout time.Duration, command string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, command, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		klog.Infof("command %v failed: %v, %s\n", cmd, err, string(out))
		return "", err
	}

	return strings.TrimSuffix(string(out), "\n"), nil
}
