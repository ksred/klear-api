# Klear Trading System

A trading system that simulates the complete lifecycle of securities trading, from order submission through execution, clearing, and final settlement.

## Overview

Klear is a high-performance trading infrastructure that provides:
- Order management and execution across multiple exchanges
- Real-time trade clearing and settlement processing
- Multi-lateral netting for efficient settlement
- Simulated market conditions and exchange behaviors
- Complete trade lifecycle management

## Features

- **Multi-Exchange Support**: Simulates trading across 4 different exchanges with varying characteristics:
  - Primary Exchange (low latency, high liquidity)
  - Secondary Exchange (medium latency, medium liquidity)
  - Regional Exchange (higher latency, lower liquidity)
  - Dark Pool (highest latency, lowest liquidity)

- **Complete Trade Lifecycle**:
  - Order submission and validation
  - Smart order routing
  - Trade execution
  - Clearing process
  - T+2 Settlement (removed for testing)

- **Risk Management**:
  - Pre-trade validation
  - Real-time position monitoring
  - Dynamic margin calculations
  - Exposure limits

## Getting Started

### Prerequisites

- Go 1.21 or higher
- SQLite3
- Make

### Installation

1. Clone the repository:
   git clone https://github.com/ksred/klear-api
   cd klear-api

2. Install dependencies:
   go mod download

3. Build the project:
   make build

### Running the Application

Start the server:
   make run

The API will be available at http://localhost:8080

### Running the Simulation

To run the trading simulation:
   make simulate

The simulation will:
- Generate random orders across multiple symbols
- Route orders to different exchanges
- Process executions, clearing, and settlement
- Log detailed statistics about the trading activity

## Development

### Available Make Commands

- make build - Build the application
- make clean - Remove build artifacts
- make run - Run the application
- make test - Run tests
- make test-coverage - Run tests with coverage
- make lint - Run linters
- make all - Clean, lint, test, and build
- make simulate - Run the trading simulation
- make watch - Run in development mode with auto-reload (requires fswatch)

### Project Structure

.
├── cmd/
│   ├── server/     # Main application
│   └── simulation/ # Trading simulation
├── internal/
│   ├── auth/       # Authentication
│   ├── clearing/   # Trade clearing
│   ├── database/   # Database operations
│   ├── exchange/   # Exchange simulation
│   ├── models/     # Data models
│   ├── settlement/ # Settlement processing
│   └── trading/    # Order management
├── pkg/
│   ├── common/     # Shared utilities
│   └── middleware/ # HTTP middleware
└── configs/        # Configuration files

## API Documentation

See DOCS.md for detailed API documentation.

## Testing

Run the test suite:
   make test

Generate test coverage report:
   make test-coverage

## Configuration

The application can be configured using environment variables:

- ENV - Environment (development/production)
- DEBUG - Enable debug logging (true/false)
- PORT - Server port (default: 8080)

## Contributing

1. Fork the repository
2. Create your feature branch (git checkout -b feature/amazing-feature)
3. Commit your changes (git commit -m 'Add amazing feature')
4. Push to the branch (git push origin feature/amazing-feature)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.