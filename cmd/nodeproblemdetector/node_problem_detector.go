/*
Copyright 2016 The Kubernetes Authors All rights reserved.

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

package main

import (
	"context"
	"fmt"
	"net"

	"k8s.io/klog/v2"

	"k8s.io/client-go/kubernetes"
	_ "k8s.io/node-problem-detector/cmd/nodeproblemdetector/exporterplugins"
	_ "k8s.io/node-problem-detector/cmd/nodeproblemdetector/problemdaemonplugins"
	"k8s.io/node-problem-detector/cmd/options"
	"k8s.io/node-problem-detector/pkg/exporters"
	"k8s.io/node-problem-detector/pkg/exporters/k8sexporter"
	"k8s.io/node-problem-detector/pkg/exporters/prometheusexporter"
	"k8s.io/node-problem-detector/pkg/k8sclient"
	"k8s.io/node-problem-detector/pkg/problemdaemon"
	"k8s.io/node-problem-detector/pkg/problemdetector"
	"k8s.io/node-problem-detector/pkg/types"
	"k8s.io/node-problem-detector/pkg/version"
	ctrl "sigs.k8s.io/controller-runtime"
)

func npdMain(ctx context.Context, npdo *options.NodeProblemDetectorOptions) error {
	if npdo.PrintVersion {
		version.PrintVersion()
		return nil
	}

	npdo.SetNodeNameOrDie()
	npdo.SetConfigFromDeprecatedOptionsOrDie()
	npdo.ValidOrDie()

	client, err := kubernetes.NewForConfig(ctrl.GetConfigOrDie())
	if err != nil {
		return err
	}

	// Initialize br-mgmt and br-storagepub ip to k8s node label.
	err = initNodeIfaceIP(client)
	if err != nil {
		klog.Errorf("init node iface ip failed: %v\n", err)
		return err
	}

	// Initialize problem daemons.
	problemDaemons := problemdaemon.NewProblemDaemons(npdo.MonitorConfigPaths)
	if len(problemDaemons) == 0 {
		klog.Fatalf("No problem daemon is configured")
	}

	// Initialize exporters.
	defaultExporters := []types.Exporter{}
	if ke := k8sexporter.NewExporterOrDie(ctx, client, npdo); ke != nil {
		defaultExporters = append(defaultExporters, ke)
		klog.Info("K8s exporter started.")
	}
	if pe := prometheusexporter.NewExporterOrDie(npdo); pe != nil {
		defaultExporters = append(defaultExporters, pe)
		klog.Info("Prometheus exporter started.")
	}

	plugableExporters := exporters.NewExporters()

	npdExporters := []types.Exporter{}
	npdExporters = append(npdExporters, defaultExporters...)
	npdExporters = append(npdExporters, plugableExporters...)

	if len(npdExporters) == 0 {
		klog.Fatalf("No exporter is successfully setup")
	}

	// Initialize NPD core.
	p := problemdetector.NewProblemDetector(problemDaemons, npdExporters)
	return p.Run(ctx)
}

func initNodeIfaceIP(client *kubernetes.Clientset) error {
	cli, err := k8sclient.NewClient(client)
	if err != nil {
		return err
	}

	ctx := context.Background()
	node, err := cli.GetNode(ctx)
	if err != nil {
		return fmt.Errorf("get node failed: %v\n", err)
	}
	if node.Labels == nil {
		node.Labels = make(map[string]string)
	}
	if node.Labels[k8sclient.IfaceMgmtLabel] != "" &&
		node.Labels[k8sclient.IfaceStoragepubLabel] != "" {
		klog.Infof("iface ips are existed, br-mgmt ip: %s, br-storagepub ip: %s", node.Labels[k8sclient.IfaceMgmtLabel], node.Labels[k8sclient.IfaceStoragepubLabel])
		return nil
	}
	// config in /etc/sysconfig/network-scripts/ifcfg-*
	// can get by `ip addr show *`
	if node.Labels[k8sclient.IfaceMgmtLabel] == "" {
		ifaceIp, err := getIfaceIp(k8sclient.IfaceMgmt)
		if err != nil {
			klog.Errorf("get iface %s failed: %v\n", k8sclient.IfaceMgmt, err)
			return err
		}
		node.Labels[k8sclient.IfaceMgmtLabel] = ifaceIp
	}
	if node.Labels[k8sclient.IfaceStoragepubLabel] == "" {
		ifaceIp, err := getIfaceIp(k8sclient.IfaceStoragepub)
		if err != nil {
			klog.Errorf("get iface %s failed: %v\n", k8sclient.IfaceStoragepub, err)
			return err
		}
		node.Labels[k8sclient.IfaceStoragepubLabel] = ifaceIp
	}
	// update node labels
	_, err = cli.UpdateNode(ctx, node)
	if err != nil {
		return err
	}

	return nil
}

func getIfaceIp(name string) (string, error) {
	iface, err := net.InterfaceByName(name)
	if err != nil {
		return "", fmt.Errorf("iface %s get error: %v\n", name, err)
	}
	if iface == nil {
		return "", fmt.Errorf("iface %s not found\n", name)
	}
	addrs, err := iface.Addrs()
	if err != nil {
		return "", err
	}
	if len(addrs) == 0 {
		return "", fmt.Errorf("iface %s is down\n", name)
	}
	// addrs[0] demo: "192.168.10.11/24"
	res := addrs[0].(*net.IPNet).IP.String()
	return res, nil
}
