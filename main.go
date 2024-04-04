package main

import (
	_ "context"
	"encoding/json"
	"fmt"
	"github.com/dgraph-io/badger/v4"
	"log"
	"reflect"
	_ "time"
)

type DBPrefixes struct {
	// The key prefixes for the key-value database. To store a particular
	// type of data, we create a key prefix and store all those types of
	// data with a key prefixed by that key prefix.
	// Bitcoin does a similar thing that you can see at this link:
	// https://bitcoin.stackexchange.com/questions/28168/what-are-the-keys-used-in-the-blockchain-leveldb-ie-what-are-the-keyvalue-pair

	// The prefix for the block index:
	// Key format: <prefix_id, hash BlockHash>
	// Value format: serialized MsgDeSoBlock
	PrefixBlockHashToBlock []byte `prefix_id:"[0]" core_state:"true"`

	// The prefix for the node index that we use to reconstruct the block tree.
	// Storing the height in big-endian byte order allows us to read in all the
	// blocks in height-sorted order from the db and construct the block tree by connecting
	// nodes to their parents as we go.
	//
	// Key format: <prefix_id, height uint32 (big-endian), hash BlockHash>
	// Value format: serialized BlockNode
	PrefixHeightHashToNodeInfo        []byte `prefix_id:"[1]"`
	PrefixBitcoinHeightHashToNodeInfo []byte `prefix_id:"[2]"`

	// We store the hash of the node that is the current tip of the main chain.
	// This key is used to look it up.
	// Value format: BlockHash
	PrefixBestDeSoBlockHash []byte `prefix_id:"[3]"`

	PrefixBestBitcoinHeaderHash []byte `prefix_id:"[4]"`

	// Utxo table.
	// <prefix_id, txid BlockHash, output_index uint64> -> UtxoEntry
	PrefixUtxoKeyToUtxoEntry []byte `prefix_id:"[5]" is_state:"true"`
	// <prefix_id, pubKey [33]byte, utxoKey< txid BlockHash, index uint32 >> -> <>
	PrefixPubKeyUtxoKey []byte `prefix_id:"[7]" is_state:"true"`
	// The number of utxo entries in the database.
	PrefixUtxoNumEntries []byte `prefix_id:"[8]" is_state:"true"`
	// Utxo operations table.
	// This table contains, for each blockhash on the main chain, the UtxoOperations
	// that were applied by this block. To roll back the block, one must loop through
	// the UtxoOperations for a particular block backwards and invert them.
	//
	// <prefix_id, hash *BlockHash > -> < serialized [][]UtxoOperation using custom encoding >
	PrefixBlockHashToUtxoOperations []byte `prefix_id:"[9]" core_state:"true"`
	// The below are mappings related to the validation of BitcoinExchange transactions.
	//
	// The number of nanos that has been purchased thus far.
	PrefixNanosPurchased []byte `prefix_id:"[10]" is_state:"true"`
	// How much Bitcoin is work in USD cents.
	PrefixUSDCentsPerBitcoinExchangeRate []byte `prefix_id:"[27]" is_state:"true"`
	// <prefix_id, key> -> <GlobalParamsEntry encoded>
	PrefixGlobalParams []byte `prefix_id:"[40]" is_state:"true"`

	// The prefix for the Bitcoin TxID map. If a key is set for a TxID that means this
	// particular TxID has been processed as part of a BitcoinExchange transaction. If
	// no key is set for a TxID that means it has not been processed (and thus it can be
	// used to create new nanos).
	// <prefix_id, BitcoinTxID BlockHash> -> <nothing>
	PrefixBitcoinBurnTxIDs []byte `prefix_id:"[11]" is_state:"true"`
	// Messages are indexed by the public key of their senders and receivers. If
	// a message sends from pkFrom to pkTo then there will be two separate entries,
	// one for pkFrom and one for pkTo. The exact format is as follows:
	// <public key (33 bytes) || uint64 big-endian> -> <MessageEntry>
	PrefixPublicKeyTimestampToPrivateMessage []byte `prefix_id:"[12]" is_state:"true" core_state:"true"`

	// Tracks the tip of the transaction index. This is used to determine
	// which blocks need to be processed in order to update the index.
	PrefixTransactionIndexTip []byte `prefix_id:"[14]" is_txindex:"true"`
	// <prefix_id, transactionID BlockHash> -> <TransactionMetadata struct>
	PrefixTransactionIDToMetadata []byte `prefix_id:"[15]" is_txindex:"true"`
	// <prefix_id, publicKey []byte, index uint32> -> <txid BlockHash>
	PrefixPublicKeyIndexToTransactionIDs []byte `prefix_id:"[16]" is_txindex:"true"`
	// <prefix_id, publicKey []byte> -> <index uint32>
	PrefixPublicKeyToNextIndex []byte `prefix_id:"[42]" is_txindex:"true"`

	// Main post index.
	// <prefix_id, PostHash BlockHash> -> PostEntry
	PrefixPostHashToPostEntry []byte `prefix_id:"[17]" is_state:"true" core_state:"true"`
	// Post sorts
	// <prefix_id, publicKey [33]byte, PostHash> -> <>
	PrefixPosterPublicKeyPostHash []byte `prefix_id:"[18]" is_state:"true"`

	// <prefix_id, tstampNanos uint64, PostHash> -> <>
	PrefixTstampNanosPostHash []byte `prefix_id:"[19]" is_state:"true"`
	// <prefix_id, creatorbps uint64, PostHash> -> <>
	PrefixCreatorBpsPostHash []byte `prefix_id:"[20]" is_state:"true"`
	// <prefix_id, multiplebps uint64, PostHash> -> <>
	PrefixMultipleBpsPostHash []byte `prefix_id:"[21]" is_state:"true"`
	// Comments are just posts that have their ParentStakeID set, and
	// so we have a separate index that allows us to return all the
	// comments for a given StakeID
	// <prefix_id, parent stakeID [33]byte, tstampnanos uint64, post hash> -> <>
	PrefixCommentParentStakeIDToPostHash []byte `prefix_id:"[22]" is_state:"true"`

	// Main profile index
	// <prefix_id, PKID [33]byte> -> ProfileEntry
	PrefixPKIDToProfileEntry []byte `prefix_id:"[23]" is_state:"true" core_state:"true"`
	// Profile sorts
	// For username, we set the PKID as a value since the username is not fixed width.
	// We always lowercase usernames when using them as map keys in order to make
	// all uniqueness checks case-insensitive
	// <prefix_id, username> -> <PKID>
	PrefixProfileUsernameToPKID []byte `prefix_id:"[25]" is_state:"true"`
	// This allows us to sort the profiles by the value of their coin (since
	// the amount of DeSo locked in a profile is proportional to coin price).
	PrefixCreatorDeSoLockedNanosCreatorPKID []byte `prefix_id:"[32]" is_state:"true"`
	// The StakeID is a post hash for posts and a public key for users.
	// <prefix_id, StakeIDType, AmountNanos uint64, StakeID [var]byte> -> <>
	PrefixStakeIDTypeAmountStakeIDIndex []byte `prefix_id:"[26]" is_state:"true"`

	// Prefixes for follows:
	// <prefix_id, follower PKID [33]byte, followed PKID [33]byte> -> <>
	// <prefix_id, followed PKID [33]byte, follower PKID [33]byte> -> <>
	PrefixFollowerPKIDToFollowedPKID []byte `prefix_id:"[28]" is_state:"true" core_state:"true"`
	PrefixFollowedPKIDToFollowerPKID []byte `prefix_id:"[29]" is_state:"true"`

	// Prefixes for likes:
	// <prefix_id, user pub key [33]byte, liked post hash [32]byte> -> <>
	// <prefix_id, post hash [32]byte, user pub key [33]byte> -> <>
	PrefixLikerPubKeyToLikedPostHash []byte `prefix_id:"[30]" is_state:"true" core_state:"true"`
	PrefixLikedPostHashToLikerPubKey []byte `prefix_id:"[31]" is_state:"true"`

	// Prefixes for creator coin fields:
	// <prefix_id, HODLer PKID [33]byte, creator PKID [33]byte> -> <BalanceEntry>
	// <prefix_id, creator PKID [33]byte, HODLer PKID [33]byte> -> <BalanceEntry>
	PrefixHODLerPKIDCreatorPKIDToBalanceEntry []byte `prefix_id:"[33]" is_state:"true"`
	PrefixCreatorPKIDHODLerPKIDToBalanceEntry []byte `prefix_id:"[34]" is_state:"true" core_state:"true"`

	PrefixPosterPublicKeyTimestampPostHash []byte `prefix_id:"[35]" is_state:"true"`
	// If no mapping exists for a particular public key, then the PKID is simply
	// the public key itself.
	// <prefix_id, [33]byte> -> <PKID [33]byte>
	PrefixPublicKeyToPKID []byte `prefix_id:"[36]" is_state:"true" core_state:"true"`
	// <prefix_id, PKID [33]byte> -> <PublicKey [33]byte>
	PrefixPKIDToPublicKey []byte `prefix_id:"[37]" is_state:"true"`
	// Prefix for storing mempool transactions in badger. These stored transactions are
	// used to restore the state of a node after it is shutdown.
	// <prefix_id, tx hash BlockHash> -> <*MsgDeSoTxn>
	PrefixMempoolTxnHashToMsgDeSoTxn []byte `prefix_id:"[38]"`

	// Prefixes for Reposts:
	// <prefix_id, user pub key [39]byte, reposted post hash [39]byte> -> RepostEntry
	PrefixReposterPubKeyRepostedPostHashToRepostPostHash []byte `prefix_id:"[39]" is_state:"true"`
	// Prefixes for diamonds:
	//  <prefix_id, DiamondReceiverPKID [33]byte, DiamondSenderPKID [33]byte, posthash> -> <DiamondEntry>
	//  <prefix_id, DiamondSenderPKID [33]byte, DiamondReceiverPKID [33]byte, posthash> -> <DiamondEntry>
	PrefixDiamondReceiverPKIDDiamondSenderPKIDPostHash []byte `prefix_id:"[41]" is_state:"true"`
	PrefixDiamondSenderPKIDDiamondReceiverPKIDPostHash []byte `prefix_id:"[43]" is_state:"true" core_state:"true"`
	// Public keys that have been restricted from signing blocks.
	// <prefix_id, ForbiddenPublicKey [33]byte> -> <>
	PrefixForbiddenBlockSignaturePubKeys []byte `prefix_id:"[44]" is_state:"true"`

	// These indexes are used in order to fetch the pub keys of users that liked or diamonded a post.
	// 		Reposts: <prefix_id, RepostedPostHash, ReposterPubKey> -> <>
	// 		Quote Reposts: <prefix_id, RepostedPostHash, ReposterPubKey, RepostPostHash> -> <>
	// 		Diamonds: <prefix_id, DiamondedPostHash, DiamonderPubKey [33]byte, DiamondLevel (uint64)> -> <>
	PrefixRepostedPostHashReposterPubKey               []byte `prefix_id:"[45]" is_state:"true"`
	PrefixRepostedPostHashReposterPubKeyRepostPostHash []byte `prefix_id:"[46]" is_state:"true"`
	PrefixDiamondedPostHashDiamonderPKIDDiamondLevel   []byte `prefix_id:"[47]" is_state:"true"`
	// Prefixes for NFT ownership:
	// 	<prefix_id, NFTPostHash [32]byte, SerialNumber uint64> -> NFTEntry
	PrefixPostHashSerialNumberToNFTEntry []byte `prefix_id:"[48]" is_state:"true" core_state:"true"`
	//  <prefix_id, PKID [33]byte, IsForSale bool, BidAmountNanos uint64, NFTPostHash[32]byte, SerialNumber uint64> -> NFTEntry
	PrefixPKIDIsForSaleBidAmountNanosPostHashSerialNumberToNFTEntry []byte `prefix_id:"[49]" is_state:"true"`
	// Prefixes for NFT bids:
	//  <prefix_id, NFTPostHash [32]byte, SerialNumber uint64, BidNanos uint64, PKID [33]byte> -> <>
	PrefixPostHashSerialNumberBidNanosBidderPKID []byte `prefix_id:"[50]" is_state:"true" core_state:"true"`
	//  <prefix_id, BidderPKID [33]byte, NFTPostHash [32]byte, SerialNumber uint64> -> <BidNanos uint64>
	PrefixBidderPKIDPostHashSerialNumberToBidNanos []byte `prefix_id:"[51]" is_state:"true"`

	// <prefix_id, PublicKey [33]byte> -> uint64
	PrefixPublicKeyToDeSoBalanceNanos []byte `prefix_id:"[52]" is_state:"true" core_state:"true"`

	// Block reward prefix:
	//   - This index is needed because block rewards take N blocks to mature, which means we need
	//     a way to deduct them from balance calculations until that point. Without this index, it
	//     would be impossible to figure out which of a user's UTXOs have yet to mature.
	//   - Schema: <prefix_id, hash BlockHash> -> <pubKey [33]byte, uint64 blockRewardNanos>
	PrefixPublicKeyBlockHashToBlockReward []byte `prefix_id:"[53]" is_state:"true"`

	// Prefix for NFT accepted bid entries:
	//   - Note: this index uses a slice to track the history of winning bids for an NFT. It is
	//     not core to consensus and should not be relied upon as it could get inefficient.
	//   - Schema: <prefix_id>, NFTPostHash [32]byte, SerialNumber uint64 -> []NFTBidEntry
	PrefixPostHashSerialNumberToAcceptedBidEntries []byte `prefix_id:"[54]" is_state:"true"`

	// Prefixes for DAO coin fields:
	// <prefix, HODLer PKID [33]byte, creator PKID [33]byte> -> <BalanceEntry>
	// <prefix, creator PKID [33]byte, HODLer PKID [33]byte> -> <BalanceEntry>
	PrefixHODLerPKIDCreatorPKIDToDAOCoinBalanceEntry []byte `prefix_id:"[55]" is_state:"true" core_state:"true"`
	PrefixCreatorPKIDHODLerPKIDToDAOCoinBalanceEntry []byte `prefix_id:"[56]" is_state:"true"`

	// Prefix for MessagingGroupEntries indexed by OwnerPublicKey and GroupKeyName:
	//
	// * This index is used to store information about messaging groups. A group is indexed
	//   by the "owner" public key of the user who created the group and the key
	//   name the owner selected when creating the group (can be anything, user-defined).
	//
	// * Groups can have members that all use a shared key to communicate. In this case,
	//   the MessagingGroupEntry will contain the metadata required for each participant to
	//   compute the shared key.
	//
	// * Groups can also consist of a single person, and this is useful for "registering"
	//   a key so that other people can message you. Generally, every user has a mapping of
	//   the form:
	//   - <OwnerPublicKey, "default-key"> -> MessagingGroupEntry
	//   This "singleton" group is used to register a default key so that people can
	//   message this user. Allowing users to register default keys on-chain in this way is required
	//   to make it so that messages can be decrypted on mobile devices, where apps do not have
	//   easy access to the owner key for decrypting messages.
	//
	// <prefix, AccessGroupOwnerPublicKey [33]byte, GroupKeyName [32]byte> -> <MessagingGroupEntry>
	PrefixMessagingGroupEntriesByOwnerPubKeyAndGroupKeyName []byte `prefix_id:"[57]" is_state:"true"`

	// Prefix for Message MessagingGroupMembers:
	//
	// * For each group that a user is a member of, we store a value in this index of
	//   the form:
	//   - <OwnerPublicKey for user, GroupMessagingPublicKey> -> <HackedMessagingGroupEntry>
	//   The value needs to contain enough information to allow us to look up the
	//   group's metatdata in the _PrefixMessagingGroupEntriesByOwnerPubKeyAndGroupKeyName index. It's also convenient for
	//   the value to contain the encrypted messaging key for the user so that we can
	//   decrypt messages for this user *without* looking up the group.
	//
	// * HackedMessagingGroupEntry is a MessagingGroupEntry that we overload to store
	// 	 information on a member of a group. We couldn't use the MessagingGroupMember
	//   because we wanted to store additional information that "back-references" the
	//   MessagingGroupEntry for this group.
	//
	// * Note that GroupMessagingPublicKey != AccessGroupOwnerPublicKey. For this index
	//   it was convenient for various reasons to put the messaging public key into
	//   the index rather than the group owner's public key. This becomes clear if
	//   you read all the fetching code around this index.
	//
	// <prefix, OwnerPublicKey [33]byte, GroupMessagingPublicKey [33]byte> -> <HackedMessagingKeyEntry>
	PrefixMessagingGroupMetadataByMemberPubKeyAndGroupMessagingPubKey []byte `prefix_id:"[58]" is_state:"true"`

	// Prefix for Authorize Derived Key transactions:
	// 		<prefix_id, OwnerPublicKey [33]byte, DerivedPublicKey [33]byte> -> <DerivedKeyEntry>
	PrefixAuthorizeDerivedKey []byte `prefix_id:"[59]" is_state:"true" core_state:"true"`

	// Prefixes for DAO coin limit orders
	// This index powers the order book.
	// <
	//   _PrefixDAOCoinLimitOrder
	//   BuyingDAOCoinCreatorPKID [33]byte
	//   SellingDAOCoinCreatorPKID [33]byte
	//   ScaledExchangeRateCoinsToSellPerCoinToBuy [32]byte
	//   BlockHeight [32]byte
	//   OrderID [32]byte
	// > -> <DAOCoinLimitOrderEntry>
	//
	// This index allows users to query for their open orders.
	// <
	//   _PrefixDAOCoinLimitOrderByTransactorPKID
	//   TransactorPKID [33]byte
	//   BuyingDAOCoinCreatorPKID [33]byte
	//   SellingDAOCoinCreatorPKID [33]byte
	//   OrderID [32]byte
	// > -> <DAOCoinLimitOrderEntry>
	//
	// This index allows users to query for a single order by ID.
	// This is useful in e.g. cancelling an order.
	// <
	//   _PrefixDAOCoinLimitOrderByOrderID
	//   OrderID [32]byte
	// > -> <DAOCoinLimitOrderEntry>
	PrefixDAOCoinLimitOrder                 []byte `prefix_id:"[60]" is_state:"true" core_state:"true"`
	PrefixDAOCoinLimitOrderByTransactorPKID []byte `prefix_id:"[61]" is_state:"true"`
	PrefixDAOCoinLimitOrderByOrderID        []byte `prefix_id:"[62]" is_state:"true"`

	// User Association prefixes
	// PrefixUserAssociationByID:
	//  <
	//   PrefixUserAssociationByID
	//   AssociationID [32]byte
	//  > -> < UserAssociationEntry >
	PrefixUserAssociationByID []byte `prefix_id:"[63]" is_state:"true" core_state:"true"`
	// PrefixUserAssociationByTransactor:
	//  <
	//   PrefixUserAssociationByTransactor
	//   TransactorPKID [33]byte
	//   AssociationType + NULL TERMINATOR byte
	//   AssociationValue + NULL TERMINATOR byte
	//   TargetUserPKID [33]byte
	//   AppPKID [33]byte
	//  > -> < AssociationID > # note: AssociationID is a BlockHash type
	PrefixUserAssociationByTransactor []byte `prefix_id:"[64]" is_state:"true"`
	// PrefixUserAssociationByUsers:
	//  <
	//   PrefixUserAssociationByUsers
	//   TransactorPKID [33]byte
	//   TargetUserPKID [33]byte
	//   AssociationType + NULL TERMINATOR byte
	//   AssociationValue + NULL TERMINATOR byte
	//   AppPKID [33]byte
	//  > -> < AssociationID > # note: AssociationID is a BlockHash type
	PrefixUserAssociationByTargetUser []byte `prefix_id:"[65]" is_state:"true"`
	// PrefixUserAssociationByTargetUser
	//  <
	//   PrefixUserAssociationByTargerUser
	//   TargetUserPKID [33]byte
	//   AssociationType + NULL TERMINATOR byte
	//   AssociationValue + NULL TERMINATOR byte
	//   TransactorPKID [33]byte
	//   AppPKID [33]byte
	//  > -> < AssociationID > # note: Association is a BlockHash type
	PrefixUserAssociationByUsers []byte `prefix_id:"[66]" is_state:"true"`

	// Post Association prefixes
	// PrefixPostAssociationByID
	//  <
	//   PrefixPostAssociationByID
	//   AssociationID [32]byte
	//  > -> < PostAssociationEntry >
	PrefixPostAssociationByID []byte `prefix_id:"[67]" is_state:"true" core_state:"true"`
	// PrefixPostAssociationByTransactor
	//  <
	//   PrefixPostAssociationByTransactor
	//   TransactorPKID [33]byte
	//   AssociationType + NULL TERMINATOR byte
	//   AssociationValue + NULL TERMINATOR byte
	//   PostHash [32]byte
	//   AppPKID [33]byte
	// > -> < AssociationID > # note: AssociationID is a BlockHash type
	PrefixPostAssociationByTransactor []byte `prefix_id:"[68]" is_state:"true"`
	// PrefixPostAssociationByPost
	//  <
	//   PostHash [32]byte
	//   AssociationType + NULL TERMINATOR byte
	//   AssociationValue + NULL TERMINATOR byte
	//   TransactorPKID [33]byte
	//   AppPKID [33]byte
	//  > -> < AssociationID > # note: AssociationID is a BlockHash type
	PrefixPostAssociationByPost []byte `prefix_id:"[69]" is_state:"true"`
	// PrefixPostAssociationByType
	//  <
	//   AssociationType + NULL TERMINATOR byte
	//   AssociationValue + NULL TERMINATOR byte
	//   PostHash [32]byte
	//   TransactorPKID [33]byte
	//   AppPKID [33]byte
	//  > -> < AssociationID > # note: AssociationID is a BlockHash type
	PrefixPostAssociationByType []byte `prefix_id:"[70]" is_state:"true"`

	// Prefix for MessagingGroupEntries indexed by AccessGroupOwnerPublicKey and GroupKeyName:
	//
	// * This index is used to store information about messaging groups. A group is indexed
	//   by the "owner" public key of the user who created the group and the key
	//   name the owner selected when creating the group (can be anything, user-defined).
	//
	// * Groups can have members that all use a shared key to communicate. In this case,
	//   the MessagingGroupEntry will contain the metadata required for each participant to
	//   compute the shared key.
	//
	// * Groups can also consist of a single person, and this is useful for "registering"
	//   a key so that other people can message you. Generally, every user has a default mapping of
	//   the form:
	//   - <AccessGroupOwnerPublicKey, "default-key"> -> AccessGroupEntry
	//   This "singleton" group is used to register a default key so that people can
	//   message this user in the form of traditional DMs. Allowing users to register default keys on-chain in this
	//   way is required to make it so that messages can be decrypted on mobile devices, where apps do not have
	//   easy access to the owner key for decrypting messages.
	//
	// <prefix, AccessGroupOwnerPublicKey [33]byte, GroupKeyName [32]byte> -> <AccessGroupEntry>
	PrefixAccessGroupEntriesByAccessGroupId []byte `prefix_id:"[71]" is_state:"true" core_state:"true"`

	// This prefix is used to store all mappings for access group members. The group owner has a
	// special-case mapping with <groupOwnerPk, groupOwnerPk, groupName> and then everybody else has
	// <memberPk, groupOwnerPk, groupName>. We don't need to store members in a group entry anymore since
	// we can just iterate over the members in the group membership index here. This saves us a lot of space
	// and makes it easier to add and remove members from groups.
	//
	// * Note that as mentioned above, there is a special case where AccessGroupMemberPublicKey == AccessGroupOwnerPublicKey.
	//   For this index it was convenient for various reasons to automatically save an entry
	//   with such a key in the db whenever a user registers a group. This becomes clear if
	//   you read all the fetching code around this index. Particularly functions containing
	//   the 'owner' keyword. This is not a bug, it's a feature because we might want an owner to be a member
	//   of their own group for various reasons:
	//   - To be able to read messages sent to the group if the group was created with a derived key.
	//   - To be able to fetch all groups that a user is a member of (including groups that
	//     they own). This is especially useful for allowing the Backend API to fetch all groups for a user.
	//
	// New <GroupMembershipIndex> :
	// <prefix, AccessGroupMemberPublicKey [33]byte, AccessGroupOwnerPublicKey [33]byte, GroupKeyName [32]byte> -> <AccessGroupMemberEntry>
	PrefixAccessGroupMembershipIndex []byte `prefix_id:"[72]" is_state:"true" core_state:"true"`

	// Prefix for enumerating all the members of a group. Note that the previous index allows us to
	// answer the question, "what groups is this person a member of?" while this index allows us to
	// answer "who are the members of this particular group?"
	// <prefix, AccessGroupOwnerPublicKey [33]byte, GroupKeyName [32]byte, AccessGroupMemberPublicKey [33]byte>
	//		-> <AccessGroupMemberEnumerationEntry>
	PrefixAccessGroupMemberEnumerationIndex []byte `prefix_id:"[73]" is_state:"true"`

	// PrefixGroupChatMessagesIndex is modified by the NewMessage transaction and is used to store group chat
	// NewMessageEntry objects for each message sent to a group chat. The index has the following structure:
	// 	<prefix, AccessGroupOwnerPublicKey, AccessGroupKeyName, TimestampNanos> -> <NewMessageEntry>
	PrefixGroupChatMessagesIndex []byte `prefix_id:"[74]" is_state:"true" core_state:"true"`

	// PrefixDmMessagesIndex is modified by the NewMessage transaction and is used to store NewMessageEntry objects for
	// each message sent to a Dm thread. It answers the question: "Give me all the messages between these two users."
	// The index has the following structure:
	// 	<prefix, MinorAccessGroupOwnerPublicKey, MinorAccessGroupKeyName,
	//		MajorAccessGroupOwnerPublicKey, MajorAccessGroupKeyName, TimestampNanos> -> <NewMessageEntry>
	// The Minor/Major distinction is used to deterministically map the two accessGroupIds of message's sender/recipient
	// into a single pair based on the lexicographical ordering of the two accessGroupIds. This is done to ensure that
	// both sides of the conversation have the same key for the same conversation, and we can store just a single message.
	PrefixDmMessagesIndex []byte `prefix_id:"[75]" is_state:"true"`

	// PrefixDmThreadIndex is modified by the NewMessage transaction and is used to store a DmThreadEntry
	// for each existing dm thread. It answers the question: "Give me all the threads for a particular user."
	// The index has the following structure:
	// 	<prefix, UserAccessGroupOwnerPublicKey, UserAccessGroupKeyName,
	//		PartyAccessGroupOwnerPublicKey, PartyAccessGroupKeyName> -> <DmThreadEntry>
	// It's worth noting that two of these entries are stored for each Dm thread, one being the inverse of the other.
	PrefixDmThreadIndex []byte `prefix_id:"[76]" is_state:"true"`

	// PrefixNoncePKIDIndex is used to track unexpired nonces. Each nonce is uniquely identified by its expiration block
	// height, the PKID of the user who created it, and a partial ID that is unique to the nonce. The partial ID is any
	// random uint64.
	// The index has the following structure:
	// 	<prefix, expirationBlockHeight, PKID, partialID> -> <>
	PrefixNoncePKIDIndex []byte `prefix_id:"[77]" is_state:"true"`

	// PrefixTxnHashToTxn is used to store transactions that have been processed by the node. This isn't actually stored
	// in badger, but is tracked by state syncer when processing mempool transactions.
	// The index has the following structure:
	// 	<prefix, txnHash> -> <Transaction>
	PrefixTxnHashToTxn []byte `prefix_id:"[78]" core_state:"true"`

	// PrefixTxnHashToUtxoOps is used to store UtxoOps for transactions that have been processed by the node.
	// This isn't actually stored in badger, but is tracked by state syncer when processing mempool transactions.
	PrefixTxnHashToUtxoOps []byte `prefix_id:"[79]" core_state:"true"`

	// NEXT_TAG: 80

}

func main() {
	// Create a new Badger DB instance.
	db, err := badger.Open(badger.DefaultOptions("\\\\wsl.localhost\\docker-desktop-data\\data\\docker\\volumes\\run_db\\_data\\v-00000\\badgerdb"))
	if err != nil {
		log.Fatalf("Error opening Badger database: %v", err)
	}
	defer func(db *badger.DB) {
		err := db.Close()
		if err != nil {

		}
	}(db)

	var newTnx = db.NewTransaction(false)
	prefixes := GetPrefixes()
	prefixElements := reflect.ValueOf(prefixes).Elem()
	structFields := prefixElements.Type()

	for i := 0; i < prefixElements.NumField(); i++ {
		field := prefixElements.Field(i)
		fieldName := structFields.Field(i).Name

		// if fieldName is not than continue "PrefixMempoolTxnHashToMsgDeSoTxn"
		if fieldName != "PrefixMempoolTxnHashToMsgDeSoTxn" {
			//log the prefix
			fmt.Printf("Field Name: %s, Prefix ID: %v\n", fieldName, field.Bytes())
			continue
		}
		fmt.Printf("Field Name: %s, Prefix ID: %v\n", fieldName, field.Bytes())
		prefixCopy := append([]byte{}, field.Bytes()...)

		txn, _, err := _enumerateKeysForPrefixWithTxn(newTnx, prefixCopy)
		if err != nil {
			return
		}

		for i, key := range txn {
			log.Printf("Key: %s, Value: %s\n", key, i)
		}

		// log the length of the transactions
		log.Printf("Length of transactions: %d\n", len(txn))
	}

}

func _enumerateKeysForPrefixWithTxn(txn *badger.Txn, dbPrefix []byte) (_keysFound [][]byte, _valsFound [][]byte, _err error) {
	var keysFound [][]byte
	var valsFound [][]byte

	opts := badger.DefaultIteratorOptions
	nodeIterator := txn.NewIterator(opts)
	defer nodeIterator.Close()
	prefix := dbPrefix
	for nodeIterator.Seek(prefix); nodeIterator.ValidForPrefix(prefix); nodeIterator.Next() {
		key := nodeIterator.Item().Key()
		keyCopy := make([]byte, len(key))
		copy(keyCopy[:], key[:])

		valCopy, err := nodeIterator.Item().ValueCopy(nil)
		if err != nil {
			return nil, nil, err
		}
		keysFound = append(keysFound, keyCopy)
		valsFound = append(valsFound, valCopy)
	}
	return keysFound, valsFound, nil
}

// GetPrefixes loads all prefix_id byte array values into a DBPrefixes struct, and returns it.
func GetPrefixes() *DBPrefixes {
	prefixes := &DBPrefixes{}

	// Iterate over all DBPrefixes fields and parse their prefix_id tags.
	prefixElements := reflect.ValueOf(prefixes).Elem()
	structFields := prefixElements.Type()
	for i := 0; i < structFields.NumField(); i++ {
		prefixField := prefixElements.Field(i)
		prefixId := getPrefixIdValue(structFields.Field(i), prefixField.Type())
		prefixField.Set(prefixId)
	}
	return prefixes
}

// getPrefixIdValue parses the DBPrefixes struct tags to fetch the prefix_id values.
func getPrefixIdValue(structFields reflect.StructField, fieldType reflect.Type) (prefixId reflect.Value) {
	var ref reflect.Value
	// Get the prefix_id tag and parse it as byte array.
	if value := structFields.Tag.Get("prefix_id"); value != "-" {
		ref = reflect.New(fieldType)
		ref.Elem().Set(reflect.MakeSlice(fieldType, 0, 0))
		if value != "" && value != "[]" {
			if err := json.Unmarshal([]byte(value), ref.Interface()); err != nil {
				panic(any(err))
			}
		}
	} else {
		panic(any(fmt.Errorf("prefix_id cannot be empty")))
	}
	return ref.Elem()
}
