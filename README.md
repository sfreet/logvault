# Logvault

Logvault is a simple, tag-based syslog processor that listens for alarms, stores them in Redis, and provides a real-time web interface for viewing and managing them.

It's designed to be a lightweight solution for scenarios where you need to track state from syslog messages (e.g., a device is offline/online) and have a simple visual dashboard for the current status.

## Features

- **Syslog Listener**: Receives syslog messages (RFC5424 format) over UDP.
- **Tag-Based Processing**: Creates or deletes alarms based on a "tag" (the syslog `app_name`).
  - `ALARM`: Creates/updates an alarm.
  - `CLEAR`: Deletes an alarm.
- **Redis Backend**: Uses Redis to store the current state of active alarms.
- **Real-time Web UI**: A clean web interface that automatically refreshes to show the current list of active alarms.
- **Web UI Authentication**: Secure your web interface with a configurable secret password.
- **Logout Functionality**: Allows users to securely log out of the web UI.
- **Manual Deletion**: Allows manual deletion of alarms directly from the web UI.
- **Configuration**: Easily configurable via a `config.yaml` file.

## Getting Started

Follow these instructions to get Logvault up and running.

### Prerequisites

You need to have the following software installed:

- **Go**: Version 1.18 or higher (for building from source).
- **Docker** and **Docker Compose**: For running the application in containers.

### Installation and Running

Logvault can be run in a Docker container or built from source.

1.  **Using Docker Compose (Recommended)**

    This is the easiest way to get started. It automatically builds the Go application and runs it alongside a Redis container.

    1.  **Clone the repository:**
        ```sh
        git clone https://github.com/sfreet/logvault.git
        cd logvault
        ```

    2.  **Configure the application:**
        Rename `config.yaml.example` to `config.yaml` and edit it to suit your needs. You must set a `secret` for the web UI and a `bearer_token` for the API.
        ```sh
        cp config.yaml.example config.yaml
        # Now edit config.yaml
        ```

    3.  **Build and run with Docker Compose:**
        ```sh
        docker-compose up --build
        ```
        The application will be available at `http://localhost:8080`.

2.  **Building and Running Docker Image Manually**

    If you want to build and run the Docker image yourself:

    1.  **Clone the repository and configure as above.**

    2.  **Build the Docker image:**
        ```sh
        make docker # This runs `docker build -t logvault .`
        ```

    3.  **Run the Docker container:**
        ```sh
        docker run -d --name logvault -p 8080:8080 -p 514:514/udp --network host logvault
        # Make sure Redis is running and accessible from the container
        ```

3.  **Building from Source**

    If you prefer to build the application manually:

    1.  **Install Redis:**
        Make sure you have a running Redis instance.
        ```sh
        # Using Docker
        docker run -d --name logvault-redis -p 6379:6379 redis
        ```

    2.  **Clone the repository and configure as above.**

    3.  **Build and run the application:**
        ```sh
        go mod tidy
        make build # This compiles the Go application into a binary named 'logvault'
        ./logvault
        ```
        You may need to use `sudo` if you are listening on a privileged port like 514.

    **Note:** You can use `make help` to see all available Makefile commands.

## Usage

### Sending Syslog Alarms

You can use the standard `logger` utility to send syslog messages to Logvault.

-   **To create an alarm:**
    Use the `-t ALARM` tag. The message format should be `"<key> <message>"`. The `<key>` is typically an IP address or hostname.
    ```sh
    logger -n 127.0.0.1 -P 514 -t ALARM "192.168.1.100 System is overheating"
    ```

-   **To clear an alarm:**
    Use the `-t CLEAR` tag. The message should contain the `<key>` of the alarm to be cleared.
    ```sh
    logger -n 127.0.0.1 -P 514 -t CLEAR "192.168.1.100"
    ```

### Accessing the Web UI

Once the application is running, open your web browser and navigate to:

**http://localhost:8080**

You will be prompted to enter the secret configured in `config.yaml` to access the dashboard. The UI will display a list of all active alarms. It auto-refreshes every 5 seconds. You can also manually delete an alarm by clicking the "Delete" button or log out using the "Logout" button.

### REST API

Logvault provides two sets of REST API endpoints for accessing data.

#### Bearer Token Authenticated Endpoint

This endpoint is intended for external services and programmatic access. It is secured with a Bearer token.

-   **Endpoint:** `GET /api/data`
-   **Authentication:** `Bearer Token`
    -   Set a `bearer_token` in your `config.yaml` under the `api` section.
    -   Include it in the `Authorization` header of your request.
-   **Description:** Retrieves all raw key-value pairs currently stored in Redis, including non-alarm data.

**Example Request:**
```bash
curl -H "Authorization: Bearer your_bearer_token" http://localhost:8080/api/data
```

#### Web UI / Session Authenticated Endpoints

These endpoints are used by the web UI and are protected by the same session cookie as the web interface. They are primarily for managing alarms.

-   **Endpoint:** `GET /api/alarms`
    -   **Description:** Retrieves all active alarms. The response is a JSON object containing key-value pairs for all entries with the `alarm:` prefix.

-   **Endpoint:** `DELETE /api/alarms/{key}`
    -   **Description:** Deletes a specific alarm by its key. For example, a request to `/api/alarms/192.168.1.100` will delete the `alarm:192.168.1.100` key from Redis.

-   **Endpoint:** `DELETE /api/alarms` or `DELETE /api/alarms/`
    -   **Description:** Deletes all alarms from Redis. This is used by the "Delete All" button in the web UI.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
