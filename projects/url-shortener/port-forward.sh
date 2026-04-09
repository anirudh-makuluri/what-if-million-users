#!/bin/bash
# Port-forward all services for url-shortener Kubernetes deployment
# Usage: ./port-forward.sh [start|stop|list]

NAMESPACE="url-shortener"
ACTION="${1:-start}"

# Define port-forwards
declare -A SERVICES=(
    [url-shortener]="8083:8080:App API"
    [prometheus]="9091:9090:Prometheus"
    [grafana]="3000:3000:Grafana"
    [dynamodb-local]="8000:8000:DynamoDB"
    [kafka]="9092:9092:Kafka"
    [redis]="6379:6379:Redis"
)

case $ACTION in
    start)
        echo "Starting Kubernetes port-forwards for $NAMESPACE namespace..."
        echo ""
        
        for service in "${!SERVICES[@]}"; do
            IFS=':' read -r local_port remote_port name <<< "${SERVICES[$service]}"
            echo "  $name"
            echo "    localhost:$local_port -> $service:$remote_port"
            
            # Run port-forward in background
            kubectl port-forward svc/$service $local_port:$remote_port -n $NAMESPACE &
        done
        
        echo ""
        echo "Port-forwards started! You can access:"
        echo ""
        for service in "${!SERVICES[@]}"; do
            IFS=':' read -r local_port remote_port name <<< "${SERVICES[$service]}"
            echo "  $name:          http://localhost:$local_port"
        done
        echo ""
        echo "To stop all port-forwards, run: $0 stop"
        echo ""
        ;;
        
    stop)
        echo "Stopping all port-forwards..."
        pkill -f "kubectl port-forward"
        echo "All port-forwards stopped."
        ;;
        
    list)
        echo "Available port-forwards:"
        echo ""
        for service in "${!SERVICES[@]}"; do
            IFS=':' read -r local_port remote_port name <<< "${SERVICES[$service]}"
            echo "  $name: localhost:$local_port -> $service:$remote_port"
        done
        ;;
        
    *)
        echo "Usage: $0 [start|stop|list]"
        exit 1
        ;;
esac
