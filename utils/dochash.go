package utils

type ActionType string

const (
	ActionTypeAdd    ActionType = "add"
	ActionTypeDelete ActionType = "delete"
	ActionTypeModify ActionType = "modify"
)

type DocHash struct {
	ID   string
	Type string
	Hash string
}

type HashDiff struct {
	Action          ActionType
	Id              string
	Type            string
	SourceHashValue string
	TargetHashValue string
}

func CompareMap(source, target map[string]*DocHash) [3][]HashDiff {
	var diffs [3][]HashDiff

	// Detect deleted and modified entries
	for id, sourceHash := range source {
		targetHash, exists := target[id]
		if !exists {
			diffs[0] = append(diffs[0], HashDiff{
				Action:          ActionTypeAdd,
				Id:              id,
				Type:            sourceHash.Type,
				SourceHashValue: sourceHash.Hash,
			})
		} else if sourceHash.Hash != targetHash.Hash {
			diffs[2] = append(diffs[2], HashDiff{
				Action:          ActionTypeModify,
				Id:              id,
				Type:            targetHash.Type,
				SourceHashValue: sourceHash.Hash,
				TargetHashValue: targetHash.Hash,
			})
		}
	}

	// Detect added entries
	for id, targetHash := range target {
		_, exists := source[id]
		if !exists {
			diffs[1] = append(diffs[1], HashDiff{
				Action:          ActionTypeDelete,
				Id:              id,
				Type:            targetHash.Type,
				TargetHashValue: targetHash.Hash,
			})
		}
	}

	return diffs
}
