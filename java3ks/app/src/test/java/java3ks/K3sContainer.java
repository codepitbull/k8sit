package java3ks;

import com.github.dockerjava.api.command.InspectContainerResponse;
import io.fabric8.kubernetes.client.Config;
import org.testcontainers.containers.GenericContainer;
import org.testcontainers.containers.wait.strategy.LogMessageWaitStrategy;
import org.testcontainers.utility.DockerImageName;

import java.nio.charset.StandardCharsets;

public class K3sContainer extends GenericContainer<K3sContainer> {

    private Config kubeConfig;

    public K3sContainer(DockerImageName dockerImageName) {
        super(dockerImageName);
        addExposedPorts(6443, 8443);
        setPrivilegedMode(true);
        setCommand("server", "--no-deploy=traefik", "--token=abc123", "--tls-san=127.0.0.1");
        setWaitStrategy(new LogMessageWaitStrategy().withRegEx(".*Node controller sync successful.*"));
    }

    @Override
    protected void containerIsStarted(InspectContainerResponse containerInfo) {
        String rawKubeConfig = copyFileFromContainer(
                "/etc/rancher/k3s/k3s.yaml",
                is -> new String(is.readAllBytes(), StandardCharsets.UTF_8)
        );

        kubeConfig = Config.fromKubeconfig("default", rawKubeConfig, null);
        kubeConfig.setClientKeyAlgo("EC");
        kubeConfig.setMasterUrl("https://" + this.getHost() + ":" + this.getMappedPort(6443));
    }

    public Config getKubeConfig() {
        return kubeConfig;
    }
}
