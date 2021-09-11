package k8sit

import (
	"context"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"io/ioutil"
	appsv1 "k8s.io/api/apps/v1"
	v12 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestK3s(t *testing.T) {
	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:        "rancher/k3s:latest",
		ExposedPorts: []string{"6443/tcp","8443/tcp"},
		Privileged: true,
		Cmd: []string{"server", "--no-deploy=traefik", "--token=abc123", "--tls-san=127.0.0.1"},
		WaitingFor:   wait.ForLog("Node controller sync successful"),
	}
	k3sC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})

	require.NoError(t, err)

	defer k3sC.Terminate(ctx)

	reader, err := k3sC.CopyFileFromContainer(ctx, "/etc/rancher/k3s/k3s.yaml")
	require.NoError(t, err)

	res,err := ioutil.ReadAll(reader)
	require.NoError(t, err)

	port, err := k3sC.MappedPort(ctx, "6443/tcp")
	require.NoError(t, err)
	host, err := k3sC.Host(ctx)
	require.NoError(t, err)

	err, client := CreateClientSetFromBytes(res, host, port)
	require.NoError(t, err)

	depl := Deployment()
	depl, err = client.CreateDeployment(depl)
	require.NoError(t, err)

	err = client.AwaitDeploymentReady(depl.Name, depl.Namespace, 120)
	require.NoError(t, err)
}

func Deployment() *appsv1.Deployment {
	deployment := &appsv1.Deployment{}
	deployment.TypeMeta = metav1.TypeMeta{
		Kind:       "Deployment",
		APIVersion: "apps/v1",
	}
	deployment.ObjectMeta = metav1.ObjectMeta{
		Name:      "nginx-deployment",
		Namespace: "default",
	}
	deployment.Spec = appsv1.DeploymentSpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"app": "nginx",
			},
		},
		Template: v12.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"app": "nginx",
				},
			},
			Spec:       v12.PodSpec{
				Containers: []v12.Container{{
					Name:                     "nginx",
					Image:                    "nginx:1.14.2",
					Ports:                    []v12.ContainerPort{{
						ContainerPort: 80,
					}},
				}},
			},
		},
	}
	return deployment
}
