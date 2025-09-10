# **SafeZone**
[![CI Status](https://img.shields.io/badge/CI-Passing-green?style=for-the-badge)](https://github.com/rebodutch/safezone/actions)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg?style=for-the-badge)](https://opensource.org/licenses/MIT)

A microservices-based health safety map information system, currently using simulated COVID-19 data as a Proof-of-Concept.

## **🎯 About This Repository (Safezone)**

This repository (Safezone) serves as the **application layer** of the SafeChord project, responsible for implementing the data-driven application named **SafeZone**. The primary goal of SafeZone is to simulate a complete data ecosystem—from event generation and asynchronous ingestion to final analysis and visualization.

The code in this repository is not just about making features work; it's about crafting an application blueprint that reflects a "production-grade" mindset, demonstrated through:

* **System Design & Architecture**  
* **Code Quality & Maintainability**  
* **Design for Automation & Ops**

## **✨ Architectural Highlights**

* **Async & Event-Driven Architecture**: Implements a high-throughput asynchronous data pipeline centered around Kafka, enabling horizontal scalability and high resilience.  
* **Designed for Observability**: All services produce structured JSON logs and feature a built-in Trace ID mechanism, providing a solid foundation for end-to-end tracing and centralized log aggregation.  
* **Ready for CI/CD**: The project includes a comprehensive E2E smoke test framework and a standardized Makefile build process, allowing for seamless integration with any CI/CD platform for automated validation and delivery.  
* **Caching Layer Integration**: A built-in Redis caching mechanism is included, preparing the system for high-traffic scenarios by reducing database load and query latency.  
* **Polyglot Microservices**: Combines the strengths of Python (FastAPI) and Go, demonstrating the ability to manage and orchestrate services written in different languages.

## **💻 Tech Stack**

* **Backend**: Python (FastAPI, Pydantic), Go  
* **Frontend**: Plotly Dash  
* **Data Layer**: PostgreSQL, Redis, Kafka  
* **Testing**: Pytest, Docker Compose  
* **Build**: Docker, Make

## **🚀 Quick Start (Local Development & Testing)**

This guide is for building and testing the **application code** locally. For instructions on deploying the application to Kubernetes, please refer to the safezone-deploy repository.

### **Prerequisites**

* Docker & Docker Compose  
* make

### **1\. Build All Images**

Build the Docker images for all services and tools. The default tag will be latest.

```
make build-all  
make build-tool-all
```

### **2\. Run All Tests**

Execute the full suite of unit, integration, and end-to-end smoke tests. This command will start a temporary environment, run all tests, and automatically shut it down.

```
make smoke-test
```

If all tests pass, the application code is verified.

ℹ️ **Note**: To improve the local development experience, a command for persistently running the environment (e.g., make dev-up) is planned for a future version.

## **📖 Full Documentation**

For a deep dive into the architecture, design decisions, and detailed specifications of each component, please visit the full **SafeChord Knowledge Base**.

**(Link to MkDocs site will go here)**

## **📄 License**

This project is licensed under the MIT License. See the LICENSE file for details.