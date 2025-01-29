aws sqs create-queue --queue-name keda-test --region ${AWS_REGION} --output json

kubectl create ns sqs-consumer
kubectl create deployment sqs-consumer --image nginx -n sqs-consumer

kubectl apply -f ScaleObject.yaml

kubectl get hpa -n sqs-consumer -w

kubectl get pods -n sqs-consumer -w

x=2
a=0
while [ $a -lt $x ]
do
   aws sqs send-message --region us-east-1 --endpoint-url https://sqs.us-east-1.amazonaws.com/ --queue-url https://sqs.us-east-1.amazonaws.com/571653956102/keda-test  --message-body '{"key": "value"}'
   a=`expr $a + 1`
done

aws sqs purge-queue --queue-url "https://sqs.us-east-1.amazonaws.com/571653956102/keda-test"  



kubectl create ns keda-karpenter-scaling
kubectl config set-context --current --namespace=keda
# First create the deployment
kubectl create deployment nginx-deployment --image nginx --replicas=2

# Then set the resource requests
kubectl set resources deployment nginx-deployment --requests=cpu=1,memory=3Gi

for i in 1..2
do
  aws sqs send-message \
  --queue-url $(aws sqs get-queue-url --queue-name keda-test) \
  --message-body "Keda and Karpenter demo"
done