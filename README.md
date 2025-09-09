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

Follow these instructions to get Logvault up and running on your local machine.

### Prerequisites

You need to have the following software installed:

- **Go**: Version 1.18 or higher.
- **Redis**: A running Redis instance. You can install it locally or use Docker.
  ```sh
  # Using Docker (recommended)
  docker run -d --name logvault-redis -p 6379:6379 redis

  # Using apt (Debian/Ubuntu)
  sudo apt-get update && sudo apt-get install redis-server
  ```

### Installation

1.  **Clone the repository:**
    ```sh
    git clone https://github.com/sfreet/logvault.git
    cd logvault
    ```

2.  **Configure the application:**
    Copy or rename `config.yaml.example` to `config.yaml` if you need to make changes. The default settings should work for a local setup.

3.  **Initialize Go Modules and Build the application:**
    Ensure you are in the project root directory where `go.mod` is located.
    ```sh
    go mod tidy
    make build
    ```
    This command compiles the source code and creates an executable file named `logvault`.

4.  **Run the application:**
    ```sh
    ./logvault
    # Or with sudo if you need to listen on a privileged port like 514
    # sudo ./logvault
    ```
    You should see output indicating that the Redis connection was successful and the servers are running.

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

### REST API for Redis Data

Logvault exposes a REST API endpoint to retrieve all data stored in Redis. This API is secured using a Bearer token.

**Endpoint:** `GET /api/data`

**Authentication:**
This API requires a Bearer token in the `Authorization` header.

1.  **Configure your Bearer Token:**
    Edit your `config.yaml` file and set a strong, unique `bearer_token` under the `api` section:
    ```yaml
    # API settings
    api:
      bearer_token: "your_chosen_bearer_token" # Choose a strong, unique token
    ```

2.  **Call the API:**
    Once Logvault is running, you can call the API using `curl` (replace `your_chosen_bearer_token` with your actual token):
    ```bash
    curl -H "Authorization: Bearer your_chosen_bearer_token" http://localhost:8080/api/data
    ```
    The API will return a JSON object containing all key-value pairs currently stored in Redis.

### External API Integration

Logvault can be configured to call an external REST API when a syslog message with a specific tag is processed. This allows for event-driven integrations with other systems.

**Configuration:**
Edit your `config.yaml` file and configure the `external_api` section:

```yaml
# External API integration settings
external_api:
  enabled: false # Set to true to enable this feature
  url: "http://example.com/api/event" # The URL of your external API endpoint
  method: "POST" # HTTP method (e.g., GET, POST, PUT, DELETE)
  bearer_token: "" # Optional: Bearer token for external API authentication
  trigger_tag: "ALARM" # The syslog app_name (tag) that will trigger the API call
```

**How it works:**
When Logvault processes a syslog message whose `app_name` matches the `trigger_tag` configured in `config.yaml`, it will make an HTTP request to the specified `url` with the configured `method`. The payload sent to the external API will be a JSON object containing the `key`, `message`, and `status` (ALARM/CLEAR) of the processed syslog event.

**Example Payload:**
```json
{
  "key": "alarm:192.168.1.100",
  "message": "192.168.1.100 System is overheating",
  "status": "ALARM"
}
```

## Configuration

You can configure the application by editing the `config.yaml` file.

```yaml
# Syslog server settings
syslog:
  host: "0.0.0.0"
  port: 514
  protocol: "udp"

# Redis settings
redis:
  address: "localhost:6379"
  password: ""
  db: 0

# Web UI settings
web:
  port: 8080
  secret: "your_secret_password" # Add a secret for web UI login
```

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
oject is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
