package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	// "flag"
	"fmt"
	"io/ioutil"
	// "log"
	"net/http"
	"os"
	// "time"
)

// const (
// 	rpcURL = "https://rpc.mainnet.sui.io" // Sui mainnet RPC
// )

func FetchEvents(cursor interface{}) ([]map[string]interface{}, interface{}, error) {
	// Using the "All" filter with an empty array as specified in the error message
	filter := map[string]interface{}{
		"All": []interface{}{},
	}
	
	params := []interface{}{
		filter,
	}
	
	// Add cursor if it exists
	params = append(params, cursor)
	
	// Add limit and ascending (true = oldest first, false = newest first)
	params = append(params, 50, true)
	
	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "suix_queryEvents", // Updated method name
		"params":  params,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal payload: %v", err)
	}

	// Debug request
	fmt.Println("Sending request:", string(payloadBytes))

	resp, err := http.Post(rpcURL, "application/json", bytes.NewReader(payloadBytes))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read response: %v", err)
	}

	// Debug response status
	fmt.Println("Response status:", resp.Status)
	
	// Only print first 200 chars of response to avoid flooding console
	responsePreview := string(body)
	if len(responsePreview) > 200 {
		responsePreview = responsePreview[:200] + "..."
	}
	fmt.Println("Response preview:", responsePreview)

	var result struct {
		Result struct {
			Data       []map[string]interface{} `json:"data"`
			NextCursor interface{}              `json:"nextCursor"`
		} `json:"result"`
		Error map[string]interface{} `json:"error"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	// Check for API errors
	if result.Error != nil {
		return nil, nil, fmt.Errorf("API error: %v", result.Error)
	}

	return result.Result.Data, result.Result.NextCursor, nil
}

func SaveEventsToCSV(events []map[string]interface{}, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create CSV file: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Dynamically determine headers based on first event
	var headers []string
	if len(events) > 0 {
		// Get all fields from first event
		for key := range events[0] {
			headers = append(headers, key)
		}
	} else {
		// Fallback headers if no events
		headers = []string{"EventID", "PackageID", "TransactionDigest", "ParsedJson"}
	}

	if err := writer.Write(headers); err != nil {
		return fmt.Errorf("failed to write CSV header: %v", err)
	}

	for _, event := range events {
		var record []string
		for _, header := range headers {
			value := ""
			if val, ok := event[header]; ok && val != nil {
				// For complex objects, convert to JSON string
				if IsComplexType(val) {
					jsonBytes, err := json.Marshal(val)
					if err == nil {
						value = string(jsonBytes)
					} else {
						value = fmt.Sprintf("%v", val)
					}
				} else {
					value = fmt.Sprintf("%v", val)
				}
			}
			record = append(record, value)
		}
		
		if err := writer.Write(record); err != nil {
			return fmt.Errorf("failed to write record to CSV: %v", err)
		}
	}
	
	return nil
}

// Helper function to detect complex types (maps/slices) that need JSON serialization
func IsComplexType(v interface{}) bool {
	switch v.(type) {
	case map[string]interface{}, []interface{}:
		return true
	default:
		return false
	}
}

// func main() {
// 	// CLI flags
// 	limit := flag.Int("limit", 200, "Number of events to fetch (max)")
// 	filename := flag.String("filename", "events.csv", "Output CSV filename")
// 	flag.Parse()

// 	fmt.Println("Starting event backfill...")

// 	allEvents := []map[string]interface{}{}
// 	var cursor interface{}
// 	totalFetched := 0
// 	maxRetries := 3
// 	retryCount := 0

// 	startTime := time.Now()

// 	for {
// 		events, nextCursor, err := FetchEvents(cursor)
// 		if err != nil {
// 			fmt.Printf("Error fetching events: %v\n", err)
// 			retryCount++

// 			if retryCount > maxRetries {
// 				log.Fatalf("Failed to fetch events after %d retries: %v", maxRetries, err)
// 			}

// 			fmt.Printf("Retry attempt %d of %d\n", retryCount, maxRetries)
// 			continue
// 		}

// 		retryCount = 0

// 		if len(events) == 0 {
// 			fmt.Println("No more events found!")
// 			break
// 		}

// 		allEvents = append(allEvents, events...)
// 		totalFetched += len(events)
// 		fmt.Printf("Fetched %d events so far...\n", totalFetched)

// 		cursor = nextCursor
// 		if cursor == nil {
// 			fmt.Println("No pagination cursor returned - we've reached the end")
// 			break
// 		}

// 		// Stop if user-defined limit reached
// 		if totalFetched >= *limit {
// 			fmt.Printf("Reached user-defined limit of %d events\n", *limit)
// 			break
// 		}
// 	}

// 	elapsedTime := time.Since(startTime)

// 	if len(allEvents) == 0 {
// 		fmt.Println("No events fetched!")
// 		return
// 	}

// 	fmt.Printf("Fetched a total of %d events in %s\n", len(allEvents), elapsedTime)
// 	fmt.Println("Saving events to CSV file...")

// 	err := SaveEventsToCSV(allEvents, *filename)
// 	if err != nil {
// 		log.Fatalf("Failed to save events to CSV: %v", err)
// 	}

// 	fmt.Printf("Done! %d events saved to %s 🎉\n", len(allEvents), *filename)
// }
