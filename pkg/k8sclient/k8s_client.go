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

package k8sclient

import (
	"context"
	"fmt"
	"net/url"
	"os"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/utils/clock"
)

const (
	IfaceMgmt            = "br-mgmt"
	IfaceStoragepub      = "br-storagepub"
	IfaceMgmtLabel       = "ecns.easystack.io/br-mgmt.ip"
	IfaceStoragepubLabel = "ecns.easystack.io/br-storagepub.ip"
)

// Client is the interface of k8sclient
type Client interface {
	GetNode(ctx context.Context) (*v1.Node, error)
	GetEndpoints(ctx context.Context, namespace string, service string) (*v1.Endpoints, error)
	GetConfigMap(ctx context.Context, namespace string, name string) (*v1.ConfigMap, error)
	UpdateNode(ctx context.Context, node *v1.Node) (*v1.Node, error)
}

type k8sClient struct {
	nodeName string
	client   *kubernetes.Clientset
	clock    clock.Clock
}

var _ Client = &k8sClient{}

// NewClientOrDie creates a new problem client, panics if error occurs.
func NewClientOverride(apiServerOverride string) (Client, error) {
	c := &k8sClient{clock: clock.RealClock{}}

	// we have checked it is a valid URI after command line argument is parsed.:)
	uri, _ := url.Parse(apiServerOverride)
	cfg, err := getKubeClientConfig(uri)
	if err != nil {
		return nil, err
	}

	c.client = clientset.NewForConfigOrDie(cfg)

	nodeName := os.Getenv("NODE_NAME")
	if nodeName == "" {
		return nil, fmt.Errorf("NODE_NAME is not set")
	}

	c.nodeName = nodeName
	return c, nil
}

// use nodes's kubeconfig by set flag --kubeconfig
// it connects with apiserver by br-roller
// so it can connect with apiserver while br-mgmt and br-storagepub are down
func NewClient(client *kubernetes.Clientset) (Client, error) {
	c := &k8sClient{clock: clock.RealClock{}}
	c.client = client

	nodeName := os.Getenv("NODE_NAME")
	if nodeName == "" {
		return nil, fmt.Errorf("NODE_NAME is not set")
	}

	c.nodeName = nodeName
	return c, nil
}

func (c *k8sClient) GetNode(ctx context.Context) (*v1.Node, error) {
	// To reduce the load on APIServer & etcd, we are serving GET operations from
	// apiserver cache (the data might be slightly delayed).
	return c.client.CoreV1().Nodes().Get(ctx, c.nodeName, metav1.GetOptions{ResourceVersion: "0"})
}

func (c *k8sClient) GetEndpoints(ctx context.Context, namespace string,
	service string) (*v1.Endpoints, error) {
	return c.client.CoreV1().Endpoints(namespace).Get(ctx, service, metav1.GetOptions{ResourceVersion: "0"})
}

func (c *k8sClient) GetConfigMap(ctx context.Context, namespace string,
	name string) (*v1.ConfigMap, error) {
	return c.client.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{ResourceVersion: "0"})
}

func (c *k8sClient) UpdateNode(ctx context.Context, node *v1.Node) (*v1.Node, error) {
	return c.client.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
}
