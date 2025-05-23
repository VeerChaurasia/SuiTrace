
# SuiTrace

**SuiTrace** is a command-line toolset for the **Sui blockchain** designed to help developers, auditors, and analysts retrieve and analyze historical on-chain data. With SuiTrace, you can backfill events, trace the full lifecycle of any Sui object, and fetch detailed data between checkpoints — all via easy-to-use CLI commands.

---

## Features

- **Event Backfilling**  
  Fetch and export historical events related to any object ID with filtering by checkpoint ranges and event limits. Output format: CSV.

- **Object History Tracing**  
  Retrieve a detailed, chronological history of any Sui object, including state changes, transfers, and transactions. Supports verbose and debug output. Output format: JSON.

- **Checkpoint Range Fetching**  
  Extract all events or on-chain activity between two specified checkpoints for scoped analysis.

---

## Getting Started

### Prerequisites

- Go 1.18+ installed  
- Access to Sui blockchain RPC endpoint (default or configured)

### Installation

Clone the repo:

```bash
git clone https://github.com/VeerChaurasia/SuiTrace.git
cd suitrace
```

---

## Usage

### 1. Event Backfilling

Fetch the last 100 events and save to a CSV file:

```bash
go run event_backfilling.go --limit=100 --filename=events.csv
```

---

### 2. Object History Tracing

Trace the full history of a specific object with verbose and debug output, and save to JSON:

```bash
go run object_history.go -object=0x2c8d603bc51326b8c13cef9dd07031a408a48dddb541963357661df5d3204809 -verbose -debug -output=history.json
```

---

### 3. Checkpoint Range Fetching

Fetch all events between checkpoint 1000 and 1010, output as JSON:

```bash
go run block_ranger.go -range=1000-1010 -output=join.json -format=json
```

---

## Use Cases

- Debugging smart contracts and dApps on Sui  
- Auditing asset and object histories (tokens, NFTs)  
- Analytics and data visualization of blockchain activity  
- Monitoring on-chain behavior for development and security

---

## Future Plans
- Develop a web UI/dashboard for visualizing object histories and checkpoint data  
- Package as an installable binary for easier distribution  
- Support more flexible filters and export formats (e.g., Parquet, JSONL)

---

**SuiTrace** — Trace. Backfill. Understand Sui.
