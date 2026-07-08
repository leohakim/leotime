package backup

import (
	"fmt"
	"sort"
	"time"

	"github.com/leotime/leotime/apps/api/internal/backup/storage"
)

func sortStorageObjectsDesc(objects []storage.Object) {
	sort.Slice(objects, func(i, j int) bool {
		return objects[i].LastModified.After(objects[j].LastModified)
	})
}

func sortObjectResponsesDesc(objects []ObjectResponse) {
	sort.Slice(objects, func(i, j int) bool {
		left, errLeft := time.Parse(time.RFC3339, objects[i].LastModified)
		right, errRight := time.Parse(time.RFC3339, objects[j].LastModified)
		if errLeft != nil || errRight != nil {
			return objects[i].Key > objects[j].Key
		}
		return left.After(right)
	})
}

func wrapRemoteError(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %v", ErrRemoteStorage, err)
}
