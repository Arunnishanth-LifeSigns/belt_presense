# Belt Presence Monitoring System

This project provides a robust system for monitoring the presence of a belt, designed for safety and operational contexts. It leverages Go for real-time data processing, with Kafka and MQTT for messaging, and SQLite for state management.

## Architecture

The application's architecture is designed for scalability and clear separation of concerns, with the core logic residing in the `internal` directory:

*   **`internal/handler/kafka_handler.go`:** This handler is responsible for consuming the main data stream of belt sensor data from a Kafka topic. It decodes the incoming messages and processes them for real-time monitoring.
*   **`internal/handler/mqtt_handler.go`:** This handler manages control signals for the application. It subscribes to an MQTT topic to listen for `start` and `stop` commands from an upstream service, allowing for dynamic control of the monitoring process.
*   **`internal/database/sqlite.go`:** This package provides all the functions for interacting with the SQLite database. It is used for state management, storing the application's operational state to ensure data integrity and to enable graceful restarts.
*   **`internal/models/models.go`:** This file defines the data structures (structs) for the application. It includes models for decoding incoming Kafka and MQTT messages, as well as for structuring the data for any outgoing API payloads.

## Project Structure

The project is organized as follows:

*   `cmd/main.go`: The application's main entry point, responsible for initializing the configuration, database, and the message handlers.
*   `internal/`: Contains the core business logic, separated into the following packages:
    *   `config/`: Manages application configuration.
    *   `database/`: Handles all database interactions.
    *   `handler/`: Contains the logic for processing messages from Kafka and MQTT.
    *   `models/`: Defines the data structures for the application.
*   `processed_data/`: Contains sample JSON files that can be used for reference or testing. This data is not directly used by the main application.

The following files are generated locally during development and should not be committed to the repository:

*   `go.mod`, `go.sum`: Manage Go module dependencies and are created by `go mod tidy`.
*   `belt_presense.db`: The SQLite database file.
*   Any Python virtual environment directories.

## Getting Started

### Prerequisites

*   Go
*   Kafka and MQTT message brokers
*   SQLite

### Installation

1.  **Clone the repository:**
    ```bash
    git clone <repository-url>
    cd <repository-directory>
    ```
2.  **Install Go dependencies:** This command downloads the necessary modules and creates the `go.mod` and `go.sum` files.
    ```bash
    go mod tidy
    ```

### Running the Application

1.  **Configure the environment:** Create a `.env` file in the root directory using `.env.prod` as a template.

2.  **Run the application:** The following command starts the application and includes the necessary build flags for CGO.
    ```bash
    go run -tags=CGO_ENABLED_1 cmd/main.go
    ```

## Optional: Local Testing with `belt_app_streaming.py`

The `belt_app_streaming.py` script is provided for local testing and development. **It is not part of the core application and does not provide real-time monitoring capabilities.** It simulates sensor data and sends it to the message brokers.

### Test Environment Setup

1.  **Create a Python virtual environment:**
    ```bash
    python -m venv myenv
    ```
2.  **Activate the environment:**
    *   Windows: `.\myenv\Scripts\activate`
    *   macOS/Linux: `source myenv/bin/activate`
3.  **Install dependencies:**
    ```bash
    pip install paho-mqtt kafka-python
    ```

### Running the Test Script

Ensure your Kafka and MQTT brokers are running and the `.env` file is configured for your local setup.

```bash
python belt_app_streaming.py
```

## Deployment

The application can be deployed manually or via an automated script.

### Manual Deployment

1.  **Build the application:** Compile the application for your target architecture (e.g., `GOOS=linux GOARCH=amd64 go build -o belt_presense_app cmd/main.go`).
2.  **Prepare deployment files:** Gather the compiled binary, the production `.env` file, and an installation script.
3.  **Transfer and install:** Copy the files to the target server and execute the installation script.

### Automated Deployment with `deploy.sh`

The `deploy.sh` script automates the deployment process, which is especially useful for deploying to servers behind a bastion or jump host. The script handles building the binary, packaging the files, transferring them securely, and running the installation on the remote server.

#### Configuration

Before running the script, you must configure the following variables in `deploy.sh`:

*   `BASTION_HOST`: The address of the bastion host.
*   `TARGET_HOST`: The address of the target application server.
*   `LOCAL_PEM_KEY`: The path to your local PEM key for bastion host access.

This script significantly simplifies the deployment workflow, making it more efficient and less error-prone.