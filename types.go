package main

type TransactionData struct {
	TxnMeta struct {
		OperationType               int64  `json:"OperationType"`
		DeSoToAddNanos              int64  `json:"DeSoToAddNanos"`
		DeSoToSellNanos             int64  `json:"DeSoToSellNanos"`
		ProfilePublicKey            string `json:"ProfilePublicKey"`
		MinDeSoExpectedNanos        int64  `json:"MinDeSoExpectedNanos"`
		CreatorCoinToSellNanos      int64  `json:"CreatorCoinToSellNanos"`
		MinCreatorCoinExpectedNanos int64  `json:"MinCreatorCoinExpectedNanos"`
	} `json:"txnMeta"`
	TransactionId   string `json:"transactionId"`
	TxIndexMetadata struct {
		OperationType          string `json:"OperationType"`
		DeSoToAddNanos         int64  `json:"DeSoToAddNanos"`
		DeSoToSellNanos        int64  `json:"DeSoToSellNanos"`
		DESOLockedNanosDiff    int64  `json:"DESOLockedNanosDiff"`
		CreatorCoinToSellNanos int64  `json:"CreatorCoinToSellNanos"`
	} `json:"txIndexMetadata"`
	AffectedPublicKeys struct {
		Nodes []struct {
			PublicKey string `json:"publicKey"`
		} `json:"nodes"`
	} `json:"affectedPublicKeys"`
}
