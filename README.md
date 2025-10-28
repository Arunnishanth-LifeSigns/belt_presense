# Belt Presence Monitoring System

This project is a comprehensive system designed to monitor the presence of a belt, likely in a safety or operational context. It processes sensor data in real-time and stores presence information in a database. The system is built using Go and utilizes technologies like Kafka and MQTT for messaging.

## Architecture

The application is designed with a clear separation of concerns for robust and scalable data processing:

*   **Kafka (`kafka_handler.go`):** The core data stream of belt sensor data is processed through Kafka. The application consumes data from a specified Kafka topic.
*   **MQTT (`mqtt_handler.go`):** MQTT is used for control signals. It listens for `start` and `stop` commands from an upstream sync service, which allows for dynamic control of the monitoring process.
*   **SQLite (`sqlite.go`):** The application uses a SQLite database for state management. This allows the application to keep track of its operational state and initiate restarts appropriately, ensuring data integrity and continuity.

## Project Structure

The project is organized into several key directories and files:

*   `cmd/main.go`: The main entry point for the Go application. It initializes the configuration, database, and message handlers.
*   `internal/`: This directory contains the core logic of the Go application, separated into packages for configuration (`config`), database interaction (`database`), message handling (`handler`), and data models (`models`).
*   `docker-compose.yml`: This file can be used to define and run multi-container Docker applications, particularly for setting up the required services like Kafka and MQTT brokers.
*   `belt_presense.db`: The SQLite database file used for state management.

The following files are generated locally and should not be committed to the repository:
*   `go.mod`, `go.sum`: These files manage the Go module dependencies and are created by running `go mod tidy`.
*   `venv_beltStream/`: This is a Python virtual environment directory created for the optional testing script.

## Getting Started

### Prerequisites

*   Go
*   A message broker (Kafka or MQTT)
*   SQLite

### Installation

1.  **Set up the project:** Ensure you have the project files on your local machine.

2.  **Install Go dependencies:** This command will download the necessary Go modules and create the `go.mod` and `go.sum` files.
    ```bash
    go mod tidy
    ```

### Running the Application

1.  **Configure your environment:** Create a `.env` file for your local environment. You can use the `.env.prod` file as a template.

2.  **Start the Go application:**
    ```bash
    go run cmd/main.go
    ```

## Optional: Local Testing with `belt_app_streaming.py`

For testing purposes, a Python script `belt_app_streaming.py` is included. 
**This script is only for testing and local development. It is not part of the core application and does not enable real-time monitoring in a production environment.**

When this script is run, it generates the `processed_data/` directory, which contains JSON files simulating the sensor data output. This directory is a result of the testing script and is not part of the main application's data flow.

### Setting up the Python Test Environment

1.  **Create a Python virtual environment:**
    ```bash
    python -m venv venv_beltStream
    ```

2.  **Activate the virtual environment:**
    *   **Windows:**
        ```bash
        .\venv_beltStream\Scripts\activate
        ```
    *   **macOS/Linux:**
        ```bash
        source venv_beltStream/bin/activate
        ```

3.  **Install Python dependencies:**
    ```bash
    pip install -r requirements.txt 
    ```
    *(Note: You may need to create a `requirements.txt` file with the necessary libraries, such as `paho-mqtt` and `kafka-python`)*

### Running the Test Script

To use the script for local testing, you need to have Kafka and MQTT brokers running in the background. The `.env` file should be configured for your local environment.

```bash
python belt_app_streaming.py
```

## Deployment

You can deploy the application manually or use the provided `deploy.sh` script for an automated process.

### Manual Deployment

1.  **Build the Go application:** Create a binary for your target server's architecture. For example, for a Linux server:
    ```bash
    GOOS=linux GOARCH=amd64 go build -o belt_presense_app cmd/main.go
    ```

2.  **Prepare the deployment files:** Create a directory with the following files:
    *   The compiled binary (`belt_presense_app`)
    *   The production environment file (`.env.prod`), renamed to `.env`
    *   An `install.sh` script to move the files to the correct location and set up a service.

3.  **Transfer the files to the server:** Use `scp` or another method to copy the files to your target server.

4.  **Run the installation script:** SSH into the server and execute the `install.sh` script.

### Automated Deployment with `deploy.sh`

The project includes a `deploy.sh` script that automates the deployment process, which is particularly useful when deploying to a server behind a bastion (jump) host.

The script performs the following steps:
1.  **Builds the Go application** for a Linux ARM64 environment.
2.  **Creates a deployment package** (`package.tar.gz`) containing the binary, `install.sh`, and the production configuration (`.env.prod` renamed to `.env`).
3.  **Transfers the package** to the target server by first copying it to the bastion host and then to the target.
4.  **Executes the `install.sh` script** on the target server.
5.  **Cleans up** temporary files.

#### Configuration

Before running `deploy.sh`, you must update the following variables in the script to match your environment:

*   `BASTION_HOST`: The IP address or hostname of your bastion host.
*   `TARGET_HOST`: The IP address or hostname of your target application server.
*   `LOCAL_PEM_KEY`: The local path to the PEM key required to access the bastion host.

This script simplifies the deployment process by handling the multi-step file transfer and remote execution in a single command.
