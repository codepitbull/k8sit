package java3ks;

import io.fabric8.kubernetes.api.model.*;
import io.fabric8.kubernetes.api.model.apps.*;
import io.fabric8.kubernetes.client.DefaultKubernetesClient;
import io.fabric8.kubernetes.client.KubernetesClient;
import org.junit.jupiter.api.Test;
import org.testcontainers.junit.jupiter.Container;
import org.testcontainers.junit.jupiter.Testcontainers;
import org.testcontainers.utility.DockerImageName;

import java.util.Collections;
import java.util.Map;

import static java.util.concurrent.TimeUnit.SECONDS;
import static org.awaitility.Awaitility.await;

@Testcontainers
public class K3sTest {

    @Container
    private final K3sContainer k3s = new K3sContainer(DockerImageName.parse("rancher/k3s:latest"));

    @Test void testK3s() throws Exception{
        KubernetesClient client = new DefaultKubernetesClient(k3s.getKubeConfig());
        Deployment depl = client.apps().deployments().inNamespace("default").create(nginxDeployment());

        await().atMost(100, SECONDS).until(() -> {
            DeploymentList deplList = client.apps().deployments().inAnyNamespace().list();
            return deplList
                    .getItems()
                    .stream()
                    .filter(listedDepl -> listedDepl.getMetadata().getName().equals(depl.getMetadata().getName()))
                    .anyMatch(nginxDepl -> {
                        Integer replicas = nginxDepl.getStatus().getReadyReplicas();
                        return replicas != null &&  replicas == 1;
                    });
        });
    }

    private Deployment nginxDeployment() {

        return new DeploymentBuilder()
                .withMetadata(new ObjectMetaBuilder().withName("nginx").build())
                .withSpec(new DeploymentSpecBuilder()
                    .withSelector(new LabelSelectorBuilder().withMatchLabels(Map.of("app","nginx")).build())
                    .withTemplate(new PodTemplateSpecBuilder()
                        .withMetadata(new ObjectMetaBuilder()
                            .withName("nginx")
                            .withNamespace("default")
                            .withLabels(Map.of("app","nginx")).build())
                        .withSpec(
                            new PodSpecBuilder()
                                .withContainers(
                                    new ContainerBuilder()
                                        .withName("nginx")
                                        .withImage("nginx:1.14.2")
                                        .withPorts(new ContainerPortBuilder().withContainerPort(80).build())
                                        .build()
                                ).build()).build()).build()).build();
    }
}
