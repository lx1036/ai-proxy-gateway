package message

import "github.com/telepresenceio/watchable"

/**
目前还没精力调研 watchable，先复制粘贴过来临时用用
 */

type Update[K comparable, V any] watchable.Update[K, V]


func HandleSubscription[K comparable, V any](
	subscription <-chan watchable.Snapshot[K, V],
	handle func(updateFunc watchable.Update[K, V]),
) {
	if snapshot, ok := <-subscription; ok {
		for k, v := range snapshot.State {
			handle(watchable.Update[K, V]{
				Key:   k,
				Value: v,
			})
		}
	}

	for snapshot := range subscription {
		for _, update := range snapshot.Updates {
			handle(update)
		}
	}
}
