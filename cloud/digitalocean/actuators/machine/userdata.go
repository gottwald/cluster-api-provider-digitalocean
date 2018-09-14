package machine

import (
	"bytes"
	"fmt"
	"text/template"

	clusterv1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

// userdataParams are parameters used to parse the environment variables for bootstrap scripts.
type userdataParams struct {
	Machine        *clusterv1.Machine
	Cluster        *clusterv1.Cluster
	Token          string
	MasterEndpoint string
	PodCIDR        string
	ServiceCIDR    string
}

// masterUserdata builds master bootstrap script based on script provided in providerConfig and based on environment template.
func masterUserdata(cluster *clusterv1.Cluster, machine *clusterv1.Machine, token, metadata string) (string, error) {
	params := userdataParams{
		Cluster:     cluster,
		Machine:     machine,
		Token:       token,
		PodCIDR:     subnet(cluster.Spec.ClusterNetwork.Pods),
		ServiceCIDR: subnet(cluster.Spec.ClusterNetwork.Services),
	}
	tmpl := template.Must(template.New("masterEnvironment").Parse(masterEnvironmentVariables))

	b := &bytes.Buffer{}
	err := tmpl.Execute(b, params)
	if err != nil {
		return "", fmt.Errorf("failed to execute user-data template: %v", err)
	}
	b.Write([]byte(metadata))

	return b.String(), nil
}

// nodeUserdata builds node bootstrap script based on script provided in providerConfig and based on environment template.
func nodeUserdata(cluster *clusterv1.Cluster, machine *clusterv1.Machine, token, metadata string) (string, error) {
	params := userdataParams{
		Cluster:        cluster,
		Machine:        machine,
		Token:          token,
		MasterEndpoint: endpoint(cluster.Status.APIEndpoints[0]),
		PodCIDR:        subnet(cluster.Spec.ClusterNetwork.Pods),
		ServiceCIDR:    subnet(cluster.Spec.ClusterNetwork.Services),
	}
	tmpl := template.Must(template.New("nodeEnvironment").Parse(nodeEnvironmentVariables))

	b := &bytes.Buffer{}
	err := tmpl.Execute(b, params)
	if err != nil {
		return "", fmt.Errorf("failed to execute user-data template: %v", err)
	}
	b.Write([]byte(metadata))

	return b.String(), nil
}

func endpoint(apiEndpoint clusterv1.APIEndpoint) string {
	return fmt.Sprintf("%s:%d", apiEndpoint.Host, apiEndpoint.Port)
}

// subnet gets first IP of the subnet.
func subnet(netRange clusterv1.NetworkRanges) string {
	if len(netRange.CIDRBlocks) == 0 {
		return ""
	}
	return netRange.CIDRBlocks[0]
}

const (
	// masterEnvironment is the environment variables template for master instances.
	masterEnvironmentVariables = `#!/bin/bash
KUBELET_VERSION={{ .Machine.Spec.Versions.Kubelet }}
TOKEN={{ .Token }}
PORT=443
NAMESPACE={{ .Machine.ObjectMeta.Namespace }}
MACHINE=$NAMESPACE
MACHINE+="/"
MACHINE+={{ .Machine.ObjectMeta.Name }}
CONTROL_PLANE_VERSION={{ .Machine.Spec.Versions.ControlPlane }}
CLUSTER_DNS_DOMAIN={{ .Cluster.Spec.ClusterNetwork.ServiceDomain }}
POD_CIDR={{ .PodCIDR }}
SERVICE_CIDR={{ .ServiceCIDR }}
`
	nodeEnvironmentVariables = `#!/bin/bash
KUBELET_VERSION={{ .Machine.Spec.Versions.Kubelet }}
MASTER={{ .MasterEndpoint }}
TOKEN={{ .Token }}
NAMESPACE={{ .Machine.ObjectMeta.Namespace }}
MACHINE=$NAMESPACE
MACHINE+="/"
MACHINE+={{ .Machine.ObjectMeta.Name }}
CLUSTER_DNS_DOMAIN={{ .Cluster.Spec.ClusterNetwork.ServiceDomain }}
POD_CIDR={{ .PodCIDR }}
SERVICE_CIDR={{ .ServiceCIDR }}
`
)