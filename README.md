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

3.  **Build the application:**
    This command compiles the source code and creates an executable file named `logvault`.
    ```sh
    make build
    ```

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

The UI will display a list of all active alarms. It auto-refreshes every 5 seconds. You can also manually delete an alarm by clicking the "Delete" button.

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
```

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
oject is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
