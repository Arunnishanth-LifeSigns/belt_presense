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

*   `belt_app_streaming.py`: A Python script responsible for streaming belt presence data. This is a critical component for real-time data dissemination.
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

## Deployment

The `deploy.sh` script is provided to automate the deployment of the application. This script will likely perform the following actions:

*   Build the Go application.
*   Copy the necessary files to the deployment server.
*   Restart the application services.

To use the script, you may need to make it executable:
```bash
chmod +x deploy.sh
```

Then, you can run it with:
```bash
./deploy.sh
```

*(Note: The deployment script may need to be configured with the specific details of your deployment environment.)*

## Contributing

Contributions to this project are welcome. Please fork the repository and submit a pull request with your changes.
