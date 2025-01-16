// import { Construct } from "constructs";
// import { App, Chart, ChartProps } from "cdk8s";
// import {
//   KubeDeployment,
//   KubeService,
//   IntOrString,
//   KubeIngress,
// } from "./imports/k8s";

// export class MyChart extends Chart {
//   constructor(scope: Construct, id: string, props: ChartProps = {}) {
//     super(scope, id, props);

//     const label = { app: "hello-k8s" };

//     // Create deployment
//     new KubeDeployment(this, "deployment", {
//       spec: {
//         replicas: 2,
//         selector: {
//           matchLabels: label,
//         },
//         template: {
//           metadata: { labels: label },
//           spec: {
//             containers: [
//               {
//                 name: "nginx",
//                 image: "nginx:1.25", // Using nginx latest stable version
//                 ports: [{ containerPort: 80 }],
//               },
//             ],
//           },
//         },
//       },
//     });

//     // Create service
//     const service = new KubeService(this, "hello-service", {
//       spec: {
//         type: "ClusterIP",
//         ports: [{ port: 80, targetPort: IntOrString.fromNumber(80) }],
//         selector: label,
//       },
//     });

//     // Create ingress
//     new KubeIngress(this, "hello-ingress", {
//       metadata: {
//         annotations: {
//           "kubernetes.io/ingress.class": "alb",
//           "alb.ingress.kubernetes.io/scheme": "internet-facing",
//           "alb.ingress.kubernetes.io/target-type": "ip",
//           "alb.ingress.kubernetes.io/listen-ports": '[{"HTTP": 80}]',
//           "alb.ingress.kubernetes.io/manage-backend-security-group-rules":
//             "true",
//           "alb.ingress.kubernetes.io/group.name": "hello-app",
//         },
//       },

//       spec: {
//         ingressClassName: "alb",
//         rules: [
//           {
//             http: {
//               paths: [
//                 {
//                   path: "/",
//                   pathType: "Prefix",
//                   backend: {
//                     service: {
//                       name: service.name,
//                       port: {
//                         number: 80,
//                       },
//                     },
//                   },
//                 },
//               ],
//             },
//           },
//         ],
//       },
//     });
//   }
// }

// const app = new App();
// new MyChart(app, "hello-k8s");
// app.synth();
