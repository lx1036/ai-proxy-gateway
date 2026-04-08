package framework

import (
	"context"
	"github.com/cespare/xxhash/v2"
)

type PrefixCachePlugin struct {
}

func (plugin *PrefixCachePlugin) Score(pods []Pod) map[Pod]float64 {

	hashes := hashPrompt(ctx, request)

	PrefixCacheServers = plugin.matchLongestPrefix(ctx, hashes)

	total := len(hashes)
	scores := make(map[Pod]float64, len(pods))
	for _, pod := range pods {
		if len(total) == 0 {
			scores[pod] = 0
			continue
		}

		matchLen := PrefixCacheServers[]
		scores[pod] = float64(matchLen) / float64(total)
	}

	return scores
}

func (plugin *PrefixCachePlugin) matchLongestPrefix(ctx context.Context, hashes []BlockHash) {
	for i := 0; i < len(hashes); i++ {
		hash := hashes[i]
		cachedServers := plugin.indexer.Get(hash)
		if len(cachedServers) == 0 {
			break
		} else {

		}
	}
}

type BlockHash uint64

// hashPrompt divides the prompt into blocks and calculate the prefix cache for each block.
// hash[0] is calculated including the model name and cache_salt(if provided), since different models generally don't share prefix cache.
// For block i, hash(i) = hash(block i content, hash(i-1)).
func hashPrompt(ctx context.Context, request *LLMRequest, cacheBlockSize int, maxPrefixBlocks int) []BlockHash {
	userInput, err := getUserInputBytes(request)

	if len(userInput) < cacheBlockSize {
		return nil
	}

	if len(userInput) > cacheBlockSize*maxPrefixBlocks {

		// truncate the input to the max prefix blocks
		userInput = userInput[:maxPrefixBlocks*cacheBlockSize]
	}

	// Split the body into blocks of size cacheBlockSize.
	// If the last block is smaller than cacheBlockSize, it will be ignored.
	res := make([]BlockHash, 0, len(userInput)/cacheBlockSize)

	// Add the model to the first block hash so that different models have different hashes even with the same body.
	h := xxhash.New()
	_, _ = h.Write([]byte(request.TargetModel))
	if cacheSalt := request.Body.CacheSalt(); cacheSalt != "" {
		_, _ = h.Write([]byte(cacheSalt))
	}

	prevBlockHash := BlockHash(h.Sum64())
	for i := 0; i+cacheBlockSize <= len(userInput); i += cacheBlockSize {
		h.Reset()
		_, _ = h.Write(userInput[i : i+cacheBlockSize])
		_, _ = h.Write(toBytes(prevBlockHash)) // cacheBlockSize + prevBlockHash
		res = append(res, BlockHash(h.Sum64()))
		prevBlockHash = res[len(res)-1]
	}

	return res
}

type Indexer struct {
}

func NewIndexer(ctx context.Context, defaultLRUSize int) *Indexer {

}
