package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"time"
)

const (
	// Exported constant for RPC URL
	RpcURL = "https://rpc.mainnet.sui.io" // Sui mainnet RPC
)

type ObjectState struct {
	Version    string                 `json:"version"`
	Digest     string                 `json:"digest"`
	Type       string                 `json:"type"`
	Owner      map[string]interface{} `json:"owner"`
	PreviousTx string                 `json:"previousTransaction"`
	Content    map[string]interface{} `json:"content"`
	Timestamp  int64                  `json:"timestamp"`
}

type ObjectHistory struct {
	ID         string        `json:"id"`
	States     []ObjectState `json:"states"`
	FirstSeen  int64         `json:"firstSeen"`
	LastSeen   int64         `json:"lastSeen"`
	NumChanges int           `json:"numChanges"`
	NumOwners  int           `json:"numOwners"`
}

// Debug mode flag
var debugMode bool

// Helper function to print debug info
func DebugPrint(format string, a ...interface{}) {
	if debugMode {
		fmt.Printf("[DEBUG] "+format+"\n", a...)
	}
}

// Helper function to make RPC calls
func MakeRPCCall(method string, params []interface{}) (map[string]interface{}, error) {
	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  method,
		"params":  params,
	}
	
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %v", err)
	}
	
	DebugPrint("Sending request to %s: %s", RpcURL, string(payloadBytes))
	
	resp, err := http.Post(RpcURL, "application/json", bytes.NewReader(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()
	
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}
	
	DebugPrint("Received response: %s", string(body))
	
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}
	
	// Check for API errors
	if errObj, exists := result["error"]; exists && errObj != nil {
		return nil, fmt.Errorf("API error: %v", errObj)
	}
	
	return result, nil
}

// Get all transactions for an object
func GetAllObjectTransactions(objectID string) ([]string, error) {
	result, err := MakeRPCCall("sui_queryTransactionBlocks", []interface{}{
		map[string]interface{}{
			"InputObject": objectID,
		},
		nil, // cursor
		nil, // limit
		true, // descending order
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to query transactions: %v", err)
	}
	
	var txDigests []string
	
	if resultObj, ok := result["result"].(map[string]interface{}); ok {
		if data, ok := resultObj["data"].([]interface{}); ok {
			for _, tx := range data {
				if txObj, ok := tx.(map[string]interface{}); ok {
					if digest, ok := txObj["digest"].(string); ok {
						txDigests = append(txDigests, digest)
					}
				}
			}
		}
	}
	
	DebugPrint("Found %d transactions for object %s", len(txDigests), objectID)
	return txDigests, nil
}

// Get object details from a transaction
func GetObjectDetailsFromTransaction(txDigest string, objectID string) (*ObjectState, error) {
	result, err := MakeRPCCall("sui_getTransactionBlock", []interface{}{
		txDigest,
		map[string]interface{}{
			"showEffects": true,
			"showInput": true,
			"showEvents": false,
			"showObjectChanges": true,
			"showBalanceChanges": false,
		},
	})
	
	if err != nil {
		return nil, err
	}
	
	// Extract transaction timestamp
	var timestamp int64
	if resultObj, ok := result["result"].(map[string]interface{}); ok {
		if timestampMs, ok := resultObj["timestamp_ms"].(string); ok {
			if ts, err := strconv.ParseInt(timestampMs, 10, 64); err == nil {
				timestamp = ts
			}
		}
	}
	
	// Look for object changes related to our object
	state := &ObjectState{
		PreviousTx: txDigest,
		Timestamp:  timestamp,
	}
	
	foundObject := false
	
	if resultObj, ok := result["result"].(map[string]interface{}); ok {
		if objectChanges, ok := resultObj["objectChanges"].([]interface{}); ok {
			for _, change := range objectChanges {
				if changeObj, ok := change.(map[string]interface{}); ok {
					// Check if this change is for our object
					if objID, ok := changeObj["objectId"].(string); ok && objID == objectID {
						foundObject = true
						
						// Extract object details
						if version, ok := changeObj["version"].(float64); ok {
							state.Version = fmt.Sprintf("%d", int64(version))
						}
						
						if objType, ok := changeObj["objectType"].(string); ok {
							state.Type = objType
						}
						
						if digest, ok := changeObj["digest"].(string); ok {
							state.Digest = digest
						}
						
						// Extract owner information
						if owner, ok := changeObj["owner"].(map[string]interface{}); ok {
							state.Owner = owner
						}
						
						break
					}
				}
			}
		}
	}
	
	if !foundObject {
		return nil, fmt.Errorf("object %s not found in transaction %s", objectID, txDigest)
	}
	
	return state, nil
}

// Get object's current state
func GetObjectCurrentState(objectID string) (*ObjectState, error) {
	result, err := MakeRPCCall("sui_getObject", []interface{}{
		objectID,
		map[string]interface{}{
			"showContent": true,
			"showOwner": true,
			"showType": true,
			"showPreviousTransaction": true,
		},
	})
	
	if err != nil {
		return nil, err
	}
	
	state := &ObjectState{}
	
	if resultObj, ok := result["result"].(map[string]interface{}); ok {
		if data, ok := resultObj["data"].(map[string]interface{}); ok {
			// Extract object details
			if version, ok := data["version"].(float64); ok {
				state.Version = fmt.Sprintf("%d", int64(version))
			}
			
			if objType, ok := data["type"].(string); ok {
				state.Type = objType
			}
			
			if digest, ok := data["digest"].(string); ok {
				state.Digest = digest
			}
			
			// Extract owner information
			if owner, ok := data["owner"].(map[string]interface{}); ok {
				state.Owner = owner
			}
			
			// Extract previous transaction
			if prevTx, ok := data["previousTransaction"].(string); ok {
				state.PreviousTx = prevTx
				
				// Get timestamp from previous transaction
				txData, err := GetTransactionTimestamp(prevTx)
				if err == nil && txData > 0 {
					state.Timestamp = txData
				}
			}
			
			// Extract content
			if content, ok := data["content"].(map[string]interface{}); ok {
				state.Content = content
			}
		}
	}
	
	return state, nil
}

// Get transaction timestamp
func GetTransactionTimestamp(txDigest string) (int64, error) {
	result, err := MakeRPCCall("sui_getTransactionBlock", []interface{}{
		txDigest,
		map[string]interface{}{
			"showEffects": true,
			"showInput": false,
			"showEvents": false,
			"showObjectChanges": false,
			"showBalanceChanges": false,
		},
	})
	
	if err != nil {
		return 0, err
	}
	
	if resultObj, ok := result["result"].(map[string]interface{}); ok {
		if timestampMs, ok := resultObj["timestamp_ms"].(string); ok {
			timestamp, err := strconv.ParseInt(timestampMs, 10, 64)
			if err == nil {
				return timestamp, nil
			}
		}
	}
	
	return 0, fmt.Errorf("timestamp not found in transaction %s", txDigest)
}

// Fetch entire object history
func FetchObjectHistory(objectID string) (*ObjectHistory, error) {
	history := &ObjectHistory{
		ID:     objectID,
		States: []ObjectState{},
	}
	
	// First, get current state
	currentState, err := GetObjectCurrentState(objectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get current object state: %v", err)
	}
	
	// Add current state to history
	history.States = append(history.States, *currentState)
	
	// Get all transactions for this object
	txDigests, err := GetAllObjectTransactions(objectID)
	if err != nil {
		fmt.Printf("Warning: Failed to get all transactions: %v\n", err)
		// Continue with just the current state
	} else {
		DebugPrint("Found %d transactions for object", len(txDigests))
		
		// Get object state from each transaction
		for _, txDigest := range txDigests {
			// Skip if this is the transaction we already have
			if txDigest == currentState.PreviousTx {
				continue
			}
			
			state, err := GetObjectDetailsFromTransaction(txDigest, objectID)
			if err != nil {
				DebugPrint("Warning: Failed to get object details from tx %s: %v", txDigest, err)
				continue
			}
			
			// Add to history
			history.States = append(history.States, *state)
		}
	}
	
	// Sort states by version
	sort.Slice(history.States, func(i, j int) bool {
		vI, _ := strconv.ParseUint(history.States[i].Version, 10, 64)
		vJ, _ := strconv.ParseUint(history.States[j].Version, 10, 64)
		return vI < vJ
	})
	
	// Calculate statistics
	if len(history.States) > 0 {
		history.NumChanges = len(history.States) - 1
		
		// Track unique owners
		uniqueOwners := make(map[string]bool)
		
		// Find first and last seen timestamps
		var minTimestamp int64 = 9223372036854775807 // Max int64
		var maxTimestamp int64 = 0
		
		for _, state := range history.States {
			// Track unique owners
			ownerKey := GetOwnerKey(state.Owner)
			uniqueOwners[ownerKey] = true
			
			// Track timestamps
			if state.Timestamp > 0 {
				if state.Timestamp < minTimestamp {
					minTimestamp = state.Timestamp
				}
				if state.Timestamp > maxTimestamp {
					maxTimestamp = state.Timestamp
				}
			}
		}
		
		history.NumOwners = len(uniqueOwners)
		
		if minTimestamp < 9223372036854775807 {
			history.FirstSeen = minTimestamp
		}
		if maxTimestamp > 0 {
			history.LastSeen = maxTimestamp
		}
	}
	
	return history, nil
}

// Helper function to create a unique key for an owner
func GetOwnerKey(owner map[string]interface{}) string {
	if owner == nil {
		return "unknown"
	}
	
	// Convert owner to a unique string representation
	ownerBytes, err := json.Marshal(owner)
	if err != nil {
		return "error"
	}
	
	return string(ownerBytes)
}

// Save object history to JSON file
func SaveObjectHistoryToJSON(history *ObjectHistory, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create JSON file: %v", err)
	}
	defer file.Close()
	
	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal history data: %v", err)
	}
	
	_, err = file.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write JSON data: %v", err)
	}
	
	return nil
}

// Print a summary of the object history
func PrintObjectSummary(history *ObjectHistory) {
	fmt.Printf("Object ID: %s\n", history.ID)
	fmt.Printf("Number of versions: %d\n", len(history.States))
	fmt.Printf("Number of changes: %d\n", history.NumChanges)
	fmt.Printf("Number of owners: %d\n", history.NumOwners)
	
	if history.FirstSeen > 0 {
		firstSeen := time.Unix(history.FirstSeen/1000, 0)
		fmt.Printf("First seen: %s\n", firstSeen.Format(time.RFC3339))
	}
	
	if history.LastSeen > 0 {
		lastSeen := time.Unix(history.LastSeen/1000, 0)
		fmt.Printf("Last seen: %s\n", lastSeen.Format(time.RFC3339))
	}
	
	if len(history.States) > 0 {
		fmt.Printf("Current type: %s\n", history.States[len(history.States)-1].Type)
	}
	
	fmt.Println("Version history:")
	for i, state := range history.States {
		timestamp := "unknown"
		if state.Timestamp > 0 {
			t := time.Unix(state.Timestamp/1000, 0)
			timestamp = t.Format(time.RFC3339)
		}
		fmt.Printf("  %d. Version %s - %s\n", i+1, state.Version, timestamp)
	}
}

func main() {
	objectID := flag.String("object", "", "Object ID to track")
	outputFile := flag.String("output", "", "Output JSON file (optional)")
	verbose := flag.Bool("verbose", false, "Print detailed information")
	debug := flag.Bool("debug", false, "Enable debug mode for API responses")
	flag.Parse()
	
	debugMode = *debug
	
	if *objectID == "" {
		fmt.Println("Error: Object ID is required")
		flag.Usage()
		return
	}
	
	startTime := time.Now()
	fmt.Printf("Fetching history for object: %s\n", *objectID)
	
	history, err := FetchObjectHistory(*objectID)
	if err != nil {
		log.Fatalf("Failed to fetch object history: %v", err)
	}
	
	elapsedTime := time.Since(startTime)
	
	if len(history.States) == 0 {
		fmt.Println("No object history found!")
		return
	}
	
	fmt.Printf("Fetched %d versions in %s\n", len(history.States), elapsedTime)
	
	// Print summary
	PrintObjectSummary(history)
	
	// Save to JSON if output file is specified
	if *outputFile != "" {
		fmt.Printf("Saving history to JSON file: %s\n", *outputFile)
		if err := SaveObjectHistoryToJSON(history, *outputFile); err != nil {
			log.Fatalf("Failed to save history to JSON: %v", err)
		}
		fmt.Printf("History saved successfully to %s\n", *outputFile)
	}
	
	if *verbose && len(history.States) > 0 {
		fmt.Println("\nDetailed state information:")
		for i, state := range history.States {
			fmt.Printf("\nState %d (Version %s):\n", i+1, state.Version)
			fmt.Printf("  Digest: %s\n", state.Digest)
			fmt.Printf("  Type: %s\n", state.Type)
			fmt.Printf("  Previous Transaction: %s\n", state.PreviousTx)
			
			// Print owner details
			if state.Owner != nil {
				ownerBytes, _ := json.MarshalIndent(state.Owner, "  ", "  ")
				fmt.Printf("  Owner: %s\n", string(ownerBytes))
			}
		}
	}
}