import { Construct } from "constructs";
import { App, Chart } from "cdk8s";
import * as kplus from "cdk8s-plus-31";
import { Size } from "cdk8s"; // Import Size from cdk8s core

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
    ingress.addRule("/foo", this.echoBackend("foo"));
    ingress.addRule("/foo/bar", this.echoBackend("foo-bar"));
  }

  private echoBackend(text: string) {
    // Create deployment for each path
    const deploy = new kplus.Deployment(this, text, {
      containers: [
        {
          image: "hashicorp/http-echo",
          args: ["-text", text],
          portNumber: 5678,
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
    return kplus.IngressBackend.fromService(
      deploy.exposeViaService({
        ports: [{ port: 80, targetPort: 5678 }],
      })
    );
  }
}

const app = new App();
new HttpEcho(app, "http-echo");
app.synth();
