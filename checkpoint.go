package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	rpcURL = "https://rpc.mainnet.sui.io" // Sui mainnet RPC
)

type CheckpointData struct {
	Digest           string
	SequenceNumber   int64
	TimestampMs      int64
	ValidatorSignature string
	TransactionDigests []string
	NetworkTotalTransactions int64
	EventRoot        string
}

// Function to fetch checkpoints within a range
func FetchCheckpointRange(startCheckpoint, endCheckpoint int, maxBatchSize int) ([]CheckpointData, error) {
	allCheckpoints := []CheckpointData{}
	totalFetched := 0
	maxRetries := 3
	retryCount := 0
	
	// If no end checkpoint is specified, get the latest checkpoint first
	if endCheckpoint <= 0 {
		latestCheckpoint, err := FetchLatestCheckpoint()
		if err != nil {
			return nil, fmt.Errorf("failed to fetch latest checkpoint: %v", err)
		}
		endCheckpoint = int(latestCheckpoint.SequenceNumber)
		fmt.Printf("Latest checkpoint is %d\n", endCheckpoint)
	}
	
	// Validate range
	if startCheckpoint < 0 {
		return nil, fmt.Errorf("start checkpoint must be >= 0")
	}
	if startCheckpoint > endCheckpoint {
		return nil, fmt.Errorf("start checkpoint must be <= end checkpoint")
	}
	
	fmt.Printf("Fetching checkpoints from %d to %d\n", startCheckpoint, endCheckpoint)
	
	// Process in batches
	for currentStart := startCheckpoint; currentStart <= endCheckpoint; currentStart += maxBatchSize {
		currentEnd := currentStart + maxBatchSize - 1
		if currentEnd > endCheckpoint {
			currentEnd = endCheckpoint
		}
		
		fmt.Printf("Fetching batch from %d to %d...\n", currentStart, currentEnd)
		
		checkpoints, err := FetchCheckpointBatch(currentStart, currentEnd)
		if err != nil {
			retryCount++
			
			if retryCount > maxRetries {
				return nil, fmt.Errorf("failed to fetch checkpoints after %d retries: %v", maxRetries, err)
			}
			
			fmt.Printf("Error fetching checkpoints: %v\nRetry attempt %d of %d\n", err, retryCount, maxRetries)
			currentStart -= maxBatchSize // Retry this batch
			time.Sleep(2 * time.Second)  // Wait before retry
			continue
		}
		
		retryCount = 0
		allCheckpoints = append(allCheckpoints, checkpoints...)
		totalFetched += len(checkpoints)
		fmt.Printf("Fetched %d checkpoints so far...\n", totalFetched)
		
		// Don't overwhelm the API
		if currentStart+maxBatchSize <= endCheckpoint {
			time.Sleep(200 * time.Millisecond)
		}
	}
	
	return allCheckpoints, nil
}

// Fetch latest checkpoint to determine the current chain height
func FetchLatestCheckpoint() (*CheckpointData, error) {
	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "sui_getLatestCheckpointSequenceNumber",
		"params":  []interface{}{},
	}
	
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %v", err)
	}
	
	resp, err := http.Post(rpcURL, "application/json", bytes.NewReader(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()
	
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}
	
	var result struct {
		Result string                 `json:"result"`
		Error  map[string]interface{} `json:"error"`
	}
	
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}
	
	// Check for API errors
	if result.Error != nil {
		return nil, fmt.Errorf("API error: %v", result.Error)
	}
	
	// Convert sequence number to int
	sequenceNumber, err := strconv.ParseInt(result.Result, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse sequence number: %v", err)
	}
	
	// Now get the actual checkpoint data
	checkpoint, err := FetchCheckpoint(sequenceNumber)
	if err != nil {
		return nil, err
	}
	
	return checkpoint, nil
}

// Fetch a batch of checkpoints
func FetchCheckpointBatch(start, end int) ([]CheckpointData, error) {
	checkpoints := []CheckpointData{}
	
	for seq := start; seq <= end; seq++ {
		checkpoint, err := FetchCheckpoint(int64(seq))
		if err != nil {
			return checkpoints, err
		}
		checkpoints = append(checkpoints, *checkpoint)
	}
	
	return checkpoints, nil
}

// Fetch a single checkpoint by sequence number
func FetchCheckpoint(sequenceNumber int64) (*CheckpointData, error) {
	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "sui_getCheckpoint",
		"params":  []interface{}{strconv.FormatInt(sequenceNumber, 10)},
	}
	
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %v", err)
	}
	
	resp, err := http.Post(rpcURL, "application/json", bytes.NewReader(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()
	
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}
	
	var result struct {
		Result map[string]interface{} `json:"result"`
		Error  map[string]interface{} `json:"error"`
	}
	
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}
	
	// Check for API errors
	if result.Error != nil {
		return nil, fmt.Errorf("API error: %v", result.Error)
	}
	
	// Extract checkpoint data
	checkpoint := &CheckpointData{}
	
	// Extract basic fields
	if digest, ok := result.Result["digest"].(string); ok {
		checkpoint.Digest = digest
	}
	
	if seqStr, ok := result.Result["sequenceNumber"].(string); ok {
		seq, err := strconv.ParseInt(seqStr, 10, 64)
		if err == nil {
			checkpoint.SequenceNumber = seq
		}
	}
	
	if timestampStr, ok := result.Result["timestampMs"].(string); ok {
		timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
		if err == nil {
			checkpoint.TimestampMs = timestamp
		}
	}
	
	if networkTotalTransactionsStr, ok := result.Result["networkTotalTransactions"].(string); ok {
		networkTotal, err := strconv.ParseInt(networkTotalTransactionsStr, 10, 64)
		if err == nil {
			checkpoint.NetworkTotalTransactions = networkTotal
		}
	}
	
	if validatorSignature, ok := result.Result["validatorSignature"].(string); ok {
		checkpoint.ValidatorSignature = validatorSignature
	}
	
	if eventRoot, ok := result.Result["eventRoot"].(string); ok {
		checkpoint.EventRoot = eventRoot
	}
	
	// Extract transaction digests
	if transactions, ok := result.Result["transactions"].([]interface{}); ok {
		for _, tx := range transactions {
			if txStr, ok := tx.(string); ok {
				checkpoint.TransactionDigests = append(checkpoint.TransactionDigests, txStr)
			}
		}
	}
	
	return checkpoint, nil
}

// Save checkpoints to CSV
func SaveCheckpointsToCSV(checkpoints []CheckpointData, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create CSV file: %v", err)
	}
	defer file.Close()
	
	writer := csv.NewWriter(file)
	defer writer.Flush()
	
	// Write header
	headers := []string{
		"Digest", 
		"SequenceNumber", 
		"TimestampMs", 
		"TransactionCount", 
		"NetworkTotalTransactions",
		"EventRoot",
	}
	
	if err := writer.Write(headers); err != nil {
		return fmt.Errorf("failed to write CSV header: %v", err)
	}
	
	// Write data
	for _, checkpoint := range checkpoints {
		record := []string{
			checkpoint.Digest,
			strconv.FormatInt(checkpoint.SequenceNumber, 10),
			strconv.FormatInt(checkpoint.TimestampMs, 10),
			strconv.Itoa(len(checkpoint.TransactionDigests)),
			strconv.FormatInt(checkpoint.NetworkTotalTransactions, 10),
			checkpoint.EventRoot,
		}
		
		if err := writer.Write(record); err != nil {
			return fmt.Errorf("failed to write record to CSV: %v", err)
		}
	}
	
	return nil
}

// Save detailed checkpoint data to JSON
func SaveCheckpointsToJSON(checkpoints []CheckpointData, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create JSON file: %v", err)
	}
	defer file.Close()
	
	data, err := json.MarshalIndent(checkpoints, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal checkpoint data: %v", err)
	}
	
	_, err = file.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write JSON data: %v", err)
	}
	
	return nil
}

func ParseCheckpointRange(rangeStr string) (int, int, error) {
	if rangeStr == "" {
		return 0, 0, fmt.Errorf("checkpoint range is required")
	}
	
	parts := strings.Split(rangeStr, "-")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid range format, expected 'start-end'")
	}
	
	start, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid start checkpoint: %v", err)
	}
	
	end, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid end checkpoint: %v", err)
	}
	
	return start, end, nil
}

func main() {
	// CLI flags
	checkpointRange := flag.String("range", "", "Checkpoint range (e.g., 1000-2000), use '0-0' for latest only")
	startCheckpoint := flag.Int("start", -1, "Starting checkpoint number")
	endCheckpoint := flag.Int("end", -1, "Ending checkpoint number (0 for latest)")
	batchSize := flag.Int("batch", 10, "Number of checkpoints per batch")
	outputFile := flag.String("output", "checkpoints.csv", "Output filename")
	outputFormat := flag.String("format", "csv", "Output format (csv or json)")
	flag.Parse()
	
	var start, end int
	var err error
	
	// Parse parameters
	if *checkpointRange != "" {
		start, end, err = ParseCheckpointRange(*checkpointRange)
		if err != nil {
			log.Fatalf("Error parsing checkpoint range: %v", err)
		}
	} else {
		start = *startCheckpoint
		end = *endCheckpoint
	}
	
	if start < 0 {
		log.Fatalf("Starting checkpoint must be specified")
	}
	
	startTime := time.Now()
	fmt.Println("Starting checkpoint fetching...")
	
	// Fetch checkpoints
	checkpoints, err :=FetchCheckpointRange(start, end, *batchSize)
	if err != nil {
		log.Fatalf("Failed to fetch checkpoints: %v", err)
	}
	
	elapsedTime := time.Since(startTime)
	
	if len(checkpoints) == 0 {
		fmt.Println("No checkpoints fetched!")
		return
	}
	
	fmt.Printf("Fetched a total of %d checkpoints in %s\n", len(checkpoints), elapsedTime)
	fmt.Printf("Saving checkpoints to %s file...\n", *outputFormat)
	
	// Save to output file
	if *outputFormat == "csv" {
		err = SaveCheckpointsToCSV(checkpoints, *outputFile)
	} else if *outputFormat == "json" {
		err = SaveCheckpointsToJSON(checkpoints, *outputFile)
	} else {
		log.Fatalf("Unsupported output format: %s", *outputFormat)
	}
	
	if err != nil {
		log.Fatalf("Failed to save checkpoints: %v", err)
	}
	
	fmt.Printf("Done! %d checkpoints saved to %s 🎉\n", len(checkpoints), *outputFile)
}