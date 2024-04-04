package main

import (
	"bytes"
	"io"
	"log"
	"net/http"
)

func handleNewTnx(body io.Reader) error {
	var url = "http://127.0.0.1:54321/functions/v1/trade-bot_v2"
	//var prodUrl = "https://fwozxyxqirrokxjxckob.supabase.co/functions/v1/trade-bot-v2"
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	sb := string(responseBody)
	log.Printf(sb)

	return nil
}

func handleTransactions(data []byte) error {
	postBody := data
	responseBody := bytes.NewBuffer(postBody)

	// Send a request for the current transaction
	err := handleNewTnx(responseBody)
	if err != nil {
		return err
	}

	return nil
}

//func insertDemo(db *badger.DB) {
//	var txData TransactionData
//	err := json.NewDecoder(bytes.NewReader([]byte(getJson()))).Decode(&txData)
//	if err != nil {
//		log.Fatalf("Error decoding JSON: %s", err)
//	}
//	// Convert the transaction data to JSON
//	txDataJSON, err := json.Marshal(txData)
//	if err != nil {
//		log.Fatalf("Error encoding JSON: %s", err)
//	}
//
//	// Start a write transaction
//	txn := db.NewTransaction(true)
//	defer txn.Discard()
//
//	// Insert the transaction data into the BadgerDB database
//	err = txn.Set([]byte("transaction:"+txData.TransactionId), txDataJSON)
//	if err != nil {
//		log.Fatalf("Error inserting data into BadgerDB: %s", err)
//	}
//
//	// Commit the transaction
//	err = txn.Commit()
//	if err != nil {
//		log.Fatalf("Error committing transaction: %s", err)
//	}
//
//}
