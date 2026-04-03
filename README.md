# Overengineering

Theory is not enough. Every project here uses tech that is 100% overkill 
for the problem size, and that is the point.

## Projects
- [URL Shortener](./projects/url-shortener) — DynamoDB, Redis, Kafka, Load Balancer

## Stack
- Go
- Apache Kafka + Zookeeper
- DynamoDB Local
- Redis
- Prometheus + Grafana
- k6 for load testing
- Terraform for AWS deployment

## Running Locally
docker-compose up