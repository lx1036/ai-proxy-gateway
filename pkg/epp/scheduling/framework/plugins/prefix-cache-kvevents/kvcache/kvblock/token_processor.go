package kvblock

import (
	"fmt"
	"github.com/fxamacker/cbor/v2"
	"hash/fnv"
	"k8s.io/klog/v2"
)

// BlockHash struct represents a unique identifier for a KV-cache block.
type BlockHash uint64

// EmptyBlockHash represents an invalid or uninitialized block hash.
// This serves as the "error value".
const EmptyBlockHash BlockHash = 0

// defaultBlockSize is the default number of tokens per block.
// 16 is the default value used by vLLM.
const defaultBlockSize = 16

type TokenProcessorConfig struct {
	BlockSize int `json:"blockSize"`
	// HashSeed is used to prefix initial hash chunks, similarly to vLLM's NONE_HASH.
	// This should be aligned with vLLM's `PYTHONHASHSEED` environment variable.
	// The system's deployer is responsible for aligning the vLLM deployments
	// with the same seed value.
	HashSeed string `json:"hashSeed"`
	initHash uint64 // cache once
}

// ChunkedTokenDatabase is a concrete implementation of TokenDatabase.
// It mimics the chunkedTokenDatabase in the Python code.
type ChunkedTokenDatabase struct {
	TokenProcessorConfig
	encoder cbor.EncMode // cached CBOR encoder for interoperable encoding
}

func DefaultTokenProcessorConfig() *TokenProcessorConfig {
	return &TokenProcessorConfig{
		BlockSize: defaultBlockSize,
		HashSeed:  "",
	}
}

func NewChunkedTokenDatabase(config *TokenProcessorConfig) (*ChunkedTokenDatabase, error) {
	if config == nil {
		config = DefaultTokenProcessorConfig()
	}

	if config.BlockSize <= 0 {
		return nil, fmt.Errorf("blockSize must be greater than 0, got %d", config.BlockSize)
	}

	if config.initHash == 0 {
		// Create initial hash
		h := fnv.New64a()
		_, _ = h.Write([]byte(config.HashSeed))
		config.initHash = h.Sum64()
	}

	encoder, err := cbor.CanonicalEncOptions().EncMode()
	if err != nil {
		return nil, fmt.Errorf("failed to create CBOR encoder: %w", err)
	}

	return &ChunkedTokenDatabase{
		TokenProcessorConfig: *config,
		encoder:              encoder,
	}, nil
}

// MMHash represents a single multimodal content hash entry.
// This matches vLLM's per-block extra_keys format where each entry is
// the mm_feature.identifier string for an overlapping multimodal item.
type MMHash struct {
	Hash string
}

// BlockExtraFeatures holds per-block extra data that taints the block hash.
// A nil *BlockExtraFeatures means pure text (no taint).
type BlockExtraFeatures struct {
	MMHashes []MMHash
}

// TokensToKVBlockKeys converts tokens into kv_block.Keys.
func (db *ChunkedTokenDatabase) TokensToKVBlockKeys(parentKey BlockHash, tokens []uint32, modelName string, extraFeatures []*BlockExtraFeatures) ([]BlockHash, error) {
	var currentParentHash uint64
	if parentKey != EmptyBlockHash {
		currentParentHash = uint64(parentKey)
	} else {
		currentParentHash = db.hash(db.initHash, nil, modelName)
	}

	chunks := db.chunkTokens(tokens)
	if len(chunks) == 0 {
		return nil, nil
	}

	if extraFeatures == nil {
		extraFeatures = make([]*BlockExtraFeatures, len(chunks))
	} else if len(extraFeatures) != len(chunks) {
		return nil, fmt.Errorf("extraFeatures length does not match chunks length")
	}

	return db.prefixHashes(currentParentHash, chunks, extraFeatures), nil
}

// chunkTokens splits the input slice of tokens into chunks of size blockSize.
// /Users/lx1036/Code/k8s/LMCache/lmcache/v1/token_database.py:_chunk_tokens()
func (db *ChunkedTokenDatabase) chunkTokens(tokens []uint32) [][]uint32 {
	// 1. tokens []uint32 按照 BlockSize 分成 chunks[]
	var chunks [][]uint32
	for i := 0; i < len(tokens); i += db.TokenProcessorConfig.BlockSize {
		end := i + db.TokenProcessorConfig.BlockSize
		if end > len(tokens) {
			break // no partial blocks
		}

		chunks = append(chunks, tokens[i:end])
	}

	return chunks
}

func (db *ChunkedTokenDatabase) prefixHashes(parentHash uint64, chunks [][]uint32, extraFeatures []*BlockExtraFeatures) []BlockHash {
	hashes := make([]BlockHash, len(chunks))
	prefix := parentHash
	for i, chunk := range chunks {
		var extra interface{}
		if extraFeatures[i] != nil {
			extra = extraFeatures[i].MMHashes
		}

		prefix = db.hash(prefix, chunk, extra) // prefix hash 作为下一个 parent hash
		hashes[i] = BlockHash(prefix)
	}

	return hashes
}

// hash computes the uint64 FNV-64a hash of the given parent, tokens,
// and extra keys.
//
// The hash is computed using FNV-64a over the CBOR canonical encoding of
// [parent, tokens, extra], ensuring deterministic results across runs and
// compatibility with vLLM's prefix caching algorithm.
//
// The extra parameter enables cache differentiation for LoRA adapters and
// multi-modal content. Supported types: nil, int, string, map[string]interface{}.
// Must be CBOR-serializable.
func (db *ChunkedTokenDatabase) hash(parent uint64, tokens []uint32, extra interface{}) uint64 {
	payload := []interface{}{parent, tokens, extra}
	b, err := db.encoder.Marshal(payload)
	if err != nil {
		klog.Errorf("failed to marshal payload to CBOR: %v", err)
		return 0
	}

	h := fnv.New64a()
	_, _ = h.Write(b)
	return h.Sum64()
}
