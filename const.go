package main

func getJson() string {
	jsonData := `{
		"txnMeta": {
			"OperationType": 0,
			"DeSoToAddNanos": 100,
			"DeSoToSellNanos": 200,
			"ProfilePublicKey": "BC1YLfoQbdLBum4RKbrD5KP4S8SuhyTWKbfY8iic8PvQzv96QyQtuay",
			"MinDeSoExpectedNanos": 300,
			"CreatorCoinToSellNanos": 400,
			"MinCreatorCoinExpectedNanos": 500
		},
		"transactionId": "3JuETdm6pYSPEcnvZNSsh2HewPNe7d4dED5Szde1qU1uHutS5ZBUJ4",
		"txIndexMetadata": {
			"OperationType": "sell",
			"DeSoToAddNanos": 600,
			"DeSoToSellNanos": 700,
			"DESOLockedNanosDiff": 800,
			"CreatorCoinToSellNanos": 900
		},
		"affectedPublicKeys": {
			"nodes": [
				{"publicKey": "BC1YLfoQbdLBum4RKbrD5KP4S8SuhyTWKbfY8iic8PvQzv96QyQtuay"},
				{"publicKey": "BC1YLgwnEcYTFqJHVnDiXBTLRuqVcPfkVncve8hAF5YWtVaJYjfACvc"}
			]
		}
	}`

	return jsonData
}
