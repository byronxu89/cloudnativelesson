# download the latest image
docker pull acleverguy/simplehttpserver:k8s

#create configmap and variables for the pod environment variable
kubectl apply -f simplehttpconfig.yaml

#create the pods and choose the guarantee qos 
kubectl apply -f simplehttpdeploy.yaml

#create the service for the deployment
kubectl apply -f simplehttpsvc.yaml