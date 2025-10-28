# Belt Presence Monitoring System

This project is a comprehensive system designed to monitor the presence of a belt, likely in a safety or operational context. It processes sensor data, provides real-time streaming, and stores presence information in a database. The system is built with a combination of Go and Python, utilizing technologies like Kafka and MQTT for messaging.

## Key Features

*   **Real-time Data Processing:** The system is capable of processing sensor data in real-time to determine belt presence.
*   **Data Streaming:** It uses a Python script (`belt_app_streaming.py`) for streaming data, enabling real-time monitoring and integration with other systems.
*   **Database Storage:** Presence information is stored in a SQLite database (`belt_presense.db`), allowing for historical data analysis and reporting.
*   **Messaging Integration:** The Go application is designed to work with Kafka and MQTT, providing flexibility in how data is received and processed.
*   **Deployment Ready:** The project includes a deployment script (`deploy.sh`) to facilitate easy deployment to a target environment.

## Project Structure

The project is organized into several key directories and files:

*   `belt_app_streaming.py`: A Python script for testing purposes that simulates data flow. It is optional to use.
*   `deploy.sh`: A shell script that automates the deployment process, making it easier to set up the system in a new environment.
*   `cmd/main.go`: The main entry point for the Go application. It initializes the configuration, database, and message handlers.
*   `internal/`: This directory contains the core logic of the Go application, separated into packages for configuration, database interaction, message handling, and data models.
*   `processed_data/`: This directory contains JSON files with processed data from the sensors, organized by patient ID.
*   `go.mod` and `go.sum`: These files manage the Go module dependencies for the project.
*   `docker-compose.yml`: This file can be used to define and run multi-container Docker applications.

## Getting Started

### Prerequisites

*   Go
*   Python
*   A message broker (Kafka or MQTT)
*   SQLite

### Installation

1.  **Clone the repository:**
    ```bash
    git clone https://github.com/Arunnishanth-LifeSigns/belt_presense.git
    cd belt_presense
    ```

2.  **Install Go dependencies:**
    ```bash
    go mod tidy
    ```

3.  **Install Python dependencies:**
    ```bash
    pip install -r requirements.txt
    ```
    *(Note: A `requirements.txt` file may need to be created if it doesn't exist.)*

### Running the Application

1.  **Start the Go application:**
    ```bash
    go run cmd/main.go
    ```

2.  **Run the Python streaming script:**
    ```bash
    python belt_app_streaming.py
    ```

### Local Testing

The `belt_app_streaming.py` script is provided for local testing. It simulates the data flow of belt sensors and sends the data to the Kafka and MQTT brokers.

To use the script for local testing, you need to have the following applications running in the background:
*   Kafka
*   MQTT

The `.env` file should be configured for your local environment. The default configuration is as follows:

```
# Local Development .env

# Kafka Configuration
KAFKA_BROKERS=localhost:9092
VITALS_TOPIC=patient-vitals-data-topic
CONSUMER_GROUP=belt_presense

# Presense API Configuration (Using Test/Staging)
PRESENSE_API_ENDPOINT=https://staging-vitals.presense.icu/data
PRESENSE_API_KEY=814xqfGJWSVfKgfFOvQz24MLAuRsuDA3
DATA_SOURCE=Arun-Local
USE_TEST_URL=true

# MQTT Configuration (Local)
MQTT_BROKER_URL=tcp://localhost:1883
MQTT_CLIENT_ID=MqttCallService_local
MQTT_USERNAME=
MQTT_PASSWORD=

# Application Configuration
DB_PATH=../belt_presense.db
WRITE_TO_FILE=true
LOG_TO_CONSOLE=true
```

## Deployment

The `deploy.sh` script is provided to automate the deployment of the application. This script performs the following actions:

1.  **Builds the Go application** for a Linux ARM64 environment.
2.  **Creates a deployment package** (`package.tar.gz`) containing the binary, `install.sh`, and the production configuration file (`.env.prod` renamed to `.env`).
3.  **Transfers the package** to the target server via a bastion host.
4.  **Executes the `install.sh` script** on the target server to install the application.
5.  **Cleans up** the temporary files on the target and local machines.

### Environment-Specific Changes

Before running the `deploy.sh` script, you need to configure the following variables in the script:

*   `BASTION_HOST`: The IP address of the bastion host.
*   `TARGET_HOST`: The IP address of the target server.
*   `LOCAL_PEM_KEY`: The path to your local PEM key for accessing the bastion host.

### Deployment Process

The `deploy.sh` script uses `.env.prod` for the production deployment. This file is copied into the deployment package and renamed to `.env`. The `.env` file is used for local testing only and is not included in the deployment package.

The files are loaded onto the server in the following steps:

1.  The `package.tar.gz` file is copied from your local machine to the `/tmp` directory on the bastion host using `scp`.
2.  The `package.tar.gz` file is then moved from the bastion host to the home directory (`~/`) on the target server using `scp`.
3.  The `install.sh` script is executed on the target server, which unpacks the `package.tar.gz` file into a temporary directory (`~/install_package`), runs the installation, and then cleans up the temporary files.

## Contributing

Contributions to this project are welcome. Please fork the repository and submit a pull request with your changes.
