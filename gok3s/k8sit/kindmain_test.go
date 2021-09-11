package k8sit



import (
	"os"
	"path/filepath"
	"sigs.k8s.io/kind/pkg/cmd"
	createcluster "sigs.k8s.io/kind/pkg/cmd/kind/create/cluster"
	deletecluster "sigs.k8s.io/kind/pkg/cmd/kind/delete/cluster"
	"testing"
)

//Formerly TestMain
func MainKind(m *testing.M) {
	clusterName := "mine"
	kubeConfigPath := filepath.Join(os.TempDir(), clusterName)
	logger := cmd.NewLogger()
	streams:= cmd.StandardIOStreams()
	create := createcluster.NewCommand(logger, streams)
	create.SetArgs([]string{"--name", clusterName, "--kubeconfig", kubeConfigPath})
	os.Setenv("KUBECONFIG", kubeConfigPath)
	err := create.Execute()
	if err != nil {
		panic(err)
	}

	code := m.Run()

	delete := deletecluster.NewCommand(logger, streams)
	delete.SetArgs([]string{"--name", clusterName, "--kubeconfig", kubeConfigPath})
	err = delete.Execute()
	if err != nil {
		panic(err)
	}

	os.Exit(code)
}