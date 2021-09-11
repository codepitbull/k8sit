package k8sit

import (
	"context"
	"errors"
	"fmt"
	"github.com/docker/go-connections/nat"
	"k8s.io/client-go/tools/clientcmd/api"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/clientcmd"
	apiv1 "k8s.io/client-go/tools/clientcmd/api/v1"
	"k8s.io/client-go/util/homedir"
	"sigs.k8s.io/yaml"
)

type K8sClientSet struct {
	kubernetes.Clientset
}

func CreateClientSetFromBytes(configB []byte, host string, port nat.Port) (error, *K8sClientSet) {

	getter := func() (*api.Config, error) {
		config := &apiv1.Config{}
		err := yaml.Unmarshal(configB, &config)

		if err != nil {
			return nil, err
		}

		rawPort := strings.Replace(string(port), "/tcp", "", 1)
		config.Clusters[0].Cluster.Server = "https://" + host +":" + rawPort + "/"

		apiconfig := &api.Config{}
		err = apiv1.Convert_v1_Config_To_api_Config(config, apiconfig, nil)
		if err != nil {
			return nil, err
		}
		return apiconfig, err
	}

	config, err := clientcmd.BuildConfigFromKubeconfigGetter("", getter)
	if err != nil {
		panic(err.Error())
	}

	//kubeConfig.setClientKeyAlgo("EC");
	//kubeConfig.setMasterUrl("https://" + this.getHost() + ":" + this.getMappedPort(6443));

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	return err, &K8sClientSet{*clientset}
}

func CreateClientSetFromKubeconfigEnv() (error, *K8sClientSet) {
	kubeconfigLocation := os.Getenv("KUBECONFIG")

	if kubeconfigLocation == "" && homedir.HomeDir() != "" {
		kubeconfigLocation = filepath.Join(homedir.HomeDir(), ".kube", "config")
	} else if homedir.HomeDir() == "" {
		return errors.New("KUBECONFIG not set and couldn't get homedir"), nil
	}

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigLocation)
	if err != nil {
		panic(err.Error())
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	return err, &K8sClientSet{*clientset}
}

func (k K8sClientSet) CreateDeployment(deployment *appsv1.Deployment) (*appsv1.Deployment, error) {
	ret, err := k.AppsV1().Deployments(deployment.Namespace).Create(context.TODO(), deployment, metav1.CreateOptions{})
	return ret, err
}

func (k K8sClientSet) RemoveDeployment(deployment *appsv1.Deployment) error {
	err := k.AppsV1().Deployments(deployment.Namespace).Delete(context.TODO(), deployment.Name, metav1.DeleteOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		return err
	}
	for {
		_, err = k.AppsV1().Deployments(deployment.Namespace).Get(context.TODO(), deployment.Name, metav1.GetOptions{})
		if err != nil && !k8serrors.IsNotFound(err) {
			return err
		}
		if err != nil && k8serrors.IsNotFound(err) {
			return nil
		}
	}
}

func (k K8sClientSet) CreateService(service *v1.Service) (*v1.Service, error) {
	ret, err := k.CoreV1().Services(service.Namespace).Create(context.TODO(), service, metav1.CreateOptions{})
	return ret, err
}

func (k K8sClientSet) RemoveService(service *v1.Service) error {
	err := k.CoreV1().Services(service.Namespace).Delete(context.TODO(), service.Name, metav1.DeleteOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		return err
	}
	for {
		_, err = k.CoreV1().Services(service.Namespace).Get(context.TODO(), service.Name, metav1.GetOptions{})
		if err != nil && !k8serrors.IsNotFound(err) {
			return err
		}
		if err != nil && k8serrors.IsNotFound(err) {
			return nil
		}
	}
}

func (k K8sClientSet) CreateTempNamespace() (string, error) {
	uuid := CreateUniqueString()
	namespace := v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: uuid},
	}
	ns, err := k.CoreV1().Namespaces().Create(context.TODO(), &namespace, metav1.CreateOptions{})
	if err != nil {
		return "", err
	}

	err = k.CreateDockerSecret(namespace.Name)
	return ns.Name, nil
}

func (k K8sClientSet) DeleteNamespace(name string) error {
	err := k.CoreV1().Namespaces().Delete(context.TODO(), name, metav1.DeleteOptions{})
	if err != nil && k8serrors.IsNotFound(err) {
		return nil
	}
	return err
}

func (k K8sClientSet) AwaitDeploymentReady(name string, namespace string, timeoutSecs int64) error {
	start := time.Now().Unix()
	stop := start + timeoutSecs

	for {
		time.Sleep(1 * time.Second)
		deployment, err := k.AppsV1().Deployments(namespace).Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil && k8serrors.IsNotFound(err) {
			continue
		}
		if err != nil {
			return err
		}
		if deployment.Status.ReadyReplicas > 0 {
			return nil
		}

		if time.Now().Unix() >= stop {
			return errors.New("Timed out waiting for deployment")
		}
	}
}

//This relies on rancher.io/local-path being installed
func (k K8sClientSet) CreateLocalPathPvc(name string, namespace string) error {
	storageClassName := "standard"
	pvc := &v1.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PersistentVolumeClaim",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1.PersistentVolumeClaimSpec{
			AccessModes: []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
			Selector:    nil,
			Resources: v1.ResourceRequirements{
				Requests: map[v1.ResourceName]resource.Quantity{
					v1.ResourceStorage: *resource.NewQuantity(2*1024*1024*1024, resource.BinarySI),
				},
			},
			StorageClassName: &storageClassName,
		},
	}
	_, err := k.CoreV1().PersistentVolumeClaims(namespace).Create(context.TODO(), pvc, metav1.CreateOptions{})
	return err
}

func (k K8sClientSet) CreateLocalPathPvcs(names []string, namespace string) error {
	for _, name := range names {
		err := k.DeleteLocalPathPvc(name, namespace)
		if err != nil && !k8serrors.IsNotFound(err) {
			return err
		}
		err = k.CreateLocalPathPvc(name, namespace)
		if err != nil {
			return err
		}
	}
	return nil
}

func (k K8sClientSet) DeleteLocalPathPvc(name string, namespace string) error {
	err := k.CoreV1().PersistentVolumeClaims(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		return err
	}
	for {
		_, err = k.CoreV1().PersistentVolumeClaims(namespace).Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil && !k8serrors.IsNotFound(err) {
			return err
		}
		if err != nil && k8serrors.IsNotFound(err) {
			return nil
		}
	}
}

func (k K8sClientSet) DeleteLocalPathPvcs(names []string, namespace string) error {
	for _, name := range names {
		err := k.DeleteLocalPathPvc(name, namespace)
		if err != nil && !k8serrors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

func (k K8sClientSet) CreateDockerSecret(namespace string) error {

	username := os.Getenv("DOCKER_USER")
	password := os.Getenv("DOCKER_PASSWORD")

	if username == "" || password == "" {
		panic("Running tests requires defining the envs DOCKER_USER and DOCKER_PASSWORD")
	}

	config := fmt.Sprintf("{\"auths\":{\"%s\":{\"username\": \"%s\",\"password\":\"%s\",\"email\": \"%s\"}}}", "https://containers.instana.io", username, password, "")

	secrets := make(map[string][]byte)
	secrets[".dockerconfigjson"] = []byte(config)
	secret := &v1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   "instana-registry",
		},
		Data: secrets,
		Type: "kubernetes.io/dockerconfigjson",
	}
	_, error := k.CoreV1().Secrets(namespace).Create(context.TODO(), secret, metav1.CreateOptions{})

	return error
}




func CreateUniqueString() string {
	uuidWithHyphen := uuid.New()
	uuid := strings.Replace(uuidWithHyphen.String(), "-", "", -1)
	return uuid
}

func SkipK8sIT(t *testing.T) {
	if os.Getenv("K8SIT") != "true" {
		t.Skip("Skipping integration tests")
		t.SkipNow()
	}
}
