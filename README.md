# ip-sync

`ip-sync` is a Go application that synchronizes your public IP address with a DNS record in Cloudflare. It periodically checks your public IP address and updates the DNS record if the IP address has changed.

## Features

- Automatically fetches the public IP address.
- Updates the DNS record in Cloudflare.
- Configurable check interval.
- Supports multiple platforms (Darwin, Linux, Windows).
- Docker support for easy deployment.

## Usage

### Local Run

To run the application locally, set the required environment variables and execute the binary:

```sh
export CLOUDFLARE_API_TOKEN=your-api-token
export CLOUDFLARE_ZONE_ID=your-zone-id
export CLOUDFLARE_RECORD_NAME=your-record-name

./ip-sync
```

### Run with Docker

```sh
docker run -e CLOUDFLARE_API_TOKEN=your-api-token -e CLOUDFLARE_ZONE_ID=your-zone-id -e CLOUDFLARE_RECORD_NAME=your-record-name exelban/ip-sync:latest
```

### Run with Docker Compose

Create a `docker-compose.yml` file with the following content:

```yaml
services:
  ip-sync:
    image: exelban/ip-sync:latest
    restart: unless-stopped
    environment:
      - CLOUDFLARE_API_TOKEN=your-api-token
      - CLOUDFLARE_ZONE_ID=your-zone-id
      - CLOUDFLARE_RECORD_NAME=your-record-name
```

Then, run the application using Docker Compose:

```sh
docker-compose up -d
```

## Installation

### Prerequisites

- Go 1.18 or later
- Docker (optional)

### Clone the repository

```sh
git clone https://github.com/exelban/ip-sync.git
cd ip-sync
```

### Build

To build the application, run:

```sh
go build -o bin/ip-sync
```

### Run

To run the application, set the required environment variables and execute the binary:

```sh
export CLOUDFLARE_API_TOKEN=your-api-token
export CLOUDFLARE_ZONE_ID=your-zone-id
export CLOUDFLARE_RECORD_NAME=your-record-name

./bin/ip-sync
```

## Configuration

The application can be configured using environment variables:

- `CLOUDFLARE_API_TOKEN`: Your Cloudflare API token. Token must have the following permissions: `Zone.Zone:Read`, `Zone.DNS:Edit`.
- `CLOUDFLARE_ZONE_ID`: The ID of your Cloudflare zone.
- `CLOUDFLARE_RECORD_NAME`: The name of the DNS record to update.
- `INTERVAL`: The interval in seconds between IP address checks (default: 5m).
- `DEBUG`: Enable debug mode (default: false).

## License
[MIT License](https://github.com/exelban/ip-sync/blob/master/LICENSE)