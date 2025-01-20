import { Construct } from "constructs";
import { App, Chart } from "cdk8s";
import * as kplus from "cdk8s-plus-31";
import { Size, Duration } from "cdk8s"; // Import Size from cdk8s core

export class HttpEcho extends Chart {
  constructor(scope: Construct, id: string) {
    super(scope, id);

    // Create ingress with AWS ALB annotations
    const ingress = new kplus.Ingress(this, "ingress", {
      metadata: {
        annotations: {
          "kubernetes.io/ingress.class": "alb",
          "alb.ingress.kubernetes.io/scheme": "internet-facing",
          "alb.ingress.kubernetes.io/target-type": "ip",
          "alb.ingress.kubernetes.io/listen-ports": '[{"HTTP": 80}]',
          "alb.ingress.kubernetes.io/group.name": "echo-app",
        },
      },
    });
    // Add different paths
    ingress.addRule("/", this.echoBackend("root"));
    // ingress.addRule("/foo", this.echoBackend("foo"));
    // ingress.addRule("/foo/bar", this.echoBackend("foo-bar"));
  }

  private echoBackend(text: string) {
    // Create deployment for each path
    const deploy = new kplus.Deployment(this, text, {
      containers: [
        {
          image: "nginx:1.19.10",
          // args: ["-text", text],
          portNumber: 80,
          securityContext: {
            ensureNonRoot: false,
            readOnlyRootFilesystem: false,
          },
          resources: {
            cpu: {
              request: kplus.Cpu.millis(100),
              limit: kplus.Cpu.millis(500),
            },
            memory: {
              request: Size.mebibytes(128),
              limit: Size.mebibytes(256),
            },
          },
        },
      ],
    });

    // Create service for the deployment
    const service = deploy.exposeViaService({
      ports: [{ port: 80, targetPort: 80 }],
    });

    // Add HorizontalPodAutoscaler
    new kplus.HorizontalPodAutoscaler(this, `${text}-hpa`, {
      target: deploy,
      minReplicas: 2,
      maxReplicas: 10,
      metrics: [
        kplus.Metric.resourceCpu(kplus.MetricTarget.averageUtilization(70)),
        kplus.Metric.resourceMemory(
          kplus.MetricTarget.averageUtilization(70)
          // kplus.MetricTarget.averageValue(512 * 1024 * 1024) // Convert 512Mi to bytes
        ),
      ],
      scaleUp: {
        strategy: kplus.ScalingStrategy.MAX_CHANGE,
        stabilizationWindow: Duration.seconds(60),
        policies: [
          {
            replicas: kplus.Replicas.absolute(4),
            duration: Duration.minutes(1),
          },
          {
            replicas: kplus.Replicas.percent(200),
            duration: Duration.minutes(1),
          },
        ],
      },
      scaleDown: {
        stabilizationWindow: Duration.minutes(5),
        policies: [
          {
            replicas: kplus.Replicas.absolute(2),
            duration: Duration.minutes(1),
          },
        ],
      },
    });

    return kplus.IngressBackend.fromService(service);
  }
}

const app = new App();
new HttpEcho(app, "nginx-demo");
app.synth();
