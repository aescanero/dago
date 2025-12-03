#!/bin/bash

# Deployment script for DA Orchestrator

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Variables
DEPLOYMENT_TYPE=${1:-"docker-compose"}
NAMESPACE=${NAMESPACE:-"dago"}

usage() {
    echo "Usage: $0 [deployment-type]"
    echo ""
    echo "Deployment types:"
    echo "  docker-compose  - Deploy with Docker Compose (default)"
    echo "  k8s            - Deploy to Kubernetes"
    echo "  helm           - Deploy with Helm"
    echo ""
    echo "Environment variables:"
    echo "  NAMESPACE      - Kubernetes namespace (default: dago)"
    echo "  LLM_API_KEY    - LLM API key (required)"
    exit 1
}

check_requirements() {
    if [ -z "${LLM_API_KEY}" ]; then
        echo -e "${RED}Error: LLM_API_KEY environment variable is required${NC}"
        exit 1
    fi
}

deploy_docker_compose() {
    echo -e "${GREEN}Deploying with Docker Compose${NC}"

    check_requirements

    # Create .env file
    cat > .env <<EOF
LLM_API_KEY=${LLM_API_KEY}
WORKER_POOL_SIZE=${WORKER_POOL_SIZE:-5}
LOG_LEVEL=${LOG_LEVEL:-info}
EOF

    # Deploy
    docker-compose -f deployments/docker-compose.yml up -d

    echo -e "${GREEN}Deployment complete!${NC}"
    echo "HTTP API: http://localhost:8080"
    echo "gRPC API: localhost:9090"
    echo ""
    echo "Check logs: docker-compose -f deployments/docker-compose.yml logs -f"
}

deploy_k8s() {
    echo -e "${GREEN}Deploying to Kubernetes${NC}"

    check_requirements

    # Create namespace
    kubectl create namespace ${NAMESPACE} --dry-run=client -o yaml | kubectl apply -f -

    # Create secret
    kubectl create secret generic dago-secrets \
        --namespace ${NAMESPACE} \
        --from-literal=llm-api-key=${LLM_API_KEY} \
        --dry-run=client -o yaml | kubectl apply -f -

    # Apply manifests (if any exist)
    if [ -d "deployments/k8s" ]; then
        kubectl apply -f deployments/k8s/ --namespace ${NAMESPACE}
    else
        echo -e "${YELLOW}No K8s manifests found. Use Helm deployment instead.${NC}"
        exit 1
    fi

    echo -e "${GREEN}Deployment complete!${NC}"
    echo "Check status: kubectl get pods -n ${NAMESPACE}"
}

deploy_helm() {
    echo -e "${GREEN}Deploying with Helm${NC}"

    check_requirements

    # Create namespace
    kubectl create namespace ${NAMESPACE} --dry-run=client -o yaml | kubectl apply -f -

    # Install/upgrade with Helm
    helm upgrade --install dago deployments/helm/dago \
        --namespace ${NAMESPACE} \
        --set llm.apiKey=${LLM_API_KEY} \
        --set workers.poolSize=${WORKER_POOL_SIZE:-5} \
        --set logLevel=${LOG_LEVEL:-info}

    echo -e "${GREEN}Deployment complete!${NC}"
    echo "Check status: kubectl get pods -n ${NAMESPACE}"
    echo "Get service: kubectl get svc -n ${NAMESPACE}"
}

# Main
case ${DEPLOYMENT_TYPE} in
    docker-compose)
        deploy_docker_compose
        ;;
    k8s)
        deploy_k8s
        ;;
    helm)
        deploy_helm
        ;;
    *)
        echo -e "${RED}Unknown deployment type: ${DEPLOYMENT_TYPE}${NC}"
        usage
        ;;
esac
