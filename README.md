
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

Fetch a specified number of recent events and save them to a CSV file:

```bash
go run event_backfilling.go --limit=<number_of_events> --filename=<output_filename>.csv
```
---

### 2. Object History Tracing

Trace the full history of a specific object with verbose and debug output, and save to JSON:

```bash
go run object_history.go -object=<object_id> -verbose -debug -output=<output_filename>.json
```

---

### 3. Checkpoint Range Fetching

Fetch all events or activities that occurred between two checkpoints, with customizable output format:

```bash
go run block_ranger.go -range=<start_checkpoint>-<end_checkpoint> -output=<output_filename> -format=<json|csv>
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
